// Copyright 2025
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package audit

import (
	"bufio"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash"
	"io"
	"sort"
	"strings"
	"time"

	"golang.org/x/sync/errgroup"
)

// exportWindow exports a single (stream, tenant, hour) tuple.
//
// It:
//  1. Checks idempotency — if manifest.json already exists, the window is a no-op.
//  2. Streams events from VictoriaLogs through a gzip pipeline directly into S3
//     using multipart upload — no full compressed payload is held in memory.
//  3. Builds manifest.json (chaining to the previous manifest's SHA-256).
//  4. Signs manifest.json → manifest.json.sig.
//  5. Uploads data.jsonl.gz first, then manifest.json.sig, then manifest.json.
//     The manifest is uploaded last to signal a successfully completed window.
func (e *Exporter) exportWindow(ctx context.Context, w ExportWindow) error {
	log := e.log.WithValues(
		"stream", w.Stream,
		"tenant", w.Tenant,
		"window", w.Start.Format("2006-01-02T15:04Z"),
	)

	prefix := w.S3KeyPrefix(e.cfg.S3Prefix)
	manifestKey := prefix + "manifest.json"

	// ── Step 1: Idempotency check ──────────────────────────────────────────
	exists, err := e.s3.ObjectExists(ctx, manifestKey)
	if err != nil {
		return fmt.Errorf("idempotency check: %w", err)
	}
	if exists {
		log.Info("window already exported, skipping")
		return nil
	}

	// ── Step 2: Stream events VictoriaLogs → gzip → S3 ────────────────────
	log.Info("querying VictoriaLogs")
	body, err := e.vlogs.QueryEvents(ctx, w.Tenant, w.Start, w.End)
	if err != nil {
		return err // SourceUnavailableError or regular error
	}
	defer func() { _ = body.Close() }()

	dataKey := prefix + "data.jsonl.gz"

	// In compliance/WORM mode, a data file present without a manifest means a
	// prior run uploaded the data but was interrupted before the manifest was
	// written.  Because WORM prevents overwriting, we cannot safely resume
	// (the new data stream may differ).  Abort and require manual remediation.
	if e.cfg.ComplianceMode {
		dataExists, err := e.s3.ObjectExists(ctx, dataKey)
		if err != nil {
			return fmt.Errorf("compliance pre-check for data file: %w", err)
		}
		if dataExists {
			return fmt.Errorf(
				"compliance mode: data file %q exists without a manifest; "+
					"manual intervention is required to resolve this partial upload",
				dataKey,
			)
		}
	}

	stats, err := e.streamDataToS3(ctx, body, dataKey)
	if err != nil {
		return fmt.Errorf("stream data to S3: %w", err)
	}
	log.Info("streamed events", "count", stats.eventCount)

	// ── Step 3: Fetch previous manifest SHA-256 for chaining ──────────────
	prevSHA256, err := e.previousManifestSHA256(ctx, w)
	if err != nil {
		log.Info("could not fetch previous manifest for chaining, using empty string", "err", err)
		prevSHA256 = ""
	}

	// ── Step 4: Build manifest ─────────────────────────────────────────────
	manifest := Manifest{
		AuditPolicyVersions:        stats.auditPolicyVersions,
		ClusterAuditPolicyVersions: stats.clusterAuditPolicyVersions,
		CreatedAt:                  msTime{time.Now().UTC()},
		DataFile: DataFileMeta{
			Name:      "data.jsonl.gz",
			SHA256:    stats.sha256hex,
			SizeBytes: stats.sizeBytes,
		},
		EventCount:             stats.eventCount,
		EventSchemaVersion:     EventSchemaVersion,
		KMSKeyID:               e.signer.KeyID(),
		ManifestVersion:        ManifestVersion,
		PreviousManifestSHA256: prevSHA256,
		Producer:               e.cfg.producer(),
		SigningAlgorithm:       e.signer.Algorithm(),
		Stream:                 w.Stream,
		Tenant:                 w.Tenant,
		WindowEnd:              msTime{w.End},
		WindowStart:            msTime{w.Start},
	}

	manifestBytes, err := marshalCanonical(manifest)
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}

	// ── Step 5: Sign manifest ──────────────────────────────────────────────
	sigBytes, err := e.signer.Sign(ctx, manifestBytes)
	if err != nil {
		return fmt.Errorf("sign manifest: %w", err)
	}

	sigKey := prefix + "manifest.json.sig"

	// ── Step 6: Upload signature then manifest (manifest is completion signal) ──
	log.Info("uploading manifest signature", "key", sigKey)
	if err := e.s3.PutObject(ctx, sigKey, sigBytes, "text/plain"); err != nil {
		return fmt.Errorf("upload sig: %w", err)
	}

	log.Info("uploading manifest", "key", manifestKey)
	if err := e.s3.PutObject(ctx, manifestKey, manifestBytes, "application/json"); err != nil {
		return fmt.Errorf("upload manifest: %w", err)
	}

	log.Info("window exported successfully", "events", stats.eventCount, "size_bytes", stats.sizeBytes)
	return nil
}

// dataStats holds statistics collected while streaming events from VictoriaLogs.
type dataStats struct {
	eventCount                 int
	sizeBytes                  int64
	sha256hex                  string
	auditPolicyVersions        []string
	clusterAuditPolicyVersions []string
}

// streamDataToS3 reads JSONL from r, converts each line to canonical JSON,
// and pipes the gzip-compressed result directly into S3 via multipart upload
// — no full compressed payload is held in memory.
//
// SHA-256 and byte count are computed inline as bytes flow through the pipe.
// Policy versions are collected during the same pass and returned in dataStats.
//
// The compression goroutine and the S3 upload run concurrently: the goroutine
// writes to the write-end of an io.Pipe; UploadStream reads from the read-end.
// Errors from either side propagate cleanly via errgroup.
func (e *Exporter) streamDataToS3(ctx context.Context, r io.Reader, key string) (dataStats, error) {
	pr, pw := io.Pipe()

	var stats dataStats
	eg, ctx := errgroup.WithContext(ctx)

	// Compression goroutine — runs concurrently with the S3 upload.
	eg.Go(func() error {
		s, err := compressEvents(r, pw)
		// Always close the pipe so the upload goroutine unblocks.
		pw.CloseWithError(err) //nolint:errcheck // pipe close error is not actionable
		if err != nil {
			return err
		}
		stats = s
		return nil
	})

	// Upload goroutine — streams from the pipe reader directly into S3.
	eg.Go(func() error {
		if err := e.s3.UploadStream(ctx, key, pr, "application/gzip"); err != nil {
			pr.CloseWithError(err) //nolint:errcheck // pipe close error is not actionable
			return fmt.Errorf("upload stream: %w", err)
		}
		return nil
	})

	if err := eg.Wait(); err != nil {
		return dataStats{}, err
	}
	return stats, nil
}

// compressEvents reads JSONL from r one line at a time, converts each line to
// canonical JSON, and writes the gzip-compressed output to w.  SHA-256 and the
// compressed byte count are computed inline so the caller never needs to buffer
// the full payload.
func compressEvents(r io.Reader, w io.Writer) (dataStats, error) {
	h := sha256.New()
	cw := &countingWriter{w: io.MultiWriter(w, h)}
	gz, err := gzip.NewWriterLevel(cw, gzip.BestCompression)
	if err != nil {
		return dataStats{}, fmt.Errorf("create gzip writer: %w", err)
	}

	auditPolicies := map[string]struct{}{}
	clusterPolicies := map[string]struct{}{}
	var count int

	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var flat map[string]string
		if err := json.Unmarshal([]byte(line), &flat); err != nil {
			return dataStats{}, fmt.Errorf("line %d: parse: %w", lineNum, err)
		}
		ev, err := flatToAuditEvent(flat)
		if err != nil {
			return dataStats{}, fmt.Errorf("line %d: %w", lineNum, err)
		}

		if v := ev.AuditPolicyVersion; v != "" {
			auditPolicies[v] = struct{}{}
		}
		if v := ev.ClusterAuditPolicyVersion; v != "" {
			clusterPolicies[v] = struct{}{}
		}

		canonical, err := marshalCanonical(ev)
		if err != nil {
			return dataStats{}, fmt.Errorf("line %d: marshal canonical: %w", lineNum, err)
		}
		if _, err := gz.Write(canonical); err != nil {
			return dataStats{}, fmt.Errorf("line %d: write gzip: %w", lineNum, err)
		}
		if _, err := gz.Write([]byte("\n")); err != nil {
			return dataStats{}, fmt.Errorf("line %d: write newline: %w", lineNum, err)
		}
		count++
	}
	if err := scanner.Err(); err != nil {
		return dataStats{}, fmt.Errorf("reading events: %w", err)
	}
	if err := gz.Close(); err != nil {
		return dataStats{}, fmt.Errorf("close gzip writer: %w", err)
	}

	return dataStats{
		eventCount:                 count,
		sizeBytes:                  cw.n,
		sha256hex:                  hashHex(h),
		auditPolicyVersions:        sortedKeys(auditPolicies),
		clusterAuditPolicyVersions: sortedKeys(clusterPolicies),
	}, nil
}

// countingWriter wraps an io.Writer and counts the bytes written through it.
type countingWriter struct {
	w io.Writer
	n int64
}

func (c *countingWriter) Write(p []byte) (int, error) {
	n, err := c.w.Write(p)
	c.n += int64(n)
	return n, err
}

// hashHex returns the hex-encoded digest of h.
func hashHex(h hash.Hash) string {
	return hex.EncodeToString(h.Sum(nil))
}

// sortedKeys returns the keys of m as a sorted, non-nil slice.
func sortedKeys(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// previousManifestSHA256 fetches and hashes the manifest.json from the
// window immediately preceding w (same stream and tenant, one hour earlier).
func (e *Exporter) previousManifestSHA256(ctx context.Context, w ExportWindow) (string, error) {
	prev := ExportWindow{
		Stream: w.Stream,
		Tenant: w.Tenant,
		Start:  w.Start.Add(-time.Hour),
		End:    w.Start,
	}
	prevManifestKey := prev.S3KeyPrefix(e.cfg.S3Prefix) + "manifest.json"
	data, err := e.s3.GetObject(ctx, prevManifestKey)
	if err != nil {
		return "", err
	}
	if data == nil {
		return "", nil // no previous manifest — first export
	}
	// Verify the fetched data is valid JSON before hashing.
	if !json.Valid(data) {
		return "", fmt.Errorf("previous manifest is not valid JSON")
	}
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:]), nil
}
