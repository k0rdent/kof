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
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// VTracesClient queries VictoriaTraces via its VictoriaLogs-compatible LogsQL API.
//
// VictoriaTraces exposes the same HTTP query interface as VictoriaLogs on port 10471:
//
//	GET /select/logsql/query?query=<logsql>&start=<rfc3339>&end=<rfc3339>&limit=<n>
//	GET /select/logsql/field_values?field=<f>&query=*&start=...&end=...
//
// Each response line is a flat JSON object with OTel trace span fields.
type VTracesClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewVTracesClient creates a new VictoriaTraces client.
func NewVTracesClient(baseURL string) *VTracesClient {
	return &VTracesClient{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{Timeout: 10 * time.Minute},
	}
}

// TraceClusterField is the VictoriaTraces field that carries the cluster name.
// KOF collectors emit this as a resource attribute with the OTel k8s convention.
const TraceClusterField = "resource_attr:k8s.cluster.name"

// ExportTraces opens a streaming LogsQL query against VictoriaTraces and returns
// the raw NDJSON response body. The caller is responsible for closing it.
//
// When tenant == defaultTenant the tenant filter is omitted. Cluster filtering
// uses TraceClusterField wrapped in backticks because it contains a colon.
func (c *VTracesClient) ExportTraces(
	ctx context.Context,
	tenant, cluster string,
	start, end time.Time,
) (io.ReadCloser, error) {
	// TraceClusterField contains a colon so it must be backtick-quoted in LogsQL.
	logsql := fmt.Sprintf("`%s`:%q", TraceClusterField, cluster)
	if tenant != "" && tenant != defaultTenant {
		logsql = fmt.Sprintf(`tenant:%q AND `, tenant) + logsql
	}

	params := url.Values{}
	params.Set("query", logsql)
	params.Set("start", start.UTC().Format(time.RFC3339))
	params.Set("end", end.UTC().Format(time.RFC3339))
	params.Set("limit", "1000000")

	reqURL := c.baseURL + "/select/logsql/query?" + params.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build VTraces request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, &SourceUnavailableError{Source: "VictoriaTraces", Cause: err}
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		_ = resp.Body.Close()
		return nil, &SourceUnavailableError{
			Source: "VictoriaTraces",
			Cause:  fmt.Errorf("HTTP %d: %s", resp.StatusCode, body),
		}
	}
	return resp.Body, nil
}

// DiscoverTraceTenants returns all distinct tenant values in VictoriaTraces within [start, end).
func (c *VTracesClient) DiscoverTraceTenants(ctx context.Context, start, end time.Time) ([]string, error) {
	return c.discoverFieldValues(ctx, "tenant", "*", start, end)
}

// DiscoverTraceClusters returns all distinct cluster values (from TraceClusterField)
// for the given tenant in [start, end).
func (c *VTracesClient) DiscoverTraceClusters(ctx context.Context, tenant string, start, end time.Time) ([]string, error) {
	filter := "*"
	if tenant != "" && tenant != defaultTenant {
		// tenant field doesn't contain special chars, no backticks needed.
		filter = fmt.Sprintf(`tenant:%q`, tenant)
	}
	return c.discoverFieldValues(ctx, TraceClusterField, filter, start, end)
}

func (c *VTracesClient) discoverFieldValues(
	ctx context.Context,
	field, query string,
	start, end time.Time,
) ([]string, error) {
	params := url.Values{}
	params.Set("field", field)
	params.Set("query", query)
	params.Set("start", start.UTC().Format(time.RFC3339))
	params.Set("end", end.UTC().Format(time.RFC3339))

	reqURL := c.baseURL + "/select/logsql/field_values?" + params.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build field_values request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, &SourceUnavailableError{Source: "VictoriaTraces", Cause: err}
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, &SourceUnavailableError{
			Source: "VictoriaTraces",
			Cause:  fmt.Errorf("field_values HTTP %d: %s", resp.StatusCode, body),
		}
	}

	// Response: {"values":[{"value":"<v>","hits":<n>}, ...]}
	type fieldValue struct {
		Value string `json:"value"`
	}
	type fieldValuesResp struct {
		Values []fieldValue `json:"values"`
	}
	var result fieldValuesResp
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read field_values response: %w", err)
	}
	if err := json.Unmarshal(body, &result); err != nil {
		// Fallback: try NDJSON format
		var values []string
		scanner := bufio.NewScanner(strings.NewReader(string(body)))
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}
			var fv fieldValue
			if err2 := json.Unmarshal([]byte(line), &fv); err2 != nil {
				continue
			}
			if fv.Value != "" {
				values = append(values, fv.Value)
			}
		}
		return values, scanner.Err()
	}
	values := make([]string, 0, len(result.Values))
	for _, fv := range result.Values {
		if fv.Value != "" {
			values = append(values, fv.Value)
		}
	}
	return values, nil
}

// ScanVTracesExport reads NDJSON from r (as returned by ExportTraces), converts
// each line to a TraceRow, and calls fn with batches of rows.
func ScanVTracesExport(r io.Reader, tenant, cluster string, fn func([]TraceRow) error) error {
	const batchSize = 5000
	batch := make([]TraceRow, 0, batchSize)

	scanner := bufio.NewScanner(r)
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
			continue
		}
		row := parseVTracesLine(raw, tenant, cluster)
		batch = append(batch, row)
		if len(batch) >= batchSize {
			if err := flush(); err != nil {
				return err
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scan vtraces export: %w", err)
	}
	return flush()
}

// spanKindName maps the integer kind value to its OTel string name.
func spanKindName(s string) string {
	switch s {
	case "1":
		return "SPAN_KIND_INTERNAL"
	case "2":
		return "SPAN_KIND_SERVER"
	case "3":
		return "SPAN_KIND_CLIENT"
	case "4":
		return "SPAN_KIND_PRODUCER"
	case "5":
		return "SPAN_KIND_CONSUMER"
	default:
		return "SPAN_KIND_UNSPECIFIED"
	}
}

// statusCodeName maps the integer status code to its OTel string name.
func statusCodeName(s string) string {
	switch s {
	case "1":
		return "STATUS_CODE_OK"
	case "2":
		return "STATUS_CODE_ERROR"
	default:
		return "STATUS_CODE_UNSET"
	}
}

// parseVTracesLine maps a flat VictoriaTraces NDJSON span record to a TraceRow.
func parseVTracesLine(raw map[string]string, tenant, cluster string) TraceRow {
	row := TraceRow{
		Tenant:        tenant,
		Cluster:       cluster,
		TraceId:       raw["trace_id"],
		SpanId:        raw["span_id"],
		ParentSpanId:  raw["parent_span_id"],
		SpanName:      raw["name"],
		SpanKind:      spanKindName(raw["kind"]),
		StatusCode:    statusCodeName(raw["status_code"]),
		StatusMessage: raw["status_message"],
	}

	// Timestamp from start_time_unix_nano (preferred) or _time.
	if ns, err := strconv.ParseInt(raw["start_time_unix_nano"], 10, 64); err == nil {
		row.Timestamp = ns
	} else if t, err := time.Parse(time.RFC3339Nano, raw["_time"]); err == nil {
		row.Timestamp = t.UnixNano()
	}

	// Duration: stored as nanoseconds string in VictoriaTraces.
	if d, err := strconv.ParseInt(raw["duration"], 10, 64); err == nil {
		row.Duration = d
	}

	// TraceFlags from "flags" field.
	if f, err := strconv.ParseUint(raw["flags"], 10, 32); err == nil {
		// Flags in VictoriaTraces encodes isSampled in bit 8; OTel TraceFlags uses bit 0.
		// Extract W3C sampled flag from the high byte and map to TraceFlags bit 0.
		row.TraceFlags = uint32(f>>8) & 0x01
	}

	// Platform-reserved columns (spec F-9) from resource_attr:* fields.
	row.ClusterNamespace = raw["resource_attr:k8s.cluster.namespace"]
	row.Namespace = raw["resource_attr:k8s.namespace.name"]
	row.Pod = raw["resource_attr:k8s.pod.name"]
	row.Node = raw["resource_attr:k8s.node.name"]

	// ServiceName from resource_attr:service.name.
	row.ServiceName = raw["resource_attr:service.name"]

	// resource_attr:* → ResourceAttributes (excluding promoted columns).
	promotedResourceAttrs := map[string]bool{
		"resource_attr:k8s.cluster.name":      true,
		"resource_attr:k8s.cluster.namespace": true,
		"resource_attr:k8s.namespace.name":    true,
		"resource_attr:k8s.pod.name":          true,
		"resource_attr:k8s.node.name":         true,
		"resource_attr:service.name":          true,
	}
	resAttrs := make(map[string]string)
	spanAttrs := make(map[string]string)

	// Reserved top-level fields to exclude from attribute maps.
	reservedTop := map[string]bool{
		"_time": true, "_msg": true, "_stream": true, "_stream_id": true,
		"trace_id": true, "span_id": true, "parent_span_id": true,
		"name": true, "kind": true, "duration": true, "flags": true,
		"start_time_unix_nano": true, "end_time_unix_nano": true,
		"status_code": true, "status_message": true,
		"scope_name": true, "scope_version": true,
	}

	for k, v := range raw {
		if v == "" {
			continue
		}
		if strings.HasPrefix(k, "resource_attr:") {
			if !promotedResourceAttrs[k] {
				// Strip the prefix for a cleaner attribute key.
				resAttrs[strings.TrimPrefix(k, "resource_attr:")] = v
			}
		} else if strings.HasPrefix(k, "span_attr:") {
			spanAttrs[strings.TrimPrefix(k, "span_attr:")] = v
		} else if !reservedTop[k] {
			// Other top-level fields (scope_name etc.) go into span attrs.
			spanAttrs[k] = v
		}
	}

	// Scope fields.
	if v := raw["scope_name"]; v != "" {
		spanAttrs["otel.scope.name"] = v
	}
	if v := raw["scope_version"]; v != "" {
		spanAttrs["otel.scope.version"] = v
	}

	if len(resAttrs) > 0 {
		row.ResourceAttributes = resAttrs
	}
	if len(spanAttrs) > 0 {
		row.SpanAttributes = spanAttrs
	}
	return row
}
