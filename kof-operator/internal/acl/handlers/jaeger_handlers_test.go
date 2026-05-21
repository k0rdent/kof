package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"

	"github.com/k0rdent/kof/kof-operator/internal/server"
	"github.com/k0rdent/kof/kof-operator/internal/server/helper"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	ctrl "sigs.k8s.io/controller-runtime"
)

var _ = Describe("JaegerServicesHandler", func() {
	var (
		req         *http.Request
		res         *server.Response
		mockBackend *httptest.Server
		handler     *JaegerServicesHandler
		logger      = ctrl.Log.WithName("test")
	)

	makeIDToken := func(tenant string) interface{} {
		return MockIDToken(map[string]any{
			"email":          "user@example.com",
			"groups":         []any{"tenant:" + tenant},
			"email_verified": true,
		})
	}

	buildServiceResponse := func(services []string) []byte {
		lines := make([]string, 0, len(services))
		for _, s := range services {
			entry := map[string]string{"resource_attr:service.name": s}
			b, _ := json.Marshal(entry)
			lines = append(lines, string(b))
		}
		return []byte(strings.Join(lines, "\n"))
	}

	BeforeEach(func() {
		req = httptest.NewRequest(http.MethodGet, "/api/services", nil)
		res = &server.Response{
			Writer: httptest.NewRecorder(),
			Logger: &logger,
		}
	})

	AfterEach(func() {
		if mockBackend != nil {
			mockBackend.Close()
		}
	})

	Context("with a valid tenant token", func() {
		It("returns services filtered for the tenant", func() {
			mockBackend = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				Expect(r.URL.Path).To(Equal("/select/logsql/query"))
				Expect(r.URL.Query().Get("extra_filters")).To(ContainSubstring("test-tenant"))

				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(buildServiceResponse([]string{"svc-a", "svc-b"}))
			}))

			parsedURL, err := url.Parse(mockBackend.URL)
			Expect(err).NotTo(HaveOccurred())
			handler = &JaegerServicesHandler{config: Config{Host: parsedURL.Host, Scheme: "http"}}

			idToken := makeIDToken("test-tenant")
			ctx := context.WithValue(req.Context(), helper.IdTokenContextKey, idToken)
			req = req.WithContext(ctx)

			ACLProxy(res, req, handler)

			recorder := res.Writer.(*httptest.ResponseRecorder)
			Expect(recorder.Code).To(Equal(http.StatusOK))

			var resp JaegerServiceResponse
			Expect(json.Unmarshal(recorder.Body.Bytes(), &resp)).To(Succeed())
			Expect(resp.Data).To(ConsistOf("svc-a", "svc-b"))
			Expect(resp.Total).To(Equal(2))
		})

		It("returns empty service list when backend returns no entries", func() {
			mockBackend = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				// empty body — EOF on first Decode
			}))

			parsedURL, _ := url.Parse(mockBackend.URL)
			handler = &JaegerServicesHandler{config: Config{Host: parsedURL.Host, Scheme: "http"}}

			idToken := makeIDToken("test-tenant")
			ctx := context.WithValue(req.Context(), helper.IdTokenContextKey, idToken)
			req = req.WithContext(ctx)

			ACLProxy(res, req, handler)

			recorder := res.Writer.(*httptest.ResponseRecorder)
			Expect(recorder.Code).To(Equal(http.StatusOK))

			var resp JaegerServiceResponse
			Expect(json.Unmarshal(recorder.Body.Bytes(), &resp)).To(Succeed())
			Expect(resp.Data).To(BeEmpty())
			Expect(resp.Total).To(Equal(0))
		})

		It("returns an error when the backend responds with non-OK status", func() {
			mockBackend = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			}))

			parsedURL, _ := url.Parse(mockBackend.URL)
			handler = &JaegerServicesHandler{config: Config{Host: parsedURL.Host, Scheme: "http"}}

			idToken := makeIDToken("test-tenant")
			ctx := context.WithValue(req.Context(), helper.IdTokenContextKey, idToken)
			req = req.WithContext(ctx)

			ACLProxy(res, req, handler)

			recorder := res.Writer.(*httptest.ResponseRecorder)
			Expect(recorder.Code).To(Equal(http.StatusInternalServerError))
		})
	})

	Context("without a tenant group", func() {
		It("returns 401 when token has no tenant group", func() {
			mockBackend = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			parsedURL, _ := url.Parse(mockBackend.URL)
			handler = &JaegerServicesHandler{config: Config{Host: parsedURL.Host, Scheme: "http"}}

			idToken := MockIDToken(map[string]any{
				"email":  "user@example.com",
				"groups": []any{"other-group"},
			})
			ctx := context.WithValue(req.Context(), helper.IdTokenContextKey, idToken)
			req = req.WithContext(ctx)

			ACLProxy(res, req, handler)

			recorder := res.Writer.(*httptest.ResponseRecorder)
			Expect(recorder.Code).To(Equal(http.StatusUnauthorized))
		})
	})
})

var _ = Describe("JaegerTracesHandler", func() {
	var (
		req         *http.Request
		res         *server.Response
		mockBackend *httptest.Server
		handler     *JaegerTracesHandler
		logger      = ctrl.Log.WithName("test")
	)

	AfterEach(func() {
		if mockBackend != nil {
			mockBackend.Close()
		}
	})

	Context("with a valid tenant token", func() {
		It("injects tenant tag into the query and proxies the response", func() {
			mockBackend = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				rawTags := r.URL.Query().Get("tags")
				Expect(rawTags).NotTo(BeEmpty())

				var tags map[string]string
				Expect(json.Unmarshal([]byte(rawTags), &tags)).To(Succeed())
				Expect(tags["resource_attr:"+TenantLabelName]).To(Equal("test-tenant"))

				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"data":[],"total":0,"limit":0,"offset":0,"errors":null}`))
			}))

			parsedURL, err := url.Parse(mockBackend.URL)
			Expect(err).NotTo(HaveOccurred())
			handler = &JaegerTracesHandler{config: Config{Host: parsedURL.Host, Scheme: "http"}}

			req = httptest.NewRequest(http.MethodGet, "/traces/api/traces?service=my-svc", nil)
			res = &server.Response{
				Writer: httptest.NewRecorder(),
				Logger: &logger,
			}

			idToken := MockIDToken(map[string]any{
				"email":  "user@example.com",
				"groups": []any{"tenant:test-tenant"},
			})
			ctx := context.WithValue(req.Context(), helper.IdTokenContextKey, idToken)
			req = req.WithContext(ctx)

			ACLProxy(res, req, handler)

			recorder := res.Writer.(*httptest.ResponseRecorder)
			Expect(recorder.Code).To(Equal(http.StatusOK))
		})

		It("returns 401 when token has no tenant group", func() {
			mockBackend = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			parsedURL, _ := url.Parse(mockBackend.URL)
			handler = &JaegerTracesHandler{config: Config{Host: parsedURL.Host, Scheme: "http"}}

			req = httptest.NewRequest(http.MethodGet, "/traces/api/traces", nil)
			res = &server.Response{
				Writer: httptest.NewRecorder(),
				Logger: &logger,
			}

			idToken := MockIDToken(map[string]any{
				"email":  "user@example.com",
				"groups": []any{"other-group"},
			})
			ctx := context.WithValue(req.Context(), helper.IdTokenContextKey, idToken)
			req = req.WithContext(ctx)

			ACLProxy(res, req, handler)

			recorder := res.Writer.(*httptest.ResponseRecorder)
			Expect(recorder.Code).To(Equal(http.StatusUnauthorized))
		})
	})
})

var _ = Describe("JaegerTraceHandler", func() {
	var (
		req         *http.Request
		res         *server.Response
		mockBackend *httptest.Server
		handler     *JaegerTraceHandler
		logger      = ctrl.Log.WithName("test")
	)

	buildTrace := func(traceID, tenantValue string) *JaegerTrace {
		return &JaegerTrace{
			TraceID: traceID,
			Spans:   json.RawMessage(`[]`),
			Processes: map[string]Process{
				"p1": {
					ServiceName: "svc",
					Tags: []KeyValue{
						{Key: TenantLabelName, Type: "string", Value: tenantValue},
					},
				},
			},
		}
	}

	AfterEach(func() {
		if mockBackend != nil {
			mockBackend.Close()
		}
	})

	Context("with a valid tenant token", func() {
		It("returns only traces that belong to the tenant", func() {
			matchingTrace := buildTrace("trace-1", "test-tenant")
			otherTrace := buildTrace("trace-2", "other-tenant")

			backendResp := JaegerTraceResponse{
				Data:  []*JaegerTrace{matchingTrace, otherTrace},
				Total: 2,
			}

			mockBackend = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				Expect(json.NewEncoder(w).Encode(backendResp)).To(Succeed())
			}))

			parsedURL, err := url.Parse(mockBackend.URL)
			Expect(err).NotTo(HaveOccurred())
			handler = &JaegerTraceHandler{config: Config{Host: parsedURL.Host, Scheme: "http"}}

			req = httptest.NewRequest(http.MethodGet, "/traces/api/traces/trace-1", nil)
			res = &server.Response{
				Writer: httptest.NewRecorder(),
				Logger: &logger,
			}

			idToken := MockIDToken(map[string]any{
				"email":  "user@example.com",
				"groups": []any{"tenant:test-tenant"},
			})
			ctx := context.WithValue(req.Context(), helper.IdTokenContextKey, idToken)
			req = req.WithContext(ctx)

			ACLProxy(res, req, handler)

			recorder := res.Writer.(*httptest.ResponseRecorder)
			Expect(recorder.Code).To(Equal(http.StatusOK))

			var resp JaegerTraceResponse
			Expect(json.Unmarshal(recorder.Body.Bytes(), &resp)).To(Succeed())
			Expect(resp.Data).To(HaveLen(1))
			Expect(resp.Data[0].TraceID).To(Equal("trace-1"))
			Expect(resp.Total).To(Equal(1))
		})

		It("returns an empty data list when no traces match the tenant", func() {
			backendResp := JaegerTraceResponse{
				Data:  []*JaegerTrace{buildTrace("trace-99", "other-tenant")},
				Total: 1,
			}

			mockBackend = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
				Expect(json.NewEncoder(w).Encode(backendResp)).To(Succeed())
			}))

			parsedURL, _ := url.Parse(mockBackend.URL)
			handler = &JaegerTraceHandler{config: Config{Host: parsedURL.Host, Scheme: "http"}}

			req = httptest.NewRequest(http.MethodGet, "/traces/api/traces/trace-99", nil)
			res = &server.Response{
				Writer: httptest.NewRecorder(),
				Logger: &logger,
			}

			idToken := MockIDToken(map[string]any{
				"email":  "user@example.com",
				"groups": []any{"tenant:test-tenant"},
			})
			ctx := context.WithValue(req.Context(), helper.IdTokenContextKey, idToken)
			req = req.WithContext(ctx)

			ACLProxy(res, req, handler)

			recorder := res.Writer.(*httptest.ResponseRecorder)
			Expect(recorder.Code).To(Equal(http.StatusOK))

			var resp JaegerTraceResponse
			Expect(json.Unmarshal(recorder.Body.Bytes(), &resp)).To(Succeed())
			Expect(resp.Data).To(BeEmpty())
			Expect(resp.Total).To(Equal(0))
		})

		It("returns an error when the backend responds with non-OK status", func() {
			mockBackend = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusServiceUnavailable)
			}))

			parsedURL, _ := url.Parse(mockBackend.URL)
			handler = &JaegerTraceHandler{config: Config{Host: parsedURL.Host, Scheme: "http"}}

			req = httptest.NewRequest(http.MethodGet, "/traces/api/traces/trace-1", nil)
			res = &server.Response{
				Writer: httptest.NewRecorder(),
				Logger: &logger,
			}

			idToken := MockIDToken(map[string]any{
				"email":  "user@example.com",
				"groups": []any{"tenant:test-tenant"},
			})
			ctx := context.WithValue(req.Context(), helper.IdTokenContextKey, idToken)
			req = req.WithContext(ctx)

			ACLProxy(res, req, handler)

			recorder := res.Writer.(*httptest.ResponseRecorder)
			Expect(recorder.Code).To(Equal(http.StatusServiceUnavailable))
		})

		It("returns 401 when token has no tenant group", func() {
			mockBackend = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			}))

			parsedURL, _ := url.Parse(mockBackend.URL)
			handler = &JaegerTraceHandler{config: Config{Host: parsedURL.Host, Scheme: "http"}}

			req = httptest.NewRequest(http.MethodGet, "/traces/api/traces/trace-1", nil)
			res = &server.Response{
				Writer: httptest.NewRecorder(),
				Logger: &logger,
			}

			idToken := MockIDToken(map[string]any{
				"email":  "user@example.com",
				"groups": []any{"other-group"},
			})
			ctx := context.WithValue(req.Context(), helper.IdTokenContextKey, idToken)
			req = req.WithContext(ctx)

			ACLProxy(res, req, handler)

			recorder := res.Writer.(*httptest.ResponseRecorder)
			Expect(recorder.Code).To(Equal(http.StatusUnauthorized))
		})
	})
})
