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
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/k0rdent/kof/kof-operator/internal/vlogs"
)

const vlogsSource = "VictoriaLogs"

// SourceUnavailableError is an alias for vlogs.SourceUnavailableError so that
// existing callers within this package do not need to change their import paths.
type SourceUnavailableError = vlogs.SourceUnavailableError

// VLogsClient queries VictoriaLogs over its HTTP API.
type VLogsClient struct {
	*vlogs.BaseClient
}

// NewVLogsClient creates a new VictoriaLogs client.
func NewVLogsClient(baseURL string) *VLogsClient {
	return &VLogsClient{
		BaseClient: vlogs.NewBaseClient(baseURL, 5*time.Minute),
	}
}

// QueryEvents opens a streaming HTTP query against VictoriaLogs and returns
// the raw response body. The caller is responsible for closing the returned
// ReadCloser. The body is JSONL: one flat JSON object per line.
//
// Stream is virtual and is not used as a query filter. The platform stream
// corresponds to records where tenant is absent or equals "PLATFORM"; all
// other tenants belong to the tenant-audit-log stream.
//
// VictoriaLogs HTTP select API:
//
//	GET /select/logsql/query?query=<logsql>&start=<rfc3339>&end=<rfc3339>
func (c *VLogsClient) QueryEvents(
	ctx context.Context,
	tenant string,
	start, end time.Time,
) (io.ReadCloser, error) {
	params := url.Values{}
	params.Set("query", buildLogsQL(tenant))
	params.Set("start", start.UTC().Format(time.RFC3339))
	params.Set("end", end.UTC().Format(time.RFC3339))
	// Large limit; real deployments should paginate if volumes are huge.
	params.Set("limit", "1000000")

	return c.QueryStream(ctx, vlogsSource, "/select/logsql/query", params)
}

// DiscoverTenants returns all distinct non-platform tenant values found in
// VictoriaLogs within [start, end). Records with an absent or empty tenant
// field, or with tenant == "PLATFORM", belong to the platform-audit-log
// stream and are excluded from the result. Used when TENANTS is not
// configured.
func (c *VLogsClient) DiscoverTenants(
	ctx context.Context,
	start, end time.Time,
) ([]string, error) {
	all, err := c.DiscoverFieldValues(ctx, vlogsSource, "tenant", "*", start, end)
	if err != nil {
		return nil, err
	}
	// Exclude empty tenant and the PLATFORM sentinel — those records belong
	// to the platform-audit-log stream, not the tenant-audit-log stream.
	tenants := all[:0]
	for _, v := range all {
		if v != "" && v != TenantPlatform {
			tenants = append(tenants, v)
		}
	}
	return tenants, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// buildLogsQL constructs a LogsQL expression that selects audit events for
// the given tenant.
//
// Stream is virtual and is not stored as a label in VictoriaLogs. The stream
// is derived from the tenant value:
//   - tenant absent or equal to "PLATFORM" → platform-audit-log stream
//   - any other value                       → tenant-audit-log stream
func buildLogsQL(tenant string) string {
	if tenant == TenantPlatform {
		// Platform stream: records where tenant is "PLATFORM" or absent/empty.
		return `tenant:"PLATFORM" OR tenant:""`
	}
	return fmt.Sprintf(`tenant:%q`, tenant)
}

// knownFlatKeys is the set of dot-notation keys that flatToAuditEvent maps
// onto declared AuditEvent fields. Any key absent from this set is treated as
// an extra field and preserved verbatim in AuditEvent.Extra.
var knownFlatKeys = map[string]struct{}{
	// VictoriaLogs internal fields — not part of the audit schema.
	"_time":      {},
	"_stream_id": {},
	"_msg":       {},
	// Mapped to AuditEvent.Time.
	"time": {},
	// Mapped to AuditEvent.Tenant.
	"tenant": {},
	// Top-level scalar fields.
	"audit_policy_version":         {},
	"cluster_audit_policy_version": {},
	"correlation_id":               {},
	"event_id":                     {},
	"justification":                {},
	"schema_version":               {},
	// Nested: Action.
	"action.outcome": {},
	"action.verb":    {},
	// Nested: Actor.
	"actor.id":   {},
	"actor.type": {},
	// Nested: Authorization.
	"authorization.decision": {},
	"authorization.policy":   {},
	"authorization.reason":   {},
	// Nested: Source.
	"source.ip":         {},
	"source.origin":     {},
	"source.user_agent": {},
	// Nested: Target.
	"target.apiVersion":  {},
	"target.cluster":     {},
	"target.kind":        {},
	"target.name":        {},
	"target.namespace":   {},
	"target.subresource": {},
}

// flatToAuditEvent converts a flat VictoriaLogs dot-notation map into an AuditEvent.
// VictoriaLogs stores {"action":{"verb":"create"}} as {"action.verb":"create"}.
// Returns an error if the event timestamp cannot be parsed, since a zero-value
// timestamp would break audit integrity and ordering.
func flatToAuditEvent(f map[string]string) (AuditEvent, error) {
	t := msTime{}
	// Prefer the application-level "time" field over VLogs "_time".
	if raw := f["time"]; raw != "" {
		if err := t.UnmarshalJSON([]byte(`"` + raw + `"`)); err != nil {
			return AuditEvent{}, fmt.Errorf("parse event time %q: %w", raw, err)
		}
	} else if raw := f["_time"]; raw != "" {
		if err := t.UnmarshalJSON([]byte(`"` + raw + `"`)); err != nil {
			return AuditEvent{}, fmt.Errorf("parse event _time %q: %w", raw, err)
		}
	}
	return AuditEvent{
		Action: Action{
			Outcome: f["action.outcome"],
			Verb:    f["action.verb"],
		},
		Actor: Actor{
			ID:   f["actor.id"],
			Type: f["actor.type"],
		},
		AuditPolicyVersion: f["audit_policy_version"],
		Authorization: Authorization{
			Decision: f["authorization.decision"],
			Policy:   f["authorization.policy"],
			Reason:   f["authorization.reason"],
		},
		ClusterAuditPolicyVersion: f["cluster_audit_policy_version"],
		CorrelationID:             f["correlation_id"],
		EventID:                   f["event_id"],
		Justification:             f["justification"],
		SchemaVersion:             f["schema_version"],
		Source: Source{
			IP:        f["source.ip"],
			Origin:    f["source.origin"],
			UserAgent: f["source.user_agent"],
		},
		Target: Target{
			APIVersion:  f["target.apiVersion"],
			Cluster:     f["target.cluster"],
			Kind:        f["target.kind"],
			Name:        f["target.name"],
			Namespace:   f["target.namespace"],
			Subresource: f["target.subresource"],
		},
		Tenant: tenantOrPlatform(f["tenant"]),
		Time:   t,
		Extra:  unflattenExtra(f, knownFlatKeys),
	}, nil
}

// unflattenExtra converts dot-notation keys absent from known back into a
// nested map[string]json.RawMessage for AuditEvent.Extra.
// e.g. {"custom.field": "v"} → {"custom": {"field": "v"}} as raw JSON.
func unflattenExtra(f map[string]string, known map[string]struct{}) map[string]json.RawMessage {
	nested := map[string]interface{}{}
	for k, v := range f {
		if _, ok := known[k]; ok {
			continue
		}
		setNested(nested, strings.Split(k, "."), v)
	}
	if len(nested) == 0 {
		return nil
	}
	result := make(map[string]json.RawMessage, len(nested))
	for k, v := range nested {
		b, err := json.Marshal(v)
		if err != nil {
			continue
		}
		result[k] = b
	}
	return result
}

// setNested assigns value at the given key path within a nested
// map[string]interface{} tree, creating intermediate maps as needed.
func setNested(m map[string]interface{}, parts []string, value string) {
	if len(parts) == 1 {
		m[parts[0]] = value
		return
	}
	sub, ok := m[parts[0]].(map[string]interface{})
	if !ok {
		sub = map[string]interface{}{}
		m[parts[0]] = sub
	}
	setNested(sub, parts[1:], value)
}

// tenantOrPlatform returns TenantPlatform when tenant is empty, and tenant
// otherwise. An absent or empty tenant field means the record belongs to the
// platform-audit-log stream.
func tenantOrPlatform(tenant string) string {
	if tenant == "" {
		return TenantPlatform
	}
	return tenant
}
