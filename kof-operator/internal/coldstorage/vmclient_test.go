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
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Real NDJSON lines captured from vmselect-cluster (/select/0/prometheus/api/v1/export)
// on the mothership cluster running in the CI kind environment.

// metricsLine1 is a real metric line: go_gc_gomemlimit_bytes for the
// victoria-metrics-operator pod.  The metric has cluster, namespace, pod,
// job, and extra labels (service, instance, container, endpoint,
// clusterNamespace).  Note that the VM cluster uses the camelCase field name
// "clusterNamespace" (not "cluster_namespace"), so that field ends up in the
// Labels map rather than the promoted ClusterNamespace column.
const metricsLine1 = `{"metric":{"__name__":"go_gc_gomemlimit_bytes","namespace":"kof","cluster":"mothership","service":"victoria-metrics-operator","job":"victoria-metrics-operator","instance":"10.244.0.53:8080","pod":"victoria-metrics-operator-6c598cd4c6-vj5vx","clusterNamespace":"kcm-system","container":"operator","endpoint":"http"},"values":[9223372036854775807,9223372036854775807],"timestamps":[1779875391157,1779875421157]}`

// metricsLine2 is a second metric line with a different pod so that
// multi-line ScanVMExport can be exercised.
const metricsLine2 = `{"metric":{"__name__":"go_gc_gomemlimit_bytes","namespace":"kof","cluster":"mothership","service":"kof-collectors-opencost","job":"kof-collectors-opencost","instance":"10.244.0.105:9003","pod":"kof-collectors-opencost-5bf6d8cd6f-m7x9w","container":"kof-collectors-opencost","endpoint":"http"},"values":[9223372036854775807],"timestamps":[1779875409847]}`

// metricsLineNaN exercises the NaN / Infinity special-value code paths.
const metricsLineNaN = `{"metric":{"__name__":"test_nan","cluster":"c1"},"values":["NaN","Infinity","-Infinity",1.5],"timestamps":[1000,2000,3000,4000]}`

// metricsLineTenanted has an explicit tenant label.
const metricsLineTenanted = `{"metric":{"__name__":"cpu_seconds","tenant":"acme","cluster":"prod","namespace":"default","pod":"app-1","node":"node-1","job":"app"},"values":[0.42],"timestamps":[1779876000000]}`

var _ = Describe("ParseVMExportLine", func() {
	It("parses a real cluster metrics line into two MetricRows", func() {
		rows, err := ParseVMExportLine(metricsLine1)
		Expect(err).NotTo(HaveOccurred())
		Expect(rows).To(HaveLen(2))

		for _, r := range rows {
			Expect(r.MetricName).To(Equal("go_gc_gomemlimit_bytes"))
			Expect(r.Cluster).To(Equal("mothership"))
			Expect(r.Namespace).To(Equal("kof"))
			Expect(r.Pod).To(Equal("victoria-metrics-operator-6c598cd4c6-vj5vx"))
			Expect(r.Job).To(Equal("victoria-metrics-operator"))
			// value is int64-max represented as float
			Expect(r.Value).To(BeNumerically(">", 9.22e18))
			// ClusterNamespace not promoted (field is "clusterNamespace", not "cluster_namespace")
			Expect(r.ClusterNamespace).To(BeEmpty())
			// Non-promoted labels end up in Labels map
			Expect(r.Labels).To(HaveKey("clusterNamespace"))
			Expect(r.Labels["clusterNamespace"]).To(Equal("kcm-system"))
			Expect(r.Labels).To(HaveKey("service"))
			Expect(r.Labels).To(HaveKey("container"))
			Expect(r.Labels).To(HaveKey("endpoint"))
			Expect(r.Labels).To(HaveKey("instance"))
		}

		Expect(rows[0].Timestamp).To(Equal(int64(1779875391157)))
		Expect(rows[1].Timestamp).To(Equal(int64(1779875421157)))
	})

	It("promotes tenant and cluster_namespace when present", func() {
		rows, err := ParseVMExportLine(metricsLineTenanted)
		Expect(err).NotTo(HaveOccurred())
		Expect(rows).To(HaveLen(1))
		r := rows[0]

		Expect(r.Tenant).To(Equal("acme"))
		Expect(r.Cluster).To(Equal("prod"))
		Expect(r.Namespace).To(Equal("default"))
		Expect(r.Pod).To(Equal("app-1"))
		Expect(r.Node).To(Equal("node-1"))
		Expect(r.Job).To(Equal("app"))
		// No extra labels.
		Expect(r.Labels).To(BeNil())
	})

	It("handles NaN, Infinity, -Infinity values without error", func() {
		rows, err := ParseVMExportLine(metricsLineNaN)
		Expect(err).NotTo(HaveOccurred())
		// NaN, Inf, -Inf, 1.5 → all four should be parsed
		Expect(rows).To(HaveLen(4))
	})

	It("returns nil for an empty line", func() {
		rows, err := ParseVMExportLine("   ")
		Expect(err).NotTo(HaveOccurred())
		Expect(rows).To(BeNil())
	})

	It("returns an error for invalid JSON", func() {
		_, err := ParseVMExportLine(`{bad json`)
		Expect(err).To(HaveOccurred())
	})

	It("returns an error when values and timestamps lengths differ", func() {
		bad := `{"metric":{"__name__":"x"},"values":[1,2],"timestamps":[1000]}`
		_, err := ParseVMExportLine(bad)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("mismatch"))
	})
})

var _ = Describe("ScanVMExport", func() {
	It("collects rows from two NDJSON lines", func() {
		input := metricsLine1 + "\n" + metricsLine2 + "\n"
		var allRows []MetricRow
		err := ScanVMExport(strings.NewReader(input), func(rows []MetricRow) error {
			allRows = append(allRows, rows...)
			return nil
		})
		Expect(err).NotTo(HaveOccurred())
		// line1 → 2 rows, line2 → 1 row
		Expect(allRows).To(HaveLen(3))
	})

	It("skips blank lines without error", func() {
		input := "\n" + metricsLine1 + "\n\n"
		var count int
		err := ScanVMExport(strings.NewReader(input), func(rows []MetricRow) error {
			count += len(rows)
			return nil
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(count).To(Equal(2))
	})

	It("returns an error from the callback", func() {
		err := ScanVMExport(strings.NewReader(metricsLine1), func(_ []MetricRow) error {
			return errTest("callback error")
		})
		Expect(err).To(MatchError("callback error"))
	})

	It("returns an error on malformed JSON", func() {
		err := ScanVMExport(strings.NewReader(`{bad`), func(_ []MetricRow) error {
			return nil
		})
		Expect(err).To(HaveOccurred())
	})
})

// errTest is a simple error type for test callbacks.
type errTest string

func (e errTest) Error() string { return string(e) }
