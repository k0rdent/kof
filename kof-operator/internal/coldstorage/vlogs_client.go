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
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/k0rdent/kof/kof-operator/internal/vlogs"
)

// vlogsSource is the human-readable name used in SourceUnavailableError.
const vlogsSource = "VictoriaLogs"

// VLogsClient queries VictoriaLogs over its HTTP API.
type VLogsClient struct {
	*vlogs.BaseClient
}

// NewVLogsClient creates a new VictoriaLogs client.
func NewVLogsClient(baseURL string) *VLogsClient {
	return &VLogsClient{
		BaseClient: vlogs.NewBaseClient(baseURL, 10*time.Minute),
	}
}

// ClusterField is the VictoriaLogs field that carries the cluster name.
// KOF collectors emit this as "k8s.cluster.name" (OTel semantic convention).
const ClusterField = "k8s.cluster.name"

// ExportLogs opens a streaming HTTP query against VictoriaLogs and returns
// the raw response body. The caller is responsible for closing the ReadCloser.
// The body is NDJSON: one flat JSON object per line.
//
// When tenant == defaultTenant the tenant filter is omitted (synthetic fallback for
// dev clusters without vmauth). Cluster filtering uses ClusterField.
func (c *VLogsClient) ExportLogs(
	ctx context.Context,
	tenant, cluster string,
	start, end time.Time,
) (io.ReadCloser, error) {
	// Filter by cluster. Omit tenant filter when using the synthetic "default".
	logsql := fmt.Sprintf(`%s:%q`, ClusterField, cluster)
	if tenant != "" && tenant != defaultTenant {
		logsql = fmt.Sprintf(`tenant:%q AND `, tenant) + logsql
	}

	params := url.Values{}
	params.Set("query", logsql)
	params.Set("start", start.UTC().Format(time.RFC3339))
	params.Set("end", end.UTC().Format(time.RFC3339))

	return c.QueryStream(ctx, vlogsSource, "/select/logsql/query", params)
}

// DiscoverLogTenants returns all distinct tenant values found in VictoriaLogs
// within [start, end). Returns nil (empty) when no "tenant" field is present —
// the caller should fall back to the synthetic "default" tenant.
func (c *VLogsClient) DiscoverLogTenants(ctx context.Context, start, end time.Time) ([]string, error) {
	return c.DiscoverFieldValues(ctx, vlogsSource, "tenant", "*", start, end)
}

// DiscoverLogClusters returns all distinct cluster values (from ClusterField)
// for the given tenant in [start, end).
func (c *VLogsClient) DiscoverLogClusters(ctx context.Context, tenant string, start, end time.Time) ([]string, error) {
	filter := "*"
	if tenant != "" && tenant != defaultTenant {
		filter = fmt.Sprintf(`tenant:%q`, tenant)
	}
	return c.DiscoverFieldValues(ctx, vlogsSource, ClusterField, filter, start, end)
}

// ScanVLogsExport reads NDJSON from r (as returned by ExportLogs), converts
// each line to a LogRow, and calls fn with batches of rows.
// tenant and cluster are passed through so we can write the synthetic values.
func ScanVLogsExport(r io.Reader, tenant, cluster string, fn func([]LogRow) error) error {
	const batchSize = 5000
	batch := make([]LogRow, 0, batchSize)

	scanner := bufio.NewScanner(r)
	// VictoriaLogs lines can be large with many fields; increase buffer.
	scanner.Buffer(make([]byte, 4*1024*1024), 4*1024*1024)

	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		if err := fn(batch); err != nil {
			return err
		}
		batch = batch[:0]
		return nil
	}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var raw map[string]string
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			continue // skip malformed lines
		}

		row := parseVLogsLine(raw, tenant, cluster)
		batch = append(batch, row)
		if len(batch) >= batchSize {
			if err := flush(); err != nil {
				return err
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scan vlogs export: %w", err)
	}
	return flush()
}

// parseVLogsLine maps a flat VictoriaLogs JSON record to a LogRow.
// VictoriaLogs flattens all OTel fields into the top-level JSON object.
func parseVLogsLine(raw map[string]string, tenant, cluster string) LogRow {
	row := LogRow{
		Tenant:  tenant,
		Cluster: cluster,
	}

	// Parse timestamp from _time field (RFC3339 nanoseconds).
	if t, ok := raw["_time"]; ok {
		if ts, err := time.Parse(time.RFC3339Nano, t); err == nil {
			row.TimestampNs = ts.UnixNano()
		}
	}

	row.Body = raw["_msg"]
	row.SeverityText = raw["severity"]
	row.ServiceName = raw["service.name"]
	row.TraceId = raw["trace_id"]
	row.SpanId = raw["span_id"]

	// Platform-reserved columns (spec F-9) — map OTel k8s fields.
	row.ClusterNamespace = raw["k8s.cluster.namespace"]
	row.Namespace = raw["k8s.namespace.name"]
	row.Pod = raw["k8s.pod.name"]
	row.Node = raw["k8s.node.name"]

	// All other fields go to Attributes (spec F-10), excluding internal VL fields.
	reserved := map[string]bool{
		"_time": true, "_msg": true, "_stream": true, "_stream_id": true,
		"severity": true, "service.name": true, "trace_id": true, "span_id": true,
		"k8s.cluster.namespace": true, "k8s.namespace.name": true,
		"k8s.pod.name": true, "k8s.node.name": true,
		"k8s.cluster.name": true, // already in cluster column
		"tenant":           true,
	}
	attrs := make(map[string]string)
	for k, v := range raw {
		if !reserved[k] && v != "" {
			attrs[k] = v
		}
	}
	if len(attrs) > 0 {
		row.Attributes = attrs
	}
	return row
}
