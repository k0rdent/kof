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
	"errors"
	"io"
	"time"

	"github.com/go-logr/logr"
	s3pkg "github.com/k0rdent/kof/kof-operator/internal/s3"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// stubVLogs implements vlogsQuerier with configurable DiscoverTenants responses.
// QueryEvents panics — it is not exercised by collectWindows.
type stubVLogs struct {
	tenants []string
	err     error
}

func (s *stubVLogs) DiscoverTenants(_ context.Context, _, _ time.Time) ([]string, error) {
	return s.tenants, s.err
}

func (s *stubVLogs) QueryEvents(_ context.Context, _ string, _, _ time.Time) (io.ReadCloser, error) {
	panic("not implemented")
}

// Compile-time check: *stubVLogs satisfies vlogsQuerier.
var _ vlogsQuerier = (*stubVLogs)(nil)

// Compile-time check: *VLogsClient satisfies vlogsQuerier.
var _ vlogsQuerier = (*VLogsClient)(nil)

// newTestExporter builds a minimal Exporter wired with the provided stubs.
func newTestExporter(cfg *Config, vlogs vlogsQuerier, s3stub s3pkg.RawAPI) *Exporter {
	return &Exporter{
		cfg:   cfg,
		vlogs: vlogs,
		s3:    &S3Client{Client: s3pkg.NewClientFromRaw(s3stub, cfg.S3Bucket)},
		log:   logr.Discard(),
	}
}

// baseConfig returns a minimal Config for tests.
func baseConfig(streams []string, tenants []string) *Config {
	return &Config{
		Streams:  streams,
		Tenants:  tenants,
		S3Bucket: "test-bucket",
		S3Prefix: "audit",
	}
}

var _ = Describe("Exporter.collectWindows", func() {
	// Two-hour window: H10 and H11 on 2025-01-01.
	var (
		h10 = time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
		h11 = h10.Add(time.Hour)
		h12 = h11.Add(time.Hour)
	)

	// manifestKey returns the S3 key for a window's manifest, mirroring
	// the path layout used by collectWindows.
	manifestKey := func(stream, tenant string, start time.Time, prefix string) string {
		w := ExportWindow{Stream: stream, Tenant: tenant, Start: start, End: start.Add(time.Hour)}
		return w.S3KeyPrefix(prefix) + "manifest.json"
	}

	Describe("platform-audit-log stream", func() {
		It("always uses PLATFORM tenant without calling DiscoverTenants", func() {
			cfg := baseConfig([]string{StreamPlatformAuditLog}, nil)
			e := newTestExporter(cfg, &stubVLogs{err: errors.New("must not be called")}, &stubS3{})

			windows, err := e.collectWindows(context.Background(), h10, h12)
			Expect(err).NotTo(HaveOccurred())
			Expect(windows).To(HaveLen(2))
			for _, w := range windows {
				Expect(w.Stream).To(Equal(StreamPlatformAuditLog))
				Expect(w.Tenant).To(Equal(TenantPlatform))
			}
			Expect(windows[0].Start).To(Equal(h10))
			Expect(windows[1].Start).To(Equal(h11))
		})
	})

	Describe("tenant-audit-log stream", func() {
		It("uses configured tenants without calling DiscoverTenants", func() {
			cfg := baseConfig([]string{StreamTenantAuditLog}, []string{"acme", "globex"})
			e := newTestExporter(cfg, &stubVLogs{err: errors.New("must not be called")}, &stubS3{})

			windows, err := e.collectWindows(context.Background(), h10, h12)
			Expect(err).NotTo(HaveOccurred())
			// 2 tenants × 2 hours = 4 windows
			Expect(windows).To(HaveLen(4))
			tenants := make(map[string]int)
			for _, w := range windows {
				Expect(w.Stream).To(Equal(StreamTenantAuditLog))
				tenants[w.Tenant]++
			}
			Expect(tenants).To(Equal(map[string]int{"acme": 2, "globex": 2}))
		})

		It("auto-discovers tenants when none are configured", func() {
			cfg := baseConfig([]string{StreamTenantAuditLog}, nil)
			e := newTestExporter(cfg, &stubVLogs{tenants: []string{"acme"}}, &stubS3{})

			windows, err := e.collectWindows(context.Background(), h10, h12)
			Expect(err).NotTo(HaveOccurred())
			// 1 tenant × 2 hours = 2 windows
			Expect(windows).To(HaveLen(2))
			for _, w := range windows {
				Expect(w.Tenant).To(Equal("acme"))
			}
		})

		It("returns an empty slice when DiscoverTenants finds no tenants", func() {
			cfg := baseConfig([]string{StreamTenantAuditLog}, nil)
			e := newTestExporter(cfg, &stubVLogs{tenants: nil}, &stubS3{})

			windows, err := e.collectWindows(context.Background(), h10, h12)
			Expect(err).NotTo(HaveOccurred())
			Expect(windows).To(BeEmpty())
		})

		It("propagates DiscoverTenants errors", func() {
			cfg := baseConfig([]string{StreamTenantAuditLog}, nil)
			discoverErr := errors.New("vlogs unavailable")
			e := newTestExporter(cfg, &stubVLogs{err: discoverErr}, &stubS3{})

			_, err := e.collectWindows(context.Background(), h10, h12)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("vlogs unavailable")))
		})
	})

	Describe("idempotency — already-exported windows are skipped", func() {
		It("skips a window whose manifest.json already exists in S3", func() {
			cfg := baseConfig([]string{StreamPlatformAuditLog}, nil)
			// Mark the H10 manifest as already present.
			existing := manifestKey(StreamPlatformAuditLog, TenantPlatform, h10, cfg.S3Prefix)
			s3stub := &stubS3{existingKeys: map[string]bool{existing: true}}
			e := newTestExporter(cfg, &stubVLogs{}, s3stub)

			windows, err := e.collectWindows(context.Background(), h10, h12)
			Expect(err).NotTo(HaveOccurred())
			// Only H11 window should remain.
			Expect(windows).To(HaveLen(1))
			Expect(windows[0].Start).To(Equal(h11))
		})

		It("returns empty when all windows are already exported", func() {
			cfg := baseConfig([]string{StreamPlatformAuditLog}, nil)
			existing := map[string]bool{
				manifestKey(StreamPlatformAuditLog, TenantPlatform, h10, cfg.S3Prefix): true,
				manifestKey(StreamPlatformAuditLog, TenantPlatform, h11, cfg.S3Prefix): true,
			}
			e := newTestExporter(cfg, &stubVLogs{}, &stubS3{existingKeys: existing})

			windows, err := e.collectWindows(context.Background(), h10, h12)
			Expect(err).NotTo(HaveOccurred())
			Expect(windows).To(BeEmpty())
		})

		It("returns an error and does not export when manifest check fails", func() {
			cfg := baseConfig([]string{StreamPlatformAuditLog}, nil)
			checkErr := errors.New("s3 unreachable")
			e := newTestExporter(cfg, &stubVLogs{}, &stubS3{headErr: checkErr})

			_, err := e.collectWindows(context.Background(), h10, h12)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("s3 unreachable")))
		})
	})

	Describe("multiple streams", func() {
		It("collects windows for all configured streams", func() {
			cfg := baseConfig(
				[]string{StreamPlatformAuditLog, StreamTenantAuditLog},
				[]string{"acme"},
			)
			e := newTestExporter(cfg, &stubVLogs{}, &stubS3{})

			windows, err := e.collectWindows(context.Background(), h10, h12)
			Expect(err).NotTo(HaveOccurred())
			// platform: 1 tenant × 2h = 2; tenant-audit-log: 1 tenant × 2h = 2 → total 4
			Expect(windows).To(HaveLen(4))

			streams := make(map[string]int)
			for _, w := range windows {
				streams[w.Stream]++
			}
			Expect(streams[StreamPlatformAuditLog]).To(Equal(2))
			Expect(streams[StreamTenantAuditLog]).To(Equal(2))
		})
	})

	Describe("window ordering and boundaries", func() {
		It("produces windows in chronological order", func() {
			cfg := baseConfig([]string{StreamPlatformAuditLog}, nil)
			e := newTestExporter(cfg, &stubVLogs{}, &stubS3{})

			windows, err := e.collectWindows(context.Background(), h10, h12)
			Expect(err).NotTo(HaveOccurred())
			Expect(windows).To(HaveLen(2))
			Expect(windows[0].Start).To(BeTemporally("<", windows[1].Start))
		})

		It("each window End equals the next window Start", func() {
			h13 := h10.Add(3 * time.Hour)
			cfg := baseConfig([]string{StreamPlatformAuditLog}, nil)
			e := newTestExporter(cfg, &stubVLogs{}, &stubS3{})

			windows, err := e.collectWindows(context.Background(), h10, h13)
			Expect(err).NotTo(HaveOccurred())
			Expect(windows).To(HaveLen(3))
			for i := 0; i < len(windows)-1; i++ {
				Expect(windows[i].End).To(Equal(windows[i+1].Start))
			}
		})

		It("returns empty when from equals upToEnd", func() {
			cfg := baseConfig([]string{StreamPlatformAuditLog}, nil)
			e := newTestExporter(cfg, &stubVLogs{}, &stubS3{})

			windows, err := e.collectWindows(context.Background(), h10, h10)
			Expect(err).NotTo(HaveOccurred())
			Expect(windows).To(BeEmpty())
		})
	})
})
