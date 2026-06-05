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

// Real NDJSON lines captured from vlselect-cluster on the mothership kind cluster.
// Fields are exactly as emitted by VictoriaLogs when KOF collectors send OTel logs.

// logsLine1 is a log record from the promxy pod in kof namespace.
// It contains: _time (RFC3339 nanoseconds), _msg, severity, service.name,
// k8s.cluster.name, k8s.cluster.namespace, k8s.namespace.name,
// k8s.node.name, k8s.pod.name, and many container / k8s attribute fields.
const logsLine1 = `{"_time":"2026-05-27T10:42:04.686507125Z","_stream_id":"0000000000000000be073271f8a99e5be5d289a41321029e","_stream":"{container.id=\"cca83b40\",k8s.cluster.name=\"mothership\"}","_msg":"10.244.0.1 - - [27/May/2026 10:42:04] \"GET /-/ready HTTP/1.1 200 11\" 0.000047","container.id":"cca83b40d66d888c1ec67b1e993c0522277097e4e68b1d97d84e8d22c9a41a22","container.image.name":"quay.io/jacksontj/promxy:v0.0.93","container.image.tag":"v0.0.93","host.name":"kcm-dev-control-plane","k8s.app.instance":"kof-mothership","k8s.cluster.name":"mothership","k8s.cluster.namespace":"kcm-system","k8s.cluster.uid":"456c04cf-db9e-4809-9928-6119bd719095","k8s.container.name":"promxy","k8s.container.restart_count":"0","k8s.deployment.name":"kof-mothership-promxy","k8s.namespace.name":"kof","k8s.node.name":"kcm-dev-control-plane","k8s.node.uid":"6a18c153-ffc6-4f9c-b610-35810ec94944","k8s.pod.name":"kof-mothership-promxy-b9cc7d66c-k79xx","k8s.pod.start_time":"2026-05-27T09:42:10Z","k8s.pod.uid":"2b59c773-3216-4ec7-86be-396735b5e6f2","k8s.replicaset.name":"kof-mothership-promxy-b9cc7d66c","k8s.replicaset.uid":"d5498f2a-d639-4c71-a479-3ca05f0cd0c9","log.file.path":"/var/log/pods/kof_kof-mothership-promxy-b9cc7d66c-k79xx_2b59c773-3216-4ec7-86be-396735b5e6f2/promxy/0.log","log.iostream":"stdout","logtag":"F","os.type":"linux","scope.name":"unknown","scope.version":"unknown","service.name":"kof-mothership-promxy","severity":"INFO"}`

// logsLine2 is a second log line from kindnet (kube-system) for multi-line tests.
const logsLine2 = `{"_time":"2026-05-27T10:42:02.344161097Z","_stream_id":"00000000000000006da6f006459ee446c03b2e0f1147a627","_stream":"{k8s.cluster.name=\"mothership\"}","_msg":"I0527 10:42:02.343920       1 main.go:297] Handling node with IPs: map[172.18.0.2:{}]","k8s.cluster.name":"mothership","k8s.cluster.namespace":"kcm-system","k8s.namespace.name":"kube-system","k8s.node.name":"kcm-dev-control-plane","k8s.pod.name":"kindnet-j6s64","service.name":"kindnet","severity":"INFO"}`

// logsLineWithTraceContext has trace_id and span_id fields.
const logsLineWithTraceContext = `{"_time":"2026-05-27T10:00:00.000000000Z","_msg":"traced operation","severity":"DEBUG","service.name":"my-svc","trace_id":"abc123","span_id":"def456","k8s.cluster.name":"test","k8s.namespace.name":"prod","k8s.pod.name":"app-pod"}`

var _ = Describe("ScanVLogsExport", func() {
	const tenant = "default"
	const cluster = "mothership"

	It("parses a real cluster log line into a LogRow", func() {
		var rows []LogRow
		err := ScanVLogsExport(strings.NewReader(logsLine1), tenant, cluster, func(batch []LogRow) error {
			rows = append(rows, batch...)
			return nil
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(rows).To(HaveLen(1))
		r := rows[0]

		// Core identity columns from parameters
		Expect(r.Tenant).To(Equal(tenant))
		Expect(r.Cluster).To(Equal(cluster))

		// Timestamp parsed from _time (RFC3339 nano): 2026-05-27T10:42:04.686507125Z
		Expect(r.TimestampNs).To(Equal(int64(1779878524686507125)))

		// Body from _msg
		Expect(r.Body).To(ContainSubstring("GET /-/ready"))

		// Severity / service
		Expect(r.SeverityText).To(Equal("INFO"))
		Expect(r.ServiceName).To(Equal("kof-mothership-promxy"))

		// Platform-reserved columns
		Expect(r.ClusterNamespace).To(Equal("kcm-system"))
		Expect(r.Namespace).To(Equal("kof"))
		Expect(r.Pod).To(Equal("kof-mothership-promxy-b9cc7d66c-k79xx"))
		Expect(r.Node).To(Equal("kcm-dev-control-plane"))

		// k8s.cluster.name and reserved fields must NOT appear in Attributes
		Expect(r.Attributes).NotTo(HaveKey("k8s.cluster.name"))
		Expect(r.Attributes).NotTo(HaveKey("_time"))
		Expect(r.Attributes).NotTo(HaveKey("_msg"))
		Expect(r.Attributes).NotTo(HaveKey("severity"))
		Expect(r.Attributes).NotTo(HaveKey("service.name"))
		Expect(r.Attributes).NotTo(HaveKey("k8s.namespace.name"))
		Expect(r.Attributes).NotTo(HaveKey("k8s.pod.name"))
		Expect(r.Attributes).NotTo(HaveKey("k8s.node.name"))

		// Non-reserved OTel fields should land in Attributes
		Expect(r.Attributes).To(HaveKey("container.id"))
		Expect(r.Attributes).To(HaveKey("k8s.container.name"))
	})

	It("parses trace context fields when present", func() {
		var rows []LogRow
		err := ScanVLogsExport(strings.NewReader(logsLineWithTraceContext), tenant, "test", func(b []LogRow) error {
			rows = append(rows, b...)
			return nil
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(rows).To(HaveLen(1))
		r := rows[0]
		Expect(r.TraceId).To(Equal("abc123"))
		Expect(r.SpanId).To(Equal("def456"))
		Expect(r.Namespace).To(Equal("prod"))
		Expect(r.Pod).To(Equal("app-pod"))
	})

	It("collects rows from two log lines", func() {
		input := logsLine1 + "\n" + logsLine2 + "\n"
		var allRows []LogRow
		err := ScanVLogsExport(strings.NewReader(input), tenant, cluster, func(b []LogRow) error {
			allRows = append(allRows, b...)
			return nil
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(allRows).To(HaveLen(2))
	})

	It("skips blank lines without error", func() {
		input := "\n" + logsLine1 + "\n\n"
		var count int
		err := ScanVLogsExport(strings.NewReader(input), tenant, cluster, func(b []LogRow) error {
			count += len(b)
			return nil
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(count).To(Equal(1))
	})

	It("skips malformed JSON lines without returning an error", func() {
		input := `{bad json}` + "\n" + logsLine2 + "\n"
		var rows []LogRow
		err := ScanVLogsExport(strings.NewReader(input), tenant, cluster, func(b []LogRow) error {
			rows = append(rows, b...)
			return nil
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(rows).To(HaveLen(1))
	})

	It("propagates callback errors", func() {
		err := ScanVLogsExport(strings.NewReader(logsLine1), tenant, cluster, func(_ []LogRow) error {
			return errTest("cb error")
		})
		Expect(err).To(MatchError("cb error"))
	})

	It("returns an empty result for empty input", func() {
		var rows []LogRow
		err := ScanVLogsExport(strings.NewReader(""), tenant, cluster, func(b []LogRow) error {
			rows = append(rows, b...)
			return nil
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(rows).To(BeEmpty())
	})
})
