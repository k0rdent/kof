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
	"math"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ExportWindow.S3KeyPrefix", func() {
	var w ExportWindow

	BeforeEach(func() {
		// 2026-05-27T10:00:00Z — top of hour as observed in cluster data
		w = ExportWindow{
			Source:  SourceMetrics,
			Tenant:  "default",
			Cluster: "mothership",
			Start:   time.Date(2026, 5, 27, 10, 0, 0, 0, time.UTC),
			End:     time.Date(2026, 5, 27, 11, 0, 0, 0, time.UTC),
		}
	})

	It("produces the correct key with no prefix", func() {
		key := w.S3KeyPrefix("")
		Expect(key).To(Equal(
			"tenant=default/cluster=mothership/dt=2026-05-27/hour=10/metrics/",
		))
	})

	It("prepends the prefix when set", func() {
		key := w.S3KeyPrefix("coldstorage")
		Expect(key).To(Equal(
			"coldstorage/tenant=default/cluster=mothership/dt=2026-05-27/hour=10/metrics/",
		))
	})

	It("uses the source field in the key", func() {
		w.Source = SourceLogs
		key := w.S3KeyPrefix("")
		Expect(key).To(ContainSubstring("/logs/"))
	})

	It("uses the source field for traces", func() {
		w.Source = SourceTraces
		key := w.S3KeyPrefix("")
		Expect(key).To(ContainSubstring("/traces/"))
	})

	It("zero-pads single-digit hours", func() {
		w.Start = time.Date(2026, 5, 27, 3, 0, 0, 0, time.UTC)
		key := w.S3KeyPrefix("")
		Expect(key).To(ContainSubstring("/hour=03/"))
	})

	It("always converts Start to UTC for the date partition", func() {
		// Even if Start has a non-UTC location, the partition should use UTC.
		est := time.FixedZone("EST", -5*60*60)
		w.Start = time.Date(2026, 5, 27, 10, 0, 0, 0, est) // 15:00 UTC
		key := w.S3KeyPrefix("")
		Expect(key).To(ContainSubstring("dt=2026-05-27/hour=15"))
	})
})

var _ = Describe("parseVMValue", func() {
	It("parses a normal float64", func() {
		v, err := parseVMValue([]byte(`3.14`))
		Expect(err).NotTo(HaveOccurred())
		Expect(v).To(BeNumerically("~", 3.14))
	})

	It("parses an integer as float64", func() {
		v, err := parseVMValue([]byte(`9223372036854775807`))
		Expect(err).NotTo(HaveOccurred())
		// int64 max — as seen in cluster go_gc_gomemlimit_bytes metric
		Expect(v).To(BeNumerically(">", 9.22e18))
	})

	It("parses zero", func() {
		v, err := parseVMValue([]byte(`0`))
		Expect(err).NotTo(HaveOccurred())
		Expect(v).To(Equal(0.0))
	})

	It(`parses "NaN"`, func() {
		v, err := parseVMValue([]byte(`"NaN"`))
		Expect(err).NotTo(HaveOccurred())
		Expect(math.IsNaN(v)).To(BeTrue())
	})

	It(`parses "Infinity"`, func() {
		v, err := parseVMValue([]byte(`"Infinity"`))
		Expect(err).NotTo(HaveOccurred())
		Expect(math.IsInf(v, 1)).To(BeTrue())
	})

	It(`parses "+Infinity"`, func() {
		v, err := parseVMValue([]byte(`"+Infinity"`))
		Expect(err).NotTo(HaveOccurred())
		Expect(math.IsInf(v, 1)).To(BeTrue())
	})

	It(`parses "-Infinity"`, func() {
		v, err := parseVMValue([]byte(`"-Infinity"`))
		Expect(err).NotTo(HaveOccurred())
		Expect(math.IsInf(v, -1)).To(BeTrue())
	})

	It("returns an error for an unrecognised string", func() {
		_, err := parseVMValue([]byte(`"bogus"`))
		Expect(err).To(HaveOccurred())
	})
})
