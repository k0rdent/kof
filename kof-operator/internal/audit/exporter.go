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
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/go-logr/logr"
)

// vlogsQuerier is the subset of VLogsClient used by the Exporter.
// Extracted as an interface to allow unit-testing without a real VictoriaLogs
// instance.
type vlogsQuerier interface {
	QueryEvents(ctx context.Context, tenant string, start, end time.Time) (io.ReadCloser, error)
	DiscoverTenants(ctx context.Context, start, end time.Time) ([]string, error)
}

// Exporter orchestrates the full audit-log export run:
//  1. Bucket preflight (WORM check)
//  2. Determine all windows in [now - catchUpHours, previousCompleteHour]
//     that have not yet been exported (in chronological order)
//  3. For each window export (stream, tenant, hour)
type Exporter struct {
	cfg    *Config
	vlogs  vlogsQuerier
	s3     *S3Client
	signer Signer
	log    logr.Logger
}

// NewExporter constructs and validates an Exporter.
func NewExporter(cfg *Config, log logr.Logger) (*Exporter, error) {
	vlogs := NewVLogsClient(cfg.VLogsURL)

	s3client, err := NewS3Client(cfg)
	if err != nil {
		return nil, fmt.Errorf("create S3 client: %w", err)
	}

	signer, err := NewSigner(cfg)
	if err != nil {
		return nil, fmt.Errorf("create signer: %w", err)
	}

	return &Exporter{
		cfg:    cfg,
		vlogs:  vlogs,
		s3:     s3client,
		signer: signer,
		log:    log,
	}, nil
}

// Run executes the export job. It is intended to be called once per CronJob
// invocation and exits when all pending windows have been processed.
func (e *Exporter) Run(ctx context.Context) error {
	log := e.log

	// ── Bucket preflight ──────────────────────────────────────────────────
	warn, err := e.s3.PreflightBucket(ctx, e.cfg.ComplianceMode)
	if err != nil {
		return err // hard error in compliance mode
	}
	if warn != "" {
		log.Info(warn)
	}

	// ── Determine target window (previous complete hour minus export delay) ─
	now := time.Now().UTC()
	// previousHourEnd is the top of the most recently completed hour.
	previousHourEnd := now.Truncate(time.Hour)
	// Only export if enough time has passed for late events to arrive.
	if now.Sub(previousHourEnd) < e.cfg.ExportDelay {
		// Haven't waited long enough after the hour boundary yet.
		// Back up one more hour.
		previousHourEnd = previousHourEnd.Add(-time.Hour)
	}
	previousHourStart := previousHourEnd.Add(-time.Hour)

	// Catch-up: look back up to CatchUpHours.
	catchUpStart := previousHourStart.Add(-time.Duration(e.cfg.CatchUpHours-1) * time.Hour)
	log.Info("export run started",
		"catchup_from", catchUpStart.Format(time.RFC3339),
		"up_to_window_end", previousHourEnd.Format(time.RFC3339),
	)

	// ── Collect all (stream, tenant, window) combinations to process ───────
	windows, err := e.collectWindows(ctx, catchUpStart, previousHourEnd)
	if err != nil {
		return fmt.Errorf("collect windows: %w", err)
	}
	log.Info("windows to process", "count", len(windows))

	// ── Export each window in chronological order ─────────────────────────
	var exportErr error
	for _, w := range windows {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err := e.exportWindow(ctx, w); err != nil {
			var srcErr *SourceUnavailableError
			if errors.As(err, &srcErr) {
				// Source outage — alert and abort; the next CronJob run will catch up.
				log.Error(err, "SOURCE OUTAGE: VictoriaLogs is unreachable — alerting operator; export halted")
				return fmt.Errorf("source outage during export: %w", err)
			}
			// Destination or other error — log, record, continue to next window.
			log.Error(err, "failed to export window",
				"stream", w.Stream, "tenant", w.Tenant,
				"window", w.Start.Format("2006-01-02T15:04Z"))
			exportErr = fmt.Errorf("one or more windows failed (last: %w)", err)
		}
	}

	if exportErr != nil {
		return exportErr
	}
	log.Info("export run complete")
	return nil
}

// collectWindows builds the ordered list of ExportWindow values to process.
// Windows that already have a manifest.json in S3 are skipped (idempotency is
// also enforced inside exportWindow; the check here avoids unnecessary VLogs
// queries for large catch-up ranges).
func (e *Exporter) collectWindows(ctx context.Context, from, upToEnd time.Time) ([]ExportWindow, error) {
	var windows []ExportWindow

	for _, stream := range e.cfg.Streams {
		tenants, err := e.tenantsForStream(ctx, stream, from, upToEnd)
		if err != nil {
			return nil, fmt.Errorf("stream %q: get tenants: %w", stream, err)
		}

		for _, tenant := range tenants {
			for wStart := from; wStart.Before(upToEnd); wStart = wStart.Add(time.Hour) {
				w := ExportWindow{
					Stream: stream,
					Tenant: tenant,
					Start:  wStart,
					End:    wStart.Add(time.Hour),
				}
				// Quick idempotency pre-check to avoid redundant VLogs queries.
				manifestKey := w.S3KeyPrefix(e.cfg.S3Prefix) + "manifest.json"
				exists, err := e.s3.ObjectExists(ctx, manifestKey)
				if err != nil {
					return nil, fmt.Errorf("check manifest %q: %w", manifestKey, err)
				}
				if exists {
					continue
				}
				windows = append(windows, w)
			}
		}
	}
	return windows, nil
}

// tenantsForStream returns the list of tenants for the given stream.
// For platform-audit-log the tenant is always TenantPlatform.
// For tenant-audit-log, tenants are taken from config or auto-discovered.
func (e *Exporter) tenantsForStream(ctx context.Context, stream string, from, upToEnd time.Time) ([]string, error) {
	if stream == StreamPlatformAuditLog {
		return []string{TenantPlatform}, nil
	}

	if len(e.cfg.Tenants) > 0 {
		return e.cfg.Tenants, nil
	}

	// Auto-discover tenants from VictoriaLogs.
	tenants, err := e.vlogs.DiscoverTenants(ctx, from, upToEnd)
	if err != nil {
		return nil, fmt.Errorf("auto-discover tenants: %w", err)
	}
	if len(tenants) == 0 {
		e.log.Info("no tenants found for stream", "stream", stream)
	}
	return tenants, nil
}
