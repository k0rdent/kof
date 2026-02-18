package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"

	"github.com/k0rdent/kof/kof-operator/internal/server"
	"github.com/k0rdent/kof/kof-operator/internal/server/helper"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/prometheus/prometheus/promql/parser"
	ctrl "sigs.k8s.io/controller-runtime"
)

var _ = Describe("HandleMatchWithTenant", func() {
	var (
		req        *http.Request
		res        *server.Response
		mockPromxy *httptest.Server
		handler    *Handler
		logger     = ctrl.Log.WithName("test")
	)

	BeforeEach(func() {
		mockPromxy = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)

			matches := r.URL.Query()["match[]"]

			for _, match := range matches {
				if _, err := parser.ParseMetricSelector(match); err != nil {
					response := map[string]any{
						"status": "error",
						"error":  "invalid match[] selector",
					}
					w.WriteHeader(http.StatusBadRequest)
					Expect(json.NewEncoder(w).Encode(response)).NotTo(HaveOccurred())
					return
				}
			}

			response := map[string]any{
				"status": "success",
				"data":   []any{},
			}
			Expect(json.NewEncoder(w).Encode(response)).NotTo(HaveOccurred())
		}))

		parsedURL, err := url.Parse(mockPromxy.URL)
		Expect(err).NotTo(HaveOccurred())

		handler = NewHandler(Config{
			PromxyHost: parsedURL.Host,
			DevMode:    false,
			AdminEmail: "",
		})

		req = httptest.NewRequest(http.MethodGet, "/api/v1/series", nil)
		res = &server.Response{
			Writer: httptest.NewRecorder(),
			Logger: &logger,
		}
	})

	AfterEach(func() {
		mockPromxy.Close()
	})

	Context("when user is authenticated with valid tenant", func() {
		It("should inject tenant label into match[] parameter and forward to Promxy", func() {
			req = httptest.NewRequest(http.MethodGet, `/api/v1/series?match[]={job="prometheus"}`, nil)
			res = &server.Response{
				Writer: httptest.NewRecorder(),
				Logger: &logger,
			}

			idToken := MockIDToken(map[string]any{
				"email":          "user@example.com",
				"name":           "Test User",
				"groups":         []any{"tenant:test-tenant", "developers"},
				"email_verified": true,
			})

			ctx := context.WithValue(req.Context(), helper.IdTokenContextKey, idToken)
			req = req.WithContext(ctx)

			handler.HandleMatchWithTenant(res, req)

			recorder := res.Writer.(*httptest.ResponseRecorder)
			Expect(recorder.Code).To(Equal(http.StatusOK))

			var response map[string]any
			err := json.Unmarshal(recorder.Body.Bytes(), &response)
			Expect(err).NotTo(HaveOccurred())
			Expect(response["status"]).To(Equal("success"))
		})

		It("should add dummy match[] selector when parameter is missing", func() {
			req = httptest.NewRequest(http.MethodGet, "/api/v1/series", nil)
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

			handler.HandleMatchWithTenant(res, req)

			recorder := res.Writer.(*httptest.ResponseRecorder)
			Expect(recorder.Code).To(Equal(http.StatusOK))
		})

		It("should reject request when tenant group is missing", func() {
			idToken := MockIDToken(map[string]any{
				"email":  "user@example.com",
				"groups": []any{"developers", "admin"},
			})

			ctx := context.WithValue(req.Context(), helper.IdTokenContextKey, idToken)
			req = req.WithContext(ctx)

			handler.HandleMatchWithTenant(res, req)

			recorder := res.Writer.(*httptest.ResponseRecorder)
			Expect(recorder.Code).To(Equal(http.StatusUnauthorized))
		})
	})
})
