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
	"math"
	"time"

	"github.com/k0rdent/kof/kof-operator/internal/vlogs"
)

// Source identifiers.
const (
	SourceMetrics = "metrics"
	SourceLogs    = "logs"
	SourceTraces  = "traces"
)

// ExportWindow is one (source, tenant, cluster, hour) tuple to export.
type ExportWindow struct {
	Source  string // SourceMetrics | SourceLogs | SourceTraces
	Tenant  string
	Cluster string
	Start   time.Time // inclusive, top of hour UTC
	End     time.Time // exclusive, top of next hour UTC
}

// S3KeyPrefix returns the S3 object key prefix for the partition.
//
// Layout (spec §S3 partition layout):
//
//	<prefix>/tenant=<tenant>/cluster=<cluster>/dt=YYYY-MM-DD/hour=HH/<source>/
func (w ExportWindow) S3KeyPrefix(prefix string) string {
	dt := w.Start.UTC().Format("2006-01-02")
	hh := w.Start.UTC().Format("15")
	base := "tenant=" + w.Tenant +
		"/cluster=" + w.Cluster +
		"/dt=" + dt +
		"/hour=" + hh +
		"/" + w.Source + "/"
	if prefix != "" {
		return prefix + "/" + base
	}
	return base
}

// MetricRow is one sample in the metrics Parquet schema (spec F-9, F-10).
type MetricRow struct {
	// Promoted top-level columns (spec F-9)
	Timestamp        int64  // milliseconds UTC
	MetricName       string // __name__
	Value            float64
	Tenant           string
	Cluster          string
	ClusterNamespace string
	Namespace        string
	Pod              string
	Node             string
	Job              string
	// All other labels (spec F-10)
	Labels map[string]string
}

// vmExportLine is one NDJSON line from VictoriaMetrics /api/v1/export.
// Values are decoded as json.RawMessage to handle special strings such as
// "NaN", "Infinity", "-Infinity" that VictoriaMetrics emits for non-finite
// float values.
type vmExportLine struct {
	Metric     map[string]string `json:"metric"`
	Values     []json.RawMessage `json:"values"`
	Timestamps []int64           `json:"timestamps"`
}

// parseVMValue converts a raw JSON token from the values array into a float64.
// Handles numeric literals and the string forms "NaN", "Infinity", "-Infinity"
// emitted by VictoriaMetrics for non-finite values.
func parseVMValue(raw json.RawMessage) (float64, error) {
	// Try numeric first.
	var f float64
	if err := json.Unmarshal(raw, &f); err == nil {
		return f, nil
	}
	// Try string form ("NaN", "Infinity", "-Infinity").
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return 0, err
	}
	switch s {
	case "NaN":
		return math.NaN(), nil
	case "Infinity", "+Infinity":
		return math.Inf(1), nil
	case "-Infinity":
		return math.Inf(-1), nil
	default:
		return 0, fmt.Errorf("parseVMValue: unexpected string value %q (expected NaN, Infinity, or -Infinity)", s)
	}
}

// LogRow is one log entry in the logs Parquet schema, OTel-aligned (otel_logs).
type LogRow struct {
	// OTel core fields
	TimestampNs    int64 // Unix nanoseconds
	TraceId        string
	SpanId         string
	TraceFlags     uint32
	SeverityText   string
	SeverityNumber int32
	ServiceName    string
	Body           string
	// Platform-reserved top-level columns (spec F-9)
	Tenant           string
	Cluster          string
	ClusterNamespace string
	Namespace        string
	Pod              string
	Node             string
	// All other fields (spec F-10)
	Attributes map[string]string
}

// TraceEvent is one log/event entry on a span.
type TraceEvent struct {
	TimestampNs int64
	Name        string
	Attributes  map[string]string
}

// TraceRow is one span in the traces Parquet schema, OTel-aligned (otel_traces).
type TraceRow struct {
	// OTel core span fields
	Timestamp     int64 // Unix nanoseconds (span start)
	TraceId       string
	SpanId        string
	ParentSpanId  string
	TraceState    string
	TraceFlags    uint32
	SpanName      string
	SpanKind      string
	ServiceName   string
	Duration      int64 // nanoseconds
	StatusCode    string
	StatusMessage string
	// Platform-reserved top-level columns (spec F-9)
	Tenant           string
	Cluster          string
	ClusterNamespace string
	Namespace        string
	Pod              string
	Node             string
	// Attributes maps
	ResourceAttributes map[string]string
	SpanAttributes     map[string]string
	// Nested repeated fields (serialised as JSON bytes for Parquet compat)
	Events []TraceEvent
}

// SourceUnavailableError is an alias for vlogs.SourceUnavailableError so that
// existing callers within this package do not need to change their import paths.
type SourceUnavailableError = vlogs.SourceUnavailableError
