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
	"encoding/json"
	"fmt"
	"io"

	"github.com/parquet-go/parquet-go"
	"github.com/parquet-go/parquet-go/compress/zstd"
)

// ──────────────────────────────────────────────────────────────────────────────
// Metrics Parquet schema (spec F-9, F-10)
// ──────────────────────────────────────────────────────────────────────────────

// metricParquetRow is the Parquet-tagged struct for the metrics schema.
// All platform-reserved labels are promoted to top-level columns.
// All other labels are stored in a map column serialised as JSON bytes.
type metricParquetRow struct {
	Timestamp        int64   `parquet:"timestamp"`
	MetricName       string  `parquet:"metric_name,dict"`
	Value            float64 `parquet:"value"`
	Tenant           string  `parquet:"tenant,dict"`
	Cluster          string  `parquet:"cluster,dict"`
	ClusterNamespace string  `parquet:"cluster_namespace,dict,optional"`
	Namespace        string  `parquet:"namespace,dict,optional"`
	Pod              string  `parquet:"pod,dict,optional"`
	Node             string  `parquet:"node,dict,optional"`
	Job              string  `parquet:"job,dict,optional"`
	// Labels serialised as JSON object (map<string,string>).
	// parquet-go does not natively support map columns; we encode as a JSON
	// byte string and document that consumers should parse it.
	LabelsJSON []byte `parquet:"labels,optional"`
}

// MetricsParquetWriter streams MetricRows into a Parquet file written to w.
// Call Write for each batch of rows, then Close to flush the footer.
type MetricsParquetWriter struct {
	pw    *parquet.GenericWriter[metricParquetRow]
	count int
}

// NewMetricsParquetWriter creates a streaming Parquet writer with ZSTD compression.
func NewMetricsParquetWriter(w io.Writer) *MetricsParquetWriter {
	pw := parquet.NewGenericWriter[metricParquetRow](w,
		parquet.Compression(&zstd.Codec{}),
	)
	return &MetricsParquetWriter{pw: pw}
}

// Write converts and appends a batch of MetricRows to the Parquet file.
func (m *MetricsParquetWriter) Write(rows []MetricRow) error {
	parquetRows := make([]metricParquetRow, 0, len(rows))
	for _, r := range rows {
		pr := metricParquetRow{
			Timestamp:        r.Timestamp,
			MetricName:       r.MetricName,
			Value:            r.Value,
			Tenant:           r.Tenant,
			Cluster:          r.Cluster,
			ClusterNamespace: r.ClusterNamespace,
			Namespace:        r.Namespace,
			Pod:              r.Pod,
			Node:             r.Node,
			Job:              r.Job,
		}
		if len(r.Labels) > 0 {
			b, err := json.Marshal(r.Labels)
			if err != nil {
				return fmt.Errorf("marshal labels: %w", err)
			}
			pr.LabelsJSON = b
		}
		parquetRows = append(parquetRows, pr)
	}
	n, err := m.pw.Write(parquetRows)
	m.count += n
	return err
}

// Close flushes the Parquet footer and closes the writer. Must be called once.
func (m *MetricsParquetWriter) Close() error {
	return m.pw.Close()
}

// RowCount returns the number of rows written so far.
func (m *MetricsParquetWriter) RowCount() int {
	return m.count
}

// ──────────────────────────────────────────────────────────────────────────────
// Logs Parquet schema — aligned with OTel ClickHouse contrib exporter (otel_logs)
// ──────────────────────────────────────────────────────────────────────────────

// logParquetRow maps a VictoriaLogs NDJSON record to the OTel-compatible Parquet
// schema as produced by the ClickHouse contrib exporter for otel_logs.
//
// Platform-reserved fields (spec F-9) are promoted to top-level columns.
// All remaining fields are stored in LogAttributes as JSON bytes.
type logParquetRow struct {
	// OTel core columns
	Timestamp      int64  `parquet:"Timestamp"` // Unix nanoseconds
	TraceId        string `parquet:"TraceId,dict,optional"`
	SpanId         string `parquet:"SpanId,dict,optional"`
	TraceFlags     uint32 `parquet:"TraceFlags,optional"`
	SeverityText   string `parquet:"SeverityText,dict,optional"`
	SeverityNumber int32  `parquet:"SeverityNumber,optional"`
	ServiceName    string `parquet:"ServiceName,dict,optional"`
	Body           string `parquet:"Body,optional"`
	// Platform-reserved top-level columns (spec F-9)
	Tenant           string `parquet:"tenant,dict"`
	Cluster          string `parquet:"cluster,dict"`
	ClusterNamespace string `parquet:"cluster_namespace,dict,optional"`
	Namespace        string `parquet:"namespace,dict,optional"`
	Pod              string `parquet:"pod,dict,optional"`
	Node             string `parquet:"node,dict,optional"`
	// All other log fields serialised as JSON bytes (spec F-10).
	LogAttributes []byte `parquet:"LogAttributes,optional"`
}

// LogsParquetWriter streams LogRow values into a Parquet file written to w.
type LogsParquetWriter struct {
	pw    *parquet.GenericWriter[logParquetRow]
	count int
}

// NewLogsParquetWriter creates a streaming Parquet writer for logs with ZSTD compression.
func NewLogsParquetWriter(w io.Writer) *LogsParquetWriter {
	pw := parquet.NewGenericWriter[logParquetRow](w,
		parquet.Compression(&zstd.Codec{}),
	)
	return &LogsParquetWriter{pw: pw}
}

// Write appends a batch of log rows to the Parquet file.
func (l *LogsParquetWriter) Write(rows []LogRow) error {
	parquetRows := make([]logParquetRow, 0, len(rows))
	for _, r := range rows {
		pr := logParquetRow{
			Timestamp:        r.TimestampNs,
			TraceId:          r.TraceId,
			SpanId:           r.SpanId,
			TraceFlags:       r.TraceFlags,
			SeverityText:     r.SeverityText,
			SeverityNumber:   r.SeverityNumber,
			ServiceName:      r.ServiceName,
			Body:             r.Body,
			Tenant:           r.Tenant,
			Cluster:          r.Cluster,
			ClusterNamespace: r.ClusterNamespace,
			Namespace:        r.Namespace,
			Pod:              r.Pod,
			Node:             r.Node,
		}
		if len(r.Attributes) > 0 {
			b, err := json.Marshal(r.Attributes)
			if err != nil {
				return fmt.Errorf("marshal log attributes: %w", err)
			}
			pr.LogAttributes = b
		}
		parquetRows = append(parquetRows, pr)
	}
	n, err := l.pw.Write(parquetRows)
	l.count += n
	return err
}

// Close flushes the Parquet footer and closes the writer.
func (l *LogsParquetWriter) Close() error {
	return l.pw.Close()
}

// RowCount returns the number of rows written so far.
func (l *LogsParquetWriter) RowCount() int {
	return l.count
}

// ──────────────────────────────────────────────────────────────────────────────
// Traces Parquet schema — aligned with OTel ClickHouse contrib exporter (otel_traces)
// ──────────────────────────────────────────────────────────────────────────────

// traceParquetRow maps a TraceRow (span) to the OTel-compatible Parquet schema
// as produced by the ClickHouse contrib exporter for otel_traces.
//
// Repeated/nested columns (Events, Links) are serialised as JSON bytes because
// parquet-go does not support nested repeated structs without a custom schema.
type traceParquetRow struct {
	// OTel core span columns
	Timestamp     int64  `parquet:"Timestamp"` // Unix nanoseconds
	TraceId       string `parquet:"TraceId,dict,optional"`
	SpanId        string `parquet:"SpanId,dict,optional"`
	ParentSpanId  string `parquet:"ParentSpanId,dict,optional"`
	TraceState    string `parquet:"TraceState,dict,optional"`
	TraceFlags    uint32 `parquet:"TraceFlags,optional"`
	SpanName      string `parquet:"SpanName,dict,optional"`
	SpanKind      string `parquet:"SpanKind,dict,optional"`
	ServiceName   string `parquet:"ServiceName,dict,optional"`
	Duration      int64  `parquet:"Duration,optional"` // nanoseconds
	StatusCode    string `parquet:"StatusCode,dict,optional"`
	StatusMessage string `parquet:"StatusMessage,optional"`
	// Platform-reserved columns (spec F-9)
	Tenant           string `parquet:"tenant,dict"`
	Cluster          string `parquet:"cluster,dict"`
	ClusterNamespace string `parquet:"cluster_namespace,dict,optional"`
	Namespace        string `parquet:"namespace,dict,optional"`
	Pod              string `parquet:"pod,dict,optional"`
	Node             string `parquet:"node,dict,optional"`
	// Attribute maps serialised as JSON bytes (spec F-10)
	ResourceAttributes []byte `parquet:"ResourceAttributes,optional"`
	SpanAttributes     []byte `parquet:"SpanAttributes,optional"`
	// Nested repeated fields serialised as JSON bytes
	Events []byte `parquet:"Events,optional"`
}

// TracesParquetWriter streams TraceRows (spans) into a Parquet file written to w.
type TracesParquetWriter struct {
	pw    *parquet.GenericWriter[traceParquetRow]
	count int
}

// NewTracesParquetWriter creates a streaming Parquet writer for traces with ZSTD compression.
func NewTracesParquetWriter(w io.Writer) *TracesParquetWriter {
	pw := parquet.NewGenericWriter[traceParquetRow](w,
		parquet.Compression(&zstd.Codec{}),
	)
	return &TracesParquetWriter{pw: pw}
}

// Write appends a batch of TraceRows to the Parquet file.
func (t *TracesParquetWriter) Write(rows []TraceRow) error {
	parquetRows := make([]traceParquetRow, 0, len(rows))
	for _, r := range rows {
		pr := traceParquetRow{
			Timestamp:        r.Timestamp,
			TraceId:          r.TraceId,
			SpanId:           r.SpanId,
			ParentSpanId:     r.ParentSpanId,
			TraceState:       r.TraceState,
			TraceFlags:       r.TraceFlags,
			SpanName:         r.SpanName,
			SpanKind:         r.SpanKind,
			ServiceName:      r.ServiceName,
			Duration:         r.Duration,
			StatusCode:       r.StatusCode,
			StatusMessage:    r.StatusMessage,
			Tenant:           r.Tenant,
			Cluster:          r.Cluster,
			ClusterNamespace: r.ClusterNamespace,
			Namespace:        r.Namespace,
			Pod:              r.Pod,
			Node:             r.Node,
		}
		if len(r.ResourceAttributes) > 0 {
			b, err := json.Marshal(r.ResourceAttributes)
			if err != nil {
				return fmt.Errorf("marshal resource attributes: %w", err)
			}
			pr.ResourceAttributes = b
		}
		if len(r.SpanAttributes) > 0 {
			b, err := json.Marshal(r.SpanAttributes)
			if err != nil {
				return fmt.Errorf("marshal span attributes: %w", err)
			}
			pr.SpanAttributes = b
		}
		if len(r.Events) > 0 {
			// Serialise events as [{"ts":<ns>,"name":"...","attrs":{...}}]
			type eventJSON struct {
				Ts    int64             `json:"ts"`
				Name  string            `json:"name"`
				Attrs map[string]string `json:"attrs,omitempty"`
			}
			evs := make([]eventJSON, len(r.Events))
			for i, ev := range r.Events {
				evs[i] = eventJSON{Ts: ev.TimestampNs, Name: ev.Name, Attrs: ev.Attributes}
			}
			b, err := json.Marshal(evs)
			if err != nil {
				return fmt.Errorf("marshal events: %w", err)
			}
			pr.Events = b
		}
		parquetRows = append(parquetRows, pr)
	}
	n, err := t.pw.Write(parquetRows)
	t.count += n
	return err
}

// Close flushes the Parquet footer and closes the writer.
func (t *TracesParquetWriter) Close() error {
	return t.pw.Close()
}

// RowCount returns the number of rows (spans) written so far.
func (t *TracesParquetWriter) RowCount() int {
	return t.count
}
