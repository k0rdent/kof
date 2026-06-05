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

// Real NDJSON span lines captured from vtselect on the mothership kind cluster.
// These are actual OTel spans emitted by the kof-operator via the OpenTelemetry SDK.

// tracesLine1 is a real span: an outgoing HTTP GET made by kof-operator to the
// VictoriaLogs vlstorage proxy metrics endpoint.
// Fields: trace_id, span_id, parent_span_id, name, kind=3 (CLIENT),
// start_time_unix_nano, end_time_unix_nano, duration, flags=257,
// resource_attr:* (k8s, service), span_attr:* (http.*), scope_name/scope_version.
const tracesLine1 = `{"_msg":"-","_stream":"{name=\"GET 10.96.0.1:443/api/v1/namespaces/kof/pods/kof-storage-victoria-logs-cluster-vlstorage-0:9491/proxy/metrics\",resource_attr:service.name=\"kof-operator\"}","_stream_id":"0000000000000000733542093ed1e1d2ca518bba24e637a9","_time":"2026-05-27T10:01:42.101672962Z","duration":"6019458","end_time_unix_nano":"1779876102107692420","flags":"257","kind":"3","name":"GET 10.96.0.1:443/api/v1/namespaces/kof/pods/kof-storage-victoria-logs-cluster-vlstorage-0:9491/proxy/metrics","parent_span_id":"d2e4efe260bfb927","resource_attr:container.image.name":"docker.io/library/kof-operator-controller","resource_attr:container.image.tag":"v1.10.0-rc0","resource_attr:host.name":"kcm-dev-control-plane","resource_attr:k8s.app.instance":"kof-mothership-operator","resource_attr:k8s.cluster.name":"mothership","resource_attr:k8s.cluster.uid":"456c04cf-db9e-4809-9928-6119bd719095","resource_attr:k8s.container.name":"operator","resource_attr:k8s.deployment.name":"kof-mothership-kof-operator","resource_attr:k8s.namespace.name":"kof","resource_attr:k8s.node.name":"kcm-dev-control-plane","resource_attr:k8s.node.uid":"6a18c153-ffc6-4f9c-b610-35810ec94944","resource_attr:k8s.pod.name":"kof-mothership-kof-operator-74d9d57565-8r772","resource_attr:k8s.pod.start_time":"2026-05-27T09:43:23Z","resource_attr:k8s.pod.uid":"43e5b54a-9fb8-4da4-8c7f-80402e94aa0a","resource_attr:k8s.replicaset.name":"kof-mothership-kof-operator-74d9d57565","resource_attr:k8s.replicaset.uid":"e4944b3b-7076-4c50-a16b-a9f53224982b","resource_attr:os.type":"linux","resource_attr:service.name":"kof-operator","resource_attr:telemetry.sdk.language":"go","resource_attr:telemetry.sdk.name":"opentelemetry","resource_attr:telemetry.sdk.version":"1.43.0","scope_name":"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp","scope_version":"0.68.0","span_attr:http.request.method":"GET","span_attr:http.response.status_code":"200","span_attr:network.protocol.version":"1.1","span_attr:server.address":"10.96.0.1","span_attr:url.full":"https://10.96.0.1:443/api/v1/namespaces/kof/pods/kof-storage-victoria-logs-cluster-vlstorage-0:9491/proxy/metrics","span_id":"7e3fd52994e44de2","start_time_unix_nano":"1779876102101672962","trace_id":"1163d809514711c4503f981a9634c021"}`

// tracesLine2 is a second span from the same trace — used for multi-line scans.
const tracesLine2 = `{"_msg":"-","_stream":"{name=\"GET 10.96.0.1:443/api/v1/namespaces/kof/pods/kof-storage-victoria-logs-cluster-vlselect-678d6d6c86-p9v2p:9471/proxy/metrics\",resource_attr:service.name=\"kof-operator\"}","_stream_id":"0000000000000000bb057fa7b88b8d398945dc664136ead3","_time":"2026-05-27T10:01:42.100994587Z","duration":"3119041","end_time_unix_nano":"1779876102104113628","flags":"257","kind":"3","name":"GET 10.96.0.1:443/api/v1/namespaces/kof/pods/kof-storage-victoria-logs-cluster-vlselect-678d6d6c86-p9v2p:9471/proxy/metrics","parent_span_id":"d2e4efe260bfb927","resource_attr:k8s.cluster.name":"mothership","resource_attr:k8s.namespace.name":"kof","resource_attr:k8s.node.name":"kcm-dev-control-plane","resource_attr:k8s.pod.name":"kof-mothership-kof-operator-74d9d57565-8r772","resource_attr:service.name":"kof-operator","span_attr:http.request.method":"GET","span_attr:http.response.status_code":"200","span_id":"64600d72af635ba4","start_time_unix_nano":"1779876102100994587","trace_id":"1163d809514711c4503f981a9634c021"}`

// tracesLineAllKinds exercises all span kind integer values.
const tracesLineServer = `{"name":"serve","kind":"2","start_time_unix_nano":"1000000000","duration":"500000","flags":"256","span_id":"aaa","trace_id":"bbb","resource_attr:k8s.cluster.name":"c1","resource_attr:service.name":"svc1"}`
const tracesLineInternal = `{"name":"internal","kind":"1","start_time_unix_nano":"2000000000","duration":"100","flags":"0","span_id":"ccc","trace_id":"ddd","resource_attr:k8s.cluster.name":"c1","status_code":"1","resource_attr:service.name":"svc1"}`
const tracesLineError = `{"name":"fail","kind":"3","start_time_unix_nano":"3000000000","duration":"200","flags":"256","span_id":"eee","trace_id":"fff","resource_attr:k8s.cluster.name":"c1","status_code":"2","status_message":"something went wrong","resource_attr:service.name":"svc1"}`

var _ = Describe("ScanVTracesExport", func() {
	const tenant = "default"
	const cluster = "mothership"

	It("parses a real cluster span line into a TraceRow", func() {
		var rows []TraceRow
		err := ScanVTracesExport(strings.NewReader(tracesLine1), tenant, cluster, func(batch []TraceRow) error {
			rows = append(rows, batch...)
			return nil
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(rows).To(HaveLen(1))
		r := rows[0]

		// Identity
		Expect(r.Tenant).To(Equal(tenant))
		Expect(r.Cluster).To(Equal(cluster))

		// OTel span identity fields
		Expect(r.TraceId).To(Equal("1163d809514711c4503f981a9634c021"))
		Expect(r.SpanId).To(Equal("7e3fd52994e44de2"))
		Expect(r.ParentSpanId).To(Equal("d2e4efe260bfb927"))
		Expect(r.SpanName).To(ContainSubstring("GET 10.96.0.1"))

		// kind=3 → CLIENT
		Expect(r.SpanKind).To(Equal("SPAN_KIND_CLIENT"))

		// Timestamp from start_time_unix_nano
		Expect(r.Timestamp).To(Equal(int64(1779876102101672962)))

		// Duration in nanoseconds
		Expect(r.Duration).To(Equal(int64(6019458)))

		// TraceFlags: flags=257 (binary 100000001), bit 8 set → W3C sampled=1
		// (257 >> 8) & 0x01 = 1 & 0x01 = 1
		Expect(r.TraceFlags).To(Equal(uint32(1)))

		// Platform-reserved columns from resource_attr:*
		Expect(r.Namespace).To(Equal("kof"))
		Expect(r.Node).To(Equal("kcm-dev-control-plane"))
		Expect(r.Pod).To(Equal("kof-mothership-kof-operator-74d9d57565-8r772"))
		Expect(r.ServiceName).To(Equal("kof-operator"))

		// Status unset (no status_code field in real span)
		Expect(r.StatusCode).To(Equal("STATUS_CODE_UNSET"))

		// ResourceAttributes should contain non-promoted resource attrs
		Expect(r.ResourceAttributes).To(HaveKey("container.image.name"))
		Expect(r.ResourceAttributes).To(HaveKey("telemetry.sdk.name"))
		// Promoted resource attrs must NOT be in ResourceAttributes
		Expect(r.ResourceAttributes).NotTo(HaveKey("k8s.cluster.name"))
		Expect(r.ResourceAttributes).NotTo(HaveKey("k8s.namespace.name"))
		Expect(r.ResourceAttributes).NotTo(HaveKey("service.name"))

		// SpanAttributes from span_attr:* prefix
		Expect(r.SpanAttributes).To(HaveKey("http.request.method"))
		Expect(r.SpanAttributes["http.request.method"]).To(Equal("GET"))
		Expect(r.SpanAttributes).To(HaveKey("http.response.status_code"))
		Expect(r.SpanAttributes["http.response.status_code"]).To(Equal("200"))

		// Scope fields go into SpanAttributes as otel.scope.*
		Expect(r.SpanAttributes).To(HaveKey("otel.scope.name"))
		Expect(r.SpanAttributes["otel.scope.name"]).To(ContainSubstring("otelhttp"))
		Expect(r.SpanAttributes).To(HaveKey("otel.scope.version"))
	})

	It("maps span kind integers to OTel string names", func() {
		cases := map[string]string{
			tracesLineServer:   "SPAN_KIND_SERVER",
			tracesLineInternal: "SPAN_KIND_INTERNAL",
		}
		for line, expectedKind := range cases {
			var rows []TraceRow
			err := ScanVTracesExport(strings.NewReader(line), tenant, cluster, func(b []TraceRow) error {
				rows = append(rows, b...)
				return nil
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(rows).To(HaveLen(1))
			Expect(rows[0].SpanKind).To(Equal(expectedKind), "for line: %s", line)
		}
	})

	It("maps status_code=1 to STATUS_CODE_OK", func() {
		var rows []TraceRow
		err := ScanVTracesExport(strings.NewReader(tracesLineInternal), tenant, cluster, func(b []TraceRow) error {
			rows = append(rows, b...)
			return nil
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(rows[0].StatusCode).To(Equal("STATUS_CODE_OK"))
	})

	It("maps status_code=2 to STATUS_CODE_ERROR with message", func() {
		var rows []TraceRow
		err := ScanVTracesExport(strings.NewReader(tracesLineError), tenant, cluster, func(b []TraceRow) error {
			rows = append(rows, b...)
			return nil
		})
		Expect(err).NotTo(HaveOccurred())
		r := rows[0]
		Expect(r.StatusCode).To(Equal("STATUS_CODE_ERROR"))
		Expect(r.StatusMessage).To(Equal("something went wrong"))
	})

	It("TraceFlags=0 when flags field is absent or zero", func() {
		line := `{"name":"x","kind":"1","start_time_unix_nano":"1","duration":"1","span_id":"a","trace_id":"b","resource_attr:k8s.cluster.name":"c","resource_attr:service.name":"s"}`
		var rows []TraceRow
		_ = ScanVTracesExport(strings.NewReader(line), tenant, cluster, func(b []TraceRow) error {
			rows = append(rows, b...)
			return nil
		})
		Expect(rows[0].TraceFlags).To(Equal(uint32(0)))
	})

	It("collects rows from two span lines", func() {
		input := tracesLine1 + "\n" + tracesLine2 + "\n"
		var allRows []TraceRow
		err := ScanVTracesExport(strings.NewReader(input), tenant, cluster, func(b []TraceRow) error {
			allRows = append(allRows, b...)
			return nil
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(allRows).To(HaveLen(2))
		// Both spans belong to the same trace
		Expect(allRows[0].TraceId).To(Equal(allRows[1].TraceId))
	})

	It("skips blank lines without error", func() {
		input := "\n" + tracesLine1 + "\n\n"
		var count int
		err := ScanVTracesExport(strings.NewReader(input), tenant, cluster, func(b []TraceRow) error {
			count += len(b)
			return nil
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(count).To(Equal(1))
	})

	It("skips malformed JSON lines without returning an error", func() {
		input := `{bad}` + "\n" + tracesLine2 + "\n"
		var rows []TraceRow
		err := ScanVTracesExport(strings.NewReader(input), tenant, cluster, func(b []TraceRow) error {
			rows = append(rows, b...)
			return nil
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(rows).To(HaveLen(1))
	})

	It("propagates callback errors", func() {
		err := ScanVTracesExport(strings.NewReader(tracesLine1), tenant, cluster, func(_ []TraceRow) error {
			return errTest("trace cb error")
		})
		Expect(err).To(MatchError("trace cb error"))
	})

	It("returns empty result for empty input", func() {
		var rows []TraceRow
		err := ScanVTracesExport(strings.NewReader(""), tenant, cluster, func(b []TraceRow) error {
			rows = append(rows, b...)
			return nil
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(rows).To(BeEmpty())
	})
})

var _ = Describe("spanKindName", func() {
	DescribeTable("maps integer strings to OTel span kind names",
		func(input, expected string) {
			Expect(spanKindName(input)).To(Equal(expected))
		},
		Entry("1 → INTERNAL", "1", "SPAN_KIND_INTERNAL"),
		Entry("2 → SERVER", "2", "SPAN_KIND_SERVER"),
		Entry("3 → CLIENT", "3", "SPAN_KIND_CLIENT"),
		Entry("4 → PRODUCER", "4", "SPAN_KIND_PRODUCER"),
		Entry("5 → CONSUMER", "5", "SPAN_KIND_CONSUMER"),
		Entry("0 → UNSPECIFIED", "0", "SPAN_KIND_UNSPECIFIED"),
		Entry("empty → UNSPECIFIED", "", "SPAN_KIND_UNSPECIFIED"),
	)
})

var _ = Describe("statusCodeName", func() {
	DescribeTable("maps integer strings to OTel status code names",
		func(input, expected string) {
			Expect(statusCodeName(input)).To(Equal(expected))
		},
		Entry("1 → OK", "1", "STATUS_CODE_OK"),
		Entry("2 → ERROR", "2", "STATUS_CODE_ERROR"),
		Entry("0 → UNSET", "0", "STATUS_CODE_UNSET"),
		Entry("empty → UNSET", "", "STATUS_CODE_UNSET"),
	)
})
