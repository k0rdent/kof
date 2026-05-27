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
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/k0rdent/kof/kof-operator/internal/env"
)

// Config holds all exporter configuration sourced from environment variables.
type Config struct {
	// Source endpoints
	VMURL      string // VM_URL: base URL of VictoriaMetrics, e.g. http://vmselect:8481/select/0/prometheus
	VLogsURL   string // VLOGS_URL: base URL of VictoriaLogs, e.g. http://vlselect:9471
	VTracesURL string // VTRACES_URL: base URL of VictoriaTraces LogsQL endpoint, e.g. http://vtselect:10471

	// Sources to export: comma-separated list of "metrics", "logs", "traces"
	Sources []string // SOURCES (default: metrics)

	// Tenants (optional; auto-discovered when empty)
	Tenants []string // TENANTS: comma-separated list of tenant IDs

	// Clusters (optional; auto-discovered when empty)
	Clusters []string // CLUSTERS: comma-separated list of cluster names

	// S3-compatible storage
	S3Endpoint     string // S3_ENDPOINT
	S3Bucket       string // S3_BUCKET
	S3Prefix       string // S3_PREFIX (no trailing slash; default: "telemetry")
	S3AccessKey    string // S3_ACCESS_KEY (optional; uses default AWS credential chain when empty)
	S3SecretKey    string // S3_SECRET_KEY (optional; uses default AWS credential chain when empty)
	S3Region       string // S3_REGION (default: us-east-1)
	S3UsePathStyle bool   // S3_USE_PATH_STYLE (default: true)
	S3ForceHTTP    bool   // S3_FORCE_HTTP: skip TLS verification (dev only)

	// Export behaviour
	// EXPORT_DELAY: how long after window_end to wait before exporting.
	// Default: 5m.
	ExportDelay time.Duration

	// CATCHUP_HOURS: how many hours back to look for un-exported windows.
	// Default: 24
	CatchUpHours int
}

// LoadConfig reads configuration from environment variables.
func LoadConfig() (*Config, error) {
	cfg := &Config{
		VMURL:          env.GetEnvOrDefault("VM_URL", "http://vmselect:8481/select/0/prometheus"),
		VLogsURL:       env.GetEnvOrDefault("VLOGS_URL", "http://vlselect:9471"),
		VTracesURL:     env.GetEnvOrDefault("VTRACES_URL", "http://vt-vtselect:10471"),
		S3Endpoint:     env.GetEnvOrDefault("S3_ENDPOINT", ""),
		S3Bucket:       env.GetEnvOrDefault("S3_BUCKET", ""),
		S3Prefix:       env.GetEnvOrDefault("S3_PREFIX", "telemetry"),
		S3AccessKey:    env.GetEnvOrDefault("S3_ACCESS_KEY", ""),
		S3SecretKey:    env.GetEnvOrDefault("S3_SECRET_KEY", ""),
		S3Region:       env.GetEnvOrDefault("S3_REGION", "us-east-1"),
		S3UsePathStyle: env.GetEnvBool("S3_USE_PATH_STYLE", true),
		S3ForceHTTP:    env.GetEnvBool("S3_FORCE_HTTP", false),
		ExportDelay:    env.GetEnvDuration("EXPORT_DELAY", 5*time.Minute),
		CatchUpHours:   env.GetEnvInt("CATCHUP_HOURS", 24),
	}

	// Sources (what to export)
	sourcesRaw := env.GetEnvOrDefault("SOURCES", SourceMetrics)
	for _, s := range strings.Split(sourcesRaw, ",") {
		if s = strings.TrimSpace(s); s != "" {
			cfg.Sources = append(cfg.Sources, s)
		}
	}
	if len(cfg.Sources) == 0 {
		return nil, fmt.Errorf("SOURCES must not be empty")
	}
	for _, s := range cfg.Sources {
		switch s {
		case SourceMetrics, SourceLogs, SourceTraces:
		default:
			return nil, fmt.Errorf("unknown source %q: must be one of metrics, logs, traces", s)
		}
	}

	// Tenants (optional; auto-discovered when empty)
	if raw := os.Getenv("TENANTS"); raw != "" {
		for _, t := range strings.Split(raw, ",") {
			if t = strings.TrimSpace(t); t != "" {
				cfg.Tenants = append(cfg.Tenants, t)
			}
		}
	}

	// Clusters (optional; auto-discovered when empty)
	if raw := os.Getenv("CLUSTERS"); raw != "" {
		for _, c := range strings.Split(raw, ",") {
			if c = strings.TrimSpace(c); c != "" {
				cfg.Clusters = append(cfg.Clusters, c)
			}
		}
	}

	// Required fields
	if cfg.S3Bucket == "" {
		return nil, fmt.Errorf("S3_BUCKET is required")
	}
	if (cfg.S3AccessKey == "") != (cfg.S3SecretKey == "") {
		return nil, fmt.Errorf("S3_ACCESS_KEY and S3_SECRET_KEY must both be set or both be absent")
	}

	return cfg, nil
}
