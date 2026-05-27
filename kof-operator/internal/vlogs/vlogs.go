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

package vlogs

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

// SourceUnavailableError indicates that an upstream telemetry source (e.g.
// VictoriaLogs) could not be reached or returned an error response.
// Callers use this to distinguish "no data" from "source is down".
type SourceUnavailableError struct {
	// Source is a human-readable label for the upstream service.
	Source string
	Cause  error
}

func (e *SourceUnavailableError) Error() string {
	if e.Source != "" {
		return e.Source + " source unavailable: " + e.Cause.Error()
	}
	return "source unavailable: " + e.Cause.Error()
}

func (e *SourceUnavailableError) Unwrap() error { return e.Cause }

// BaseClient holds the HTTP connection state shared by all VictoriaLogs
// client implementations. Embed this struct and call its helpers instead of
// duplicating the HTTP boilerplate.
type BaseClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewBaseClient creates a BaseClient with a custom timeout.
func NewBaseClient(baseURL string, timeout time.Duration) *BaseClient {
	return &BaseClient{
		BaseURL:    strings.TrimRight(baseURL, "/"),
		HTTPClient: &http.Client{Timeout: timeout},
	}
}

// QueryStream issues a GET request to the given VictoriaLogs endpoint with the
// supplied query parameters and returns the response body for streaming.
// The caller is responsible for closing the returned ReadCloser.
// A non-200 status or a connection error is wrapped in SourceUnavailableError.
func (c *BaseClient) QueryStream(
	ctx context.Context,
	sourceName string,
	endpoint string,
	params url.Values,
) (io.ReadCloser, error) {
	reqURL := c.BaseURL + endpoint + "?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build VLogs request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, &SourceUnavailableError{Source: sourceName, Cause: err}
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		_ = resp.Body.Close()
		return nil, &SourceUnavailableError{
			Source: sourceName,
			Cause:  fmt.Errorf("HTTP %d: %s", resp.StatusCode, body),
		}
	}
	return resp.Body, nil
}

// DiscoverFieldValues queries /select/logsql/field_values and returns all
// non-empty distinct values for the given field within [start, end).
// It handles both the JSON-array response format (newer VictoriaLogs) and the
// NDJSON fallback (older versions).
func (c *BaseClient) DiscoverFieldValues(
	ctx context.Context,
	sourceName string,
	field, query string,
	start, end time.Time,
) ([]string, error) {
	params := url.Values{}
	params.Set("field", field)
	params.Set("query", query)
	params.Set("start", start.UTC().Format(time.RFC3339))
	params.Set("end", end.UTC().Format(time.RFC3339))

	reqURL := c.BaseURL + "/select/logsql/field_values?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build field_values request: %w", err)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, &SourceUnavailableError{Source: sourceName, Cause: err}
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, &SourceUnavailableError{
			Source: sourceName,
			Cause:  fmt.Errorf("field_values HTTP %d: %s", resp.StatusCode, body),
		}
	}

	type fieldValue struct {
		Value string `json:"value"`
	}
	type fieldValuesResp struct {
		Values []fieldValue `json:"values"`
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read field_values response: %w", err)
	}

	// Try JSON-array format first (newer VictoriaLogs).
	var result fieldValuesResp
	if err := json.Unmarshal(body, &result); err == nil {
		values := make([]string, 0, len(result.Values))
		for _, fv := range result.Values {
			if fv.Value != "" {
				values = append(values, fv.Value)
			}
		}
		return values, nil
	}

	// Fallback: NDJSON format (older VictoriaLogs versions).
	var values []string
	scanner := bufio.NewScanner(strings.NewReader(string(body)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var fv fieldValue
		if err := json.Unmarshal([]byte(line), &fv); err != nil {
			continue
		}
		if fv.Value != "" {
			values = append(values, fv.Value)
		}
	}
	return values, scanner.Err()
}
