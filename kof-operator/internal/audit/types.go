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
	"encoding/json"
	"fmt"
	"time"
)

// Stream identifiers as defined in the spec.
const (
	StreamTenantAuditLog   = "tenant-audit-log"
	StreamPlatformAuditLog = "platform-audit-log"

	// TenantPlatform is the tenant value used for the platform audit log stream.
	TenantPlatform = "PLATFORM"

	ManifestVersion      = "1"
	EventSchemaVersion   = "1"
	SigningAlgorithmHMAC = "HMAC-SHA256"
)

// msTime wraps time.Time and serialises to UTC millisecond precision.
type msTime struct{ time.Time }

func (t msTime) MarshalJSON() ([]byte, error) {
	return []byte(`"` + t.UTC().Format("2006-01-02T15:04:05.000Z07:00") + `"`), nil
}

func (t *msTime) UnmarshalJSON(b []byte) error {
	if len(b) < 2 {
		return fmt.Errorf("invalid time: %s", b)
	}
	s := string(b[1 : len(b)-1])
	for _, layout := range []string{
		"2006-01-02T15:04:05.000Z07:00",
		"2006-01-02T15:04:05Z07:00",
		time.RFC3339Nano,
		time.RFC3339,
	} {
		if parsed, err := time.Parse(layout, s); err == nil {
			t.Time = parsed.UTC()
			return nil
		}
	}
	return fmt.Errorf("cannot parse time %q", s)
}

// ---------------------------------------------------------------------------
// Audit event schema (spec §Event schema). Fields declared alphabetically
// so canonical JSON marshalling (struct field order) is alphabetical.
// ---------------------------------------------------------------------------

// Action describes the operation performed.
type Action struct {
	Outcome string `json:"outcome"` // success | failure | denied
	Verb    string `json:"verb"`
}

// Actor describes who performed the action.
type Actor struct {
	ID   string `json:"id"`
	Type string `json:"type"` // user | service_account | system | operator-staff | automation
}

// Authorization holds the authorization decision.
type Authorization struct {
	Decision string `json:"decision"`
	Policy   string `json:"policy"`
	Reason   string `json:"reason"` // populated only on deny
}

// Source describes the request origin.
type Source struct {
	IP        string `json:"ip"`
	Origin    string `json:"origin"` // ui | cli | api | terraform | flux | argo | internal_controller | unknown
	UserAgent string `json:"user_agent"`
}

// Target describes the Kubernetes resource affected.
type Target struct {
	APIVersion  string `json:"apiVersion"`
	Cluster     string `json:"cluster"`
	Kind        string `json:"kind"`
	Name        string `json:"name"`
	Namespace   string `json:"namespace"`
	Subresource string `json:"subresource"`
}

// AuditEvent is one line in data.jsonl.gz. Fields are alphabetical to
// produce canonical JSON when marshalled.
//
// Extra captures any fields present in the source event that are not part of
// the declared schema, so arbitrary audit-log data is never silently dropped.
type AuditEvent struct {
	Action                    Action        `json:"action"`
	Actor                     Actor         `json:"actor"`
	AuditPolicyVersion        string        `json:"audit_policy_version"`
	Authorization             Authorization `json:"authorization"`
	ClusterAuditPolicyVersion string        `json:"cluster_audit_policy_version"`
	CorrelationID             string        `json:"correlation_id"`
	EventID                   string        `json:"event_id"` // UUIDv7
	Justification             string        `json:"justification"`
	SchemaVersion             string        `json:"schema_version"`
	Source                    Source        `json:"source"`
	Target                    Target        `json:"target"`
	Tenant                    string        `json:"tenant"`
	Time                      msTime        `json:"time"`

	// Extra holds fields from the source event that are not part of the
	// declared schema. They are inlined into the JSON output by MarshalJSON.
	Extra map[string]json.RawMessage `json:"-"`
}

// auditEventFields is a type alias used by MarshalJSON to avoid infinite
// recursion when marshalling the declared fields of AuditEvent.
type auditEventFields AuditEvent

// MarshalJSON serialises the declared fields followed by any extra fields
// captured in Extra, ensuring unknown source data is preserved in output.
// Known fields always take precedence over Extra keys with the same name.
func (e AuditEvent) MarshalJSON() ([]byte, error) {
	base, err := json.Marshal(auditEventFields(e))
	if err != nil {
		return nil, err
	}
	if len(e.Extra) == 0 {
		return base, nil
	}
	// Merge extra keys into the base object; known fields win on collision.
	var m map[string]json.RawMessage
	if err := json.Unmarshal(base, &m); err != nil {
		return nil, err
	}
	for k, v := range e.Extra {
		if _, exists := m[k]; !exists {
			m[k] = v
		}
	}
	return json.Marshal(m)
}

// ---------------------------------------------------------------------------
// Manifest schema (spec §Manifest schema). Fields alphabetical.
// ---------------------------------------------------------------------------

// DataFileMeta describes the exported data file.
type DataFileMeta struct {
	Name      string `json:"name"`
	SHA256    string `json:"sha256"`
	SizeBytes int64  `json:"size_bytes"`
}

// Manifest is written as manifest.json. Fields are alphabetical.
type Manifest struct {
	AuditPolicyVersions        []string     `json:"audit_policy_versions"`
	ClusterAuditPolicyVersions []string     `json:"cluster_audit_policy_versions"`
	CreatedAt                  msTime       `json:"created_at"`
	DataFile                   DataFileMeta `json:"data_file"`
	EventCount                 int          `json:"event_count"`
	EventSchemaVersion         string       `json:"event_schema_version"`
	KMSKeyID                   string       `json:"kms_key_id"`
	ManifestVersion            string       `json:"manifest_version"`
	PreviousManifestSHA256     string       `json:"previous_manifest_sha256"`
	Producer                   string       `json:"producer"`
	SigningAlgorithm           string       `json:"signing_algorithm"`
	Stream                     string       `json:"stream"`
	Tenant                     string       `json:"tenant"`
	WindowEnd                  msTime       `json:"window_end"`
	WindowStart                msTime       `json:"window_start"`
}

// ExportWindow is one (stream, tenant, hour) tuple to export.
type ExportWindow struct {
	Stream string
	Tenant string
	Start  time.Time // inclusive, top of hour UTC
	End    time.Time // exclusive, top of next hour UTC
}

// S3Key returns the S3 key prefix for this window.
// Layout: <prefix>/<stream>/<tenant>/YYYY/MM/DD/HH/
func (w ExportWindow) S3KeyPrefix(prefix string) string {
	if prefix != "" {
		return fmt.Sprintf("%s/%s/%s/%s/",
			prefix, w.Stream, w.Tenant,
			w.Start.UTC().Format("2006/01/02/15"))
	}
	return fmt.Sprintf("%s/%s/%s/",
		w.Stream, w.Tenant,
		w.Start.UTC().Format("2006/01/02/15"))
}
