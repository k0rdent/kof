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

package coldstorage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/go-logr/logr"
)

// defaultTenant is the synthetic tenant name used as a fallback when no tenant
// labels are found in the data source. Several places in the package compare
// against this value so it is defined once here.
const defaultTenant = "default"

// Exporter orchestrates the full cold-storage export run:
//  1. Determine all (source, tenant, cluster, hour) windows in [catchUpStart, prevHourEnd]
//     that have not yet been exported.
//  2. For each window, extract data from the appropriate source, convert to Parquet, and
//     upload to S3.
type Exporter struct {
	cfg     *Config
	vm      *VMClient
	vlogs   *VLogsClient
	vtraces *VTracesClient
	s3      *S3Client
	log     logr.Logger
}

// NewExporter constructs and validates an Exporter.
func NewExporter(cfg *Config, log logr.Logger) (*Exporter, error) {
	s3client, err := NewS3Client(cfg)
	if err != nil {
		return nil, fmt.Errorf("create S3 client: %w", err)
	}

	return &Exporter{
		cfg:     cfg,
		vm:      NewVMClient(cfg.VMURL),
		vlogs:   NewVLogsClient(cfg.VLogsURL),
		vtraces: NewVTracesClient(cfg.VTracesURL),
		s3:      s3client,
		log:     log,
	}, nil
}

// Run executes the export job once and exits when all pending windows have
// been processed. It is intended to be called once per CronJob invocation.
func (e *Exporter) Run(ctx context.Context) error {
	log := e.log

	// Determine target window (previous complete hour minus export delay).
	now := time.Now().UTC()
	previousHourEnd := now.Truncate(time.Hour)
	if now.Sub(previousHourEnd) < e.cfg.ExportDelay {
		previousHourEnd = previousHourEnd.Add(-time.Hour)
	}
	previousHourStart := previousHourEnd.Add(-time.Hour)
	catchUpStart := previousHourStart.Add(-time.Duration(e.cfg.CatchUpHours-1) * time.Hour)

	log.Info("export run started",
		"catchup_from", catchUpStart.Format(time.RFC3339),
		"up_to_window_end", previousHourEnd.Format(time.RFC3339),
		"sources", e.cfg.Sources,
	)

	windows, err := e.collectWindows(ctx, catchUpStart, previousHourEnd)
	if err != nil {
		return fmt.Errorf("collect windows: %w", err)
	}
	log.Info("windows to process", "count", len(windows))

	var exportErr error
	for _, w := range windows {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err := e.exportWindow(ctx, w); err != nil {
			var srcErr *SourceUnavailableError
			if errors.As(err, &srcErr) {
				log.Error(err, "SOURCE OUTAGE: upstream is unreachable — halting run; next CronJob invocation will catch up")
				return fmt.Errorf("source outage: %w", err)
			}
			log.Error(err, "failed to export window",
				"source", w.Source, "tenant", w.Tenant, "cluster", w.Cluster,
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

// collectWindows builds the ordered list of ExportWindow values to process,
// skipping windows that already have a _SUCCESS marker in S3 (idempotency).
func (e *Exporter) collectWindows(ctx context.Context, from, upToEnd time.Time) ([]ExportWindow, error) {
	var windows []ExportWindow

	for _, source := range e.cfg.Sources {
		tenants, err := e.discoverTenants(ctx, source, from, upToEnd)
		if err != nil {
			var srcErr *SourceUnavailableError
			if errors.As(err, &srcErr) {
				e.log.Error(err, "source unreachable during window collection; skipping source for this run", "source", source)
				continue
			}
			return nil, fmt.Errorf("source %q: discover tenants: %w", source, err)
		}

		for _, tenant := range tenants {
			clusters, err := e.discoverClusters(ctx, source, tenant, from, upToEnd)
			if err != nil {
				var srcErr *SourceUnavailableError
				if errors.As(err, &srcErr) {
					e.log.Error(err, "source unreachable during cluster discovery; skipping source for this run", "source", source, "tenant", tenant)
					break // skip all tenants for this source
				}
				return nil, fmt.Errorf("source %q tenant %q: discover clusters: %w", source, tenant, err)
			}

			for _, cluster := range clusters {
				for wStart := from; wStart.Before(upToEnd); wStart = wStart.Add(time.Hour) {
					w := ExportWindow{
						Source:  source,
						Tenant:  tenant,
						Cluster: cluster,
						Start:   wStart,
						End:     wStart.Add(time.Hour),
					}
					// Quick idempotency pre-check: skip already-exported windows.
					successKey := w.S3KeyPrefix(e.cfg.S3Prefix) + "_SUCCESS"
					exists, err := e.s3.ObjectExists(ctx, successKey)
					if err != nil {
						return nil, fmt.Errorf("check success marker %q: %w", successKey, err)
					}
					if exists {
						continue
					}
					windows = append(windows, w)
				}
			}
		}
	}
	return windows, nil
}

// discoverTenants returns the list of tenants for the given source.
// Uses the configured list when provided; otherwise auto-discovers from the source.
func (e *Exporter) discoverTenants(ctx context.Context, source string, from, upToEnd time.Time) ([]string, error) {
	if len(e.cfg.Tenants) > 0 {
		return e.cfg.Tenants, nil
	}

	switch source {
	case SourceMetrics:
		tenants, err := e.vm.DiscoverTenants(ctx, from, upToEnd)
		if err != nil {
			return nil, err
		}
		if len(tenants) == 0 {
			// No tenant label in the data (e.g. single-cluster dev setup without
			// vmauth). Fall back to a single synthetic "default" tenant that
			// exports all metrics for each discovered cluster without filtering
			// by the tenant label.
			e.log.Info("no tenant labels found in VictoriaMetrics; falling back to synthetic tenant=default",
				"from", from.Format(time.RFC3339), "to", upToEnd.Format(time.RFC3339))
			return []string{defaultTenant}, nil
		}
		return tenants, nil
	case SourceLogs:
		tenants, err := e.vlogs.DiscoverLogTenants(ctx, from, upToEnd)
		if err != nil {
			return nil, err
		}
		if len(tenants) == 0 {
			e.log.Info("no tenant labels found in VictoriaLogs; falling back to synthetic tenant=default",
				"from", from.Format(time.RFC3339), "to", upToEnd.Format(time.RFC3339))
			return []string{defaultTenant}, nil
		}
		return tenants, nil
	default:
		// Traces: fall back to synthetic "default" tenant.
		tenants, err := e.vtraces.DiscoverTraceTenants(ctx, from, upToEnd)
		if err != nil {
			return nil, err
		}
		if len(tenants) == 0 {
			e.log.Info("no tenant labels found in VictoriaTraces; falling back to synthetic tenant=default",
				"from", from.Format(time.RFC3339), "to", upToEnd.Format(time.RFC3339))
			return []string{defaultTenant}, nil
		}
		return tenants, nil
	}
}

// discoverClusters returns the list of clusters for the given (source, tenant).
func (e *Exporter) discoverClusters(ctx context.Context, source, tenant string, from, upToEnd time.Time) ([]string, error) {
	if len(e.cfg.Clusters) > 0 {
		return e.cfg.Clusters, nil
	}

	switch source {
	case SourceMetrics:
		clusters, err := e.vm.DiscoverClusters(ctx, tenant, from, upToEnd)
		if err != nil {
			return nil, err
		}
		if len(clusters) == 0 {
			e.log.Info("no cluster labels found for tenant in VictoriaMetrics",
				"tenant", tenant)
		}
		return clusters, nil
	case SourceLogs:
		clusters, err := e.vlogs.DiscoverLogClusters(ctx, tenant, from, upToEnd)
		if err != nil {
			return nil, err
		}
		return clusters, nil
	case SourceTraces:
		// Discover clusters by inspecting span resource attributes.
		clusters, err := e.vtraces.DiscoverTraceClusters(ctx, tenant, from, upToEnd)
		if err != nil {
			return nil, err
		}
		if len(clusters) == 0 {
			e.log.Info("no cluster tags found in VictoriaTraces spans; falling back to synthetic cluster=default",
				"tenant", tenant)
			return []string{defaultTenant}, nil
		}
		return clusters, nil
	default:
		return nil, nil
	}
}

// exportWindow exports a single (source, tenant, cluster, hour) window.
func (e *Exporter) exportWindow(ctx context.Context, w ExportWindow) error {
	log := e.log.WithValues(
		"source", w.Source,
		"tenant", w.Tenant,
		"cluster", w.Cluster,
		"window", w.Start.Format("2006-01-02T15:04Z"),
	)

	prefix := w.S3KeyPrefix(e.cfg.S3Prefix)
	successKey := prefix + "_SUCCESS"

	// Idempotency: abort if already exported.
	exists, err := e.s3.ObjectExists(ctx, successKey)
	if err != nil {
		return fmt.Errorf("idempotency check: %w", err)
	}
	if exists {
		log.Info("window already exported, skipping")
		return nil
	}

	switch w.Source {
	case SourceMetrics:
		return e.exportMetricsWindow(ctx, log, w, prefix, successKey)
	case SourceLogs:
		return e.exportLogsWindow(ctx, log, w, prefix, successKey)
	case SourceTraces:
		return e.exportTracesWindow(ctx, log, w, prefix, successKey)
	default:
		log.Info("source not yet implemented, skipping", "source", w.Source)
		return nil
	}
}

// exportMetricsWindow extracts metrics from VictoriaMetrics and streams them
// as a Parquet file directly to S3 without buffering all rows in memory.
func (e *Exporter) exportMetricsWindow(ctx context.Context, log logr.Logger, w ExportWindow, prefix, successKey string) error {
	log.Info("querying VictoriaMetrics")
	// Pass the synthetic "default" tenant as an empty string so the VM query
	// does not add a tenant label filter (the label may not exist in the data).
	vmTenant := w.Tenant
	if vmTenant == defaultTenant {
		vmTenant = ""
	}
	body, err := e.vm.ExportMetrics(ctx, vmTenant, w.Cluster, w.Start, w.End)
	if err != nil {
		return err
	}
	defer func() { _ = body.Close() }()

	parquetKey := prefix + "metrics.parquet"
	log.Info("streaming parquet to S3", "key", parquetKey)

	// Stream: VM export body → Parquet writer → io.Pipe → S3 multipart upload.
	// The parquet writer and S3 upload run concurrently — no full buffering.
	pr, pw := io.Pipe()
	var rowCount int
	uploadErrCh := make(chan error, 1)
	writeErrCh := make(chan error, 1)

	// S3 upload goroutine reads from the pipe.
	go func() {
		err := e.s3.UploadStream(ctx, parquetKey, pr, "application/octet-stream")
		// Always close the reader side: if the upload fails before consuming all
		// data the writer goroutine would block forever on pipe writes.
		pr.CloseWithError(err) //nolint:errcheck
		uploadErrCh <- err
	}()

	// Parquet write goroutine: scan VM export and write rows to the parquet writer.
	go func() {
		parquetWriter := NewMetricsParquetWriter(pw)
		writeErr := ScanVMExport(body, func(rows []MetricRow) error {
			return parquetWriter.Write(rows)
		})
		if writeErr == nil {
			writeErr = parquetWriter.Close()
		}
		rowCount = parquetWriter.RowCount()
		pw.CloseWithError(writeErr) //nolint:errcheck
		writeErrCh <- writeErr
	}()

	// Wait for both goroutines.
	writeErr := <-writeErrCh
	uploadErr := <-uploadErrCh

	if writeErr != nil {
		return fmt.Errorf("write parquet stream: %w", writeErr)
	}
	if uploadErr != nil {
		return fmt.Errorf("upload parquet stream: %w", uploadErr)
	}

	log.Info("parquet upload complete", "rows", rowCount)

	// Write _SUCCESS marker (idempotency signal — uploaded last).
	if err := e.s3.PutObject(ctx, successKey, []byte{}, "text/plain"); err != nil {
		return fmt.Errorf("upload success marker: %w", err)
	}

	log.Info("window exported successfully", "rows", rowCount)
	return nil
}

// exportLogsWindow extracts logs from VictoriaLogs and writes Parquet to S3.
// The schema is OTel-aligned (otel_logs), matching the ClickHouse contrib exporter.
func (e *Exporter) exportLogsWindow(ctx context.Context, log logr.Logger, w ExportWindow, prefix, successKey string) error {
	log.Info("querying VictoriaLogs")
	body, err := e.vlogs.ExportLogs(ctx, w.Tenant, w.Cluster, w.Start, w.End)
	if err != nil {
		return err
	}
	defer func() { _ = body.Close() }()

	logsKey := prefix + "logs.parquet"
	log.Info("streaming logs parquet to S3", "key", logsKey)

	// Stream: VLogs export body → Parquet writer → io.Pipe → S3 multipart upload.
	pr, pw := io.Pipe()
	var rowCount int
	uploadErrCh := make(chan error, 1)
	writeErrCh := make(chan error, 1)

	go func() {
		err := e.s3.UploadStream(ctx, logsKey, pr, "application/octet-stream")
		pr.CloseWithError(err) //nolint:errcheck
		uploadErrCh <- err
	}()

	go func() {
		parquetWriter := NewLogsParquetWriter(pw)
		writeErr := ScanVLogsExport(body, w.Tenant, w.Cluster, func(rows []LogRow) error {
			return parquetWriter.Write(rows)
		})
		if writeErr == nil {
			writeErr = parquetWriter.Close()
		}
		rowCount = parquetWriter.RowCount()
		pw.CloseWithError(writeErr) //nolint:errcheck
		writeErrCh <- writeErr
	}()

	writeErr := <-writeErrCh
	uploadErr := <-uploadErrCh

	if writeErr != nil {
		return fmt.Errorf("write logs parquet stream: %w", writeErr)
	}
	if uploadErr != nil {
		return fmt.Errorf("upload logs parquet stream: %w", uploadErr)
	}

	log.Info("logs parquet upload complete", "rows", rowCount)

	if err := e.s3.PutObject(ctx, successKey, []byte{}, "text/plain"); err != nil {
		return fmt.Errorf("upload success marker: %w", err)
	}

	log.Info("logs window exported successfully", "rows", rowCount)
	return nil
}

// exportTracesWindow extracts traces from VictoriaTraces (LogsQL API) and writes
// Parquet to S3. The schema is OTel-aligned (otel_traces). Each row is one span.
func (e *Exporter) exportTracesWindow(ctx context.Context, log logr.Logger, w ExportWindow, prefix, successKey string) error {
	log.Info("querying VictoriaTraces")
	body, err := e.vtraces.ExportTraces(ctx, w.Tenant, w.Cluster, w.Start, w.End)
	if err != nil {
		return err
	}
	defer func() { _ = body.Close() }()

	tracesKey := prefix + "traces.parquet"
	log.Info("streaming traces parquet to S3", "key", tracesKey)

	pr, pw := io.Pipe()
	var rowCount int
	uploadErrCh := make(chan error, 1)
	writeErrCh := make(chan error, 1)

	go func() {
		err := e.s3.UploadStream(ctx, tracesKey, pr, "application/octet-stream")
		pr.CloseWithError(err) //nolint:errcheck
		uploadErrCh <- err
	}()

	go func() {
		parquetWriter := NewTracesParquetWriter(pw)
		writeErr := ScanVTracesExport(body, w.Tenant, w.Cluster, func(rows []TraceRow) error {
			return parquetWriter.Write(rows)
		})
		if writeErr == nil {
			writeErr = parquetWriter.Close()
		}
		rowCount = parquetWriter.RowCount()
		pw.CloseWithError(writeErr) //nolint:errcheck
		writeErrCh <- writeErr
	}()

	writeErr := <-writeErrCh
	uploadErr := <-uploadErrCh

	if writeErr != nil {
		return fmt.Errorf("write traces parquet stream: %w", writeErr)
	}
	if uploadErr != nil {
		return fmt.Errorf("upload traces parquet stream: %w", uploadErr)
	}

	log.Info("traces parquet upload complete", "rows", rowCount)

	if err := e.s3.PutObject(ctx, successKey, []byte{}, "text/plain"); err != nil {
		return fmt.Errorf("upload success marker: %w", err)
	}

	log.Info("traces window exported successfully", "rows", rowCount)
	return nil
}
