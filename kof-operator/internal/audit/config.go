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
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all exporter configuration sourced from environment variables.
type Config struct {
	// VictoriaLogs
	VLogsURL string // VLOGS_URL: base URL of VictoriaLogs, e.g. http://vlselect:9471

	// S3-compatible storage
	S3Endpoint     string // S3_ENDPOINT
	S3Bucket       string // S3_BUCKET
	S3Prefix       string // S3_PREFIX  (optional, no trailing slash)
	S3AccessKey    string // S3_ACCESS_KEY (optional; uses default AWS credential chain when empty)
	S3SecretKey    string // S3_SECRET_KEY (optional; uses default AWS credential chain when empty)
	S3Region       string // S3_REGION   (default: us-east-1)
	S3UsePathStyle bool   // S3_USE_PATH_STYLE (default: true)
	S3ForceHTTP    bool   // S3_FORCE_HTTP: skip TLS verification (dev only)

	// Compliance
	ComplianceMode bool // COMPLIANCE_MODE: if true, WORM/object-lock is required

	// KMS signing
	// KMS_KEY_ID:     key reference passed to the Signer implementation.
	//   - For the built-in LocalSigner: base64-encoded HMAC key material.
	//   - For AWS KMS: the key ARN / alias.
	KMSKeyID string

	// Export behaviour
	// STREAMS: comma-separated list of streams to export.
	// Default: tenant-audit-log,platform-audit-log
	Streams []string

	// TENANTS: comma-separated list of tenant IDs for tenant-audit-log.
	// If empty, tenants are auto-discovered from VictoriaLogs each run.
	Tenants []string

	// EXPORT_DELAY: how long after window_end to wait before exporting,
	// to absorb late/out-of-order events. Default: 5m.
	// Must equal the cron schedule offset so that (run_time - ExportDelay)
	// lands exactly on the hour boundary (e.g. schedule "5 * * * *" + 5m delay).
	ExportDelay time.Duration

	// CATCHUP_HOURS: how many hours back to look for un-exported windows.
	// Default: 24
	CatchUpHours int

	// Producer metadata included in the manifest.
	ProducerName    string // PRODUCER_NAME
	ProducerVersion string // PRODUCER_VERSION
}

// LoadConfig reads configuration from environment variables.
func LoadConfig() (*Config, error) {
	cfg := &Config{
		VLogsURL:        envOrDefault("VLOGS_URL", "http://vlselect:9471"),
		S3Endpoint:      envOrDefault("S3_ENDPOINT", ""),
		S3Bucket:        envOrDefault("S3_BUCKET", ""),
		S3Prefix:        envOrDefault("S3_PREFIX", "audit"),
		S3AccessKey:     envOrDefault("S3_ACCESS_KEY", ""),
		S3SecretKey:     envOrDefault("S3_SECRET_KEY", ""),
		S3Region:        envOrDefault("S3_REGION", "us-east-1"),
		S3UsePathStyle:  envBool("S3_USE_PATH_STYLE", true),
		S3ForceHTTP:     envBool("S3_FORCE_HTTP", false),
		ComplianceMode:  envBool("COMPLIANCE_MODE", false),
		KMSKeyID:        envOrDefault("KMS_KEY_ID", "local-dev-key"),
		ExportDelay:     envDuration("EXPORT_DELAY", 5*time.Minute),
		CatchUpHours:    envInt("CATCHUP_HOURS", 24),
		ProducerName:    envOrDefault("PRODUCER_NAME", "audit-logs-exporter"),
		ProducerVersion: envOrDefault("PRODUCER_VERSION", "v0.1.0"),
	}

	// Streams
	streamsRaw := envOrDefault("STREAMS", StreamTenantAuditLog+","+StreamPlatformAuditLog)
	for _, s := range strings.Split(streamsRaw, ",") {
		if s = strings.TrimSpace(s); s != "" {
			cfg.Streams = append(cfg.Streams, s)
		}
	}
	if len(cfg.Streams) == 0 {
		return nil, fmt.Errorf("STREAMS must not be empty")
	}

	// Tenants (optional; auto-discovered when empty)
	if raw := os.Getenv("TENANTS"); raw != "" {
		for _, t := range strings.Split(raw, ",") {
			if t = strings.TrimSpace(t); t != "" {
				cfg.Tenants = append(cfg.Tenants, t)
			}
		}
	}

	// Required fields
	if cfg.S3Bucket == "" {
		return nil, fmt.Errorf("S3_BUCKET is required")
	}
	// S3_ACCESS_KEY and S3_SECRET_KEY are optional: when both are set, static
	// credentials are used; otherwise the default AWS credential chain applies
	// (environment variables, shared config, IRSA, EC2 instance metadata, etc.).
	// Both must be provided together — supplying only one is a configuration error.
	if (cfg.S3AccessKey == "") != (cfg.S3SecretKey == "") {
		return nil, fmt.Errorf("S3_ACCESS_KEY and S3_SECRET_KEY must both be set or both be absent")
	}

	return cfg, nil
}

// producer returns the combined producer string embedded in manifests.
func (c *Config) producer() string {
	return c.ProducerName + "/" + c.ProducerVersion
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envBool(key string, def bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return b
}

func envInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func envDuration(key string, def time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return def
	}
	return d
}
