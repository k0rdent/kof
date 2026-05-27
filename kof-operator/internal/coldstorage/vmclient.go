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
	"strings"
	"time"
)

// VMClient queries VictoriaMetrics over its HTTP API.
type VMClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewVMClient creates a new VictoriaMetrics client.
func NewVMClient(baseURL string) *VMClient {
	return &VMClient{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{Timeout: 10 * time.Minute},
	}
}

// ExportMetrics opens a streaming HTTP export from VictoriaMetrics and returns
// the raw response body. The caller is responsible for closing the returned
// ReadCloser.  The body is NDJSON: one JSON object per line.
//
// Each line has the form:
//
//	{"metric":{"__name__":"...","label":"value",...},"values":[...],"timestamps":[...]}
//
// Ref: https://docs.victoriametrics.com/victoriametrics/url-examples/#apiv1export
func (c *VMClient) ExportMetrics(
	ctx context.Context,
	tenant, cluster string,
	start, end time.Time,
) (io.ReadCloser, error) {
	// Build a selector. We always filter by cluster; we only add a tenant
	// filter when the tenant label is known to be present in the data.
	// In multi-tenant setups (vmauth/VMUser), both labels are present.
	// In single-cluster dev setups the cluster label alone is sufficient.
	var selector string
	if tenant != "" {
		selector = fmt.Sprintf(`{tenant=%q,cluster=%q}`, tenant, cluster)
	} else {
		selector = fmt.Sprintf(`{cluster=%q}`, cluster)
	}

	params := url.Values{}
	params.Set("match[]", selector)
	params.Set("start", start.UTC().Format(time.RFC3339))
	params.Set("end", end.UTC().Format(time.RFC3339))
	params.Set("format", "jsonl") // newline-delimited JSON (default)

	reqURL := c.baseURL + "/api/v1/export?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build VM export request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, &SourceUnavailableError{Source: "VictoriaMetrics", Cause: err}
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		_ = resp.Body.Close()
		return nil, &SourceUnavailableError{
			Source: "VictoriaMetrics",
			Cause:  fmt.Errorf("HTTP %d: %s", resp.StatusCode, body),
		}
	}
	return resp.Body, nil
}

// DiscoverTenants returns all distinct tenant label values in [start, end).
// Uses the /api/v1/label/tenant/values endpoint.
func (c *VMClient) DiscoverTenants(ctx context.Context, start, end time.Time) ([]string, error) {
	return c.discoverLabelValues(ctx, "tenant", "", start, end)
}

// DiscoverClusters returns all distinct cluster label values for the given
// tenant in [start, end). When tenant is empty or "default" (synthetic fallback),
// no tenant filter is applied — all cluster label values are returned.
func (c *VMClient) DiscoverClusters(ctx context.Context, tenant string, start, end time.Time) ([]string, error) {
	var filter string
	if tenant != "" && tenant != defaultTenant {
		filter = fmt.Sprintf(`{tenant=%q}`, tenant)
	}
	return c.discoverLabelValues(ctx, "cluster", filter, start, end)
}

// discoverLabelValues queries /api/v1/label/<name>/values with an optional
// series selector and returns the list of values.
func (c *VMClient) discoverLabelValues(
	ctx context.Context,
	label, match string,
	start, end time.Time,
) ([]string, error) {
	params := url.Values{}
	params.Set("start", start.UTC().Format(time.RFC3339))
	params.Set("end", end.UTC().Format(time.RFC3339))
	if match != "" {
		params.Set("match[]", match)
	}

	reqURL := c.baseURL + "/api/v1/label/" + label + "/values?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build label values request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, &SourceUnavailableError{Source: "VictoriaMetrics", Cause: err}
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, &SourceUnavailableError{
			Source: "VictoriaMetrics",
			Cause:  fmt.Errorf("label values HTTP %d: %s", resp.StatusCode, body),
		}
	}

	// Response: {"status":"success","data":["v1","v2",...]}
	var result struct {
		Status string   `json:"status"`
		Data   []string `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode label values response: %w", err)
	}
	if result.Status != "success" {
		return nil, fmt.Errorf("label values status %q", result.Status)
	}
	return result.Data, nil
}

// ---------------------------------------------------------------------------
// NDJSON line parsing
// ---------------------------------------------------------------------------

// ParseVMExportLine parses a single NDJSON line from the /api/v1/export endpoint
// and expands it into a slice of MetricRow (one per sample).
func ParseVMExportLine(line string) ([]MetricRow, error) {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil, nil
	}

	var el vmExportLine
	if err := json.Unmarshal([]byte(line), &el); err != nil {
		return nil, fmt.Errorf("parse VM export line: %w", err)
	}
	if len(el.Values) != len(el.Timestamps) {
		return nil, fmt.Errorf("values/timestamps length mismatch: %d vs %d",
			len(el.Values), len(el.Timestamps))
	}

	// Extract promoted columns and remaining labels.
	promoted := map[string]string{
		"__name__":          "",
		"tenant":            "",
		"cluster":           "",
		"cluster_namespace": "",
		"namespace":         "",
		"pod":               "",
		"node":              "",
		"job":               "",
	}
	for k, v := range el.Metric {
		if _, ok := promoted[k]; ok {
			promoted[k] = v
		}
	}

	labels := make(map[string]string)
	for k, v := range el.Metric {
		if _, ok := promoted[k]; !ok {
			labels[k] = v
		}
	}
	if len(labels) == 0 {
		labels = nil
	}

	rows := make([]MetricRow, 0, len(el.Values))
	for i, rawVal := range el.Values {
		v, err := parseVMValue(rawVal)
		if err != nil {
			// Skip unparseable values rather than aborting the whole series.
			continue
		}
		rows = append(rows, MetricRow{
			Timestamp:        el.Timestamps[i],
			MetricName:       promoted["__name__"],
			Value:            v,
			Tenant:           promoted["tenant"],
			Cluster:          promoted["cluster"],
			ClusterNamespace: promoted["cluster_namespace"],
			Namespace:        promoted["namespace"],
			Pod:              promoted["pod"],
			Node:             promoted["node"],
			Job:              promoted["job"],
			Labels:           labels,
		})
	}
	return rows, nil
}

// ScanVMExport reads NDJSON lines from r, calls fn for each batch of MetricRows
// parsed from a single line. Parsing errors are returned immediately.
func ScanVMExport(r io.Reader, fn func([]MetricRow) error) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 256*1024), 16*1024*1024) // up to 16 MiB per line
	for scanner.Scan() {
		rows, err := ParseVMExportLine(scanner.Text())
		if err != nil {
			return err
		}
		if len(rows) == 0 {
			continue
		}
		if err := fn(rows); err != nil {
			return err
		}
	}
	return scanner.Err()
}
