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
	"net/http"
	"net/http/httptest"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("VLogsClient.DiscoverTenants", func() {
	var (
		server *httptest.Server
		client *VLogsClient
	)

	// responseBody is set per-entry before the server handler is called.
	var responseBody string
	var responseStatus int

	BeforeEach(func() {
		responseStatus = http.StatusOK
		server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			Expect(r.URL.Path).To(Equal("/select/logsql/field_values"))
			Expect(r.URL.Query().Get("field")).To(Equal("tenant"))
			Expect(r.URL.Query().Get("query")).To(Equal("*"))
			w.WriteHeader(responseStatus)
			_, _ = w.Write([]byte(responseBody))
		}))
		client = NewVLogsClient(server.URL)
	})

	AfterEach(func() { server.Close() })

	window := func() (time.Time, time.Time) {
		start := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
		return start, start.Add(time.Hour)
	}

	DescribeTable("tenant filtering",
		func(body string, expected []string) {
			responseBody = body
			start, end := window()
			tenants, err := client.DiscoverTenants(context.Background(), start, end)
			Expect(err).NotTo(HaveOccurred())
			Expect(tenants).To(ConsistOf(expected))
		},
		Entry("regular tenants are returned",
			`{"value":"acme","hits":3}`+"\n"+`{"value":"globex","hits":1}`,
			[]string{"acme", "globex"},
		),
		Entry("PLATFORM tenant is excluded",
			`{"value":"PLATFORM","hits":10}`+"\n"+`{"value":"acme","hits":2}`,
			[]string{"acme"},
		),
		Entry("empty tenant is treated as PLATFORM — excluded from tenant stream",
			`{"value":"","hits":5}`+"\n"+`{"value":"acme","hits":2}`,
			[]string{"acme"},
		),
		Entry("all platform records — returns nil",
			`{"value":"PLATFORM","hits":10}`+"\n"+`{"value":"","hits":3}`,
			nil,
		),
		Entry("empty response body — returns nil",
			``,
			nil,
		),
	)

	It("returns SourceUnavailableError on non-200 response", func() {
		responseStatus = http.StatusServiceUnavailable
		responseBody = "service unavailable"
		start, end := window()
		_, err := client.DiscoverTenants(context.Background(), start, end)
		Expect(err).To(HaveOccurred())
		Expect(err).To(BeAssignableToTypeOf(&SourceUnavailableError{}))
	})

	It("returns SourceUnavailableError when the server is unreachable", func() {
		server.Close() // close before the call
		start, end := window()
		_, err := client.DiscoverTenants(context.Background(), start, end)
		Expect(err).To(HaveOccurred())
		Expect(err).To(BeAssignableToTypeOf(&SourceUnavailableError{}))
	})
})

var _ = Describe("flatToAuditEvent extra fields", func() {
	It("preserves unknown top-level keys in Extra", func() {
		ev, err := flatToAuditEvent(map[string]string{
			"tenant":     "acme",
			"event_id":   "abc-123",
			"custom_key": "custom_value",
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(ev.EventID).To(Equal("abc-123"))
		Expect(ev.Extra).To(HaveKey("custom_key"))
		Expect(string(ev.Extra["custom_key"])).To(Equal(`"custom_value"`))
	})

	It("preserves dot-notation extra keys as nested objects in Extra", func() {
		ev, err := flatToAuditEvent(map[string]string{
			"tenant":        "acme",
			"custom.nested": "nested_value",
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(ev.Extra).To(HaveKey("custom"))
		// The nested object should marshal correctly.
		var nested map[string]string
		Expect(json.Unmarshal(ev.Extra["custom"], &nested)).To(Succeed())
		Expect(nested["nested"]).To(Equal("nested_value"))
	})

	It("extra fields appear in MarshalJSON output", func() {
		ev, err := flatToAuditEvent(map[string]string{
			"tenant":     "acme",
			"event_id":   "abc-123",
			"custom_key": "custom_value",
		})
		Expect(err).NotTo(HaveOccurred())
		b, err := json.Marshal(ev)
		Expect(err).NotTo(HaveOccurred())
		var m map[string]interface{}
		Expect(json.Unmarshal(b, &m)).To(Succeed())
		Expect(m["custom_key"]).To(Equal("custom_value"))
		// Known field must not be overwritten by a clashing Extra key.
		Expect(m["event_id"]).To(Equal("abc-123"))
	})

	It("does not include VLogs internal fields in Extra", func() {
		ev, err := flatToAuditEvent(map[string]string{
			"_time":      "2025-01-01T10:00:00Z",
			"_stream_id": "0000000000000001",
			"_msg":       "log line",
			"event_id":   "abc-123",
		})
		Expect(err).NotTo(HaveOccurred())
		Expect(ev.Extra).To(BeNil())
	})
})

var _ = Describe("flatToAuditEvent tenant normalisation", func() {
	DescribeTable("empty tenant is normalised to PLATFORM",
		func(tenant string, expected string) {
			ev, err := flatToAuditEvent(map[string]string{"tenant": tenant})
			Expect(err).NotTo(HaveOccurred())
			Expect(ev.Tenant).To(Equal(expected))
		},
		Entry("absent tenant field", "", TenantPlatform),
		Entry("explicit PLATFORM value", TenantPlatform, TenantPlatform),
		Entry("regular tenant is preserved", "acme", "acme"),
	)
})
