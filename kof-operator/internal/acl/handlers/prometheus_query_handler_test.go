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

var _ = Describe("HandleQueryWithTenant", func() {
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

			rawExpr := r.URL.Query().Get("query")
			if _, err := parser.ParseExpr(rawExpr); err != nil {
				response := map[string]any{
					"status": "error",
					"error":  "invalid query expression",
				}
				Expect(json.NewEncoder(w).Encode(response)).NotTo(HaveOccurred())
				return
			}

			response := map[string]any{
				"status": "success",
				"data": map[string]any{
					"resultType": "vector",
					"result":     []any{},
				},
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

		req = httptest.NewRequest(http.MethodGet, "/api/v1/query?query=up", nil)
		res = &server.Response{
			Writer: httptest.NewRecorder(),
			Logger: &logger,
		}
	})

	AfterEach(func() {
		mockPromxy.Close()
	})

	Context("when user is authenticated with valid tenant", func() {
		It("should inject tenant label into query and forward to Promxy", func() {
			idToken := MockIDToken(map[string]any{
				"email":          "user@example.com",
				"name":           "Test User",
				"groups":         []any{"tenant:test-tenant", "other-group"},
				"email_verified": true,
			})

			ctx := context.WithValue(req.Context(), helper.IdTokenContextKey, idToken)
			req = req.WithContext(ctx)

			handler.HandleQueryWithTenant(res, req)

			recorder := res.Writer.(*httptest.ResponseRecorder)
			Expect(recorder.Code).To(Equal(http.StatusOK))

			var response map[string]any
			err := json.Unmarshal(recorder.Body.Bytes(), &response)
			Expect(err).NotTo(HaveOccurred())
			Expect(response).To(HaveKey("status"))
			Expect(response["status"]).To(Equal("success"))
		})

		It("should reject request when tenant group is missing", func() {
			idToken := MockIDToken(map[string]any{
				"email":          "user@example.com",
				"name":           "Test User",
				"groups":         []any{"other-group", "admin-group"},
				"email_verified": true,
			})

			ctx := context.WithValue(req.Context(), helper.IdTokenContextKey, idToken)
			req = req.WithContext(ctx)

			handler.HandleQueryWithTenant(res, req)

			recorder := res.Writer.(*httptest.ResponseRecorder)
			Expect(recorder.Code).To(Equal(http.StatusUnauthorized))
			Expect(recorder.Body.String()).To(ContainSubstring("user has no tenant group"))
		})

		It("should reject request when query parameter is missing", func() {
			req = httptest.NewRequest(http.MethodGet, "/api/v1/query", nil)
			res = &server.Response{
				Writer: httptest.NewRecorder(),
				Logger: &logger,
			}

			idToken := MockIDToken(map[string]any{
				"email":          "user@example.com",
				"groups":         []any{"tenant:test-tenant"},
				"email_verified": true,
			})

			ctx := context.WithValue(req.Context(), helper.IdTokenContextKey, idToken)
			req = req.WithContext(ctx)

			handler.HandleQueryWithTenant(res, req)

			recorder := res.Writer.(*httptest.ResponseRecorder)
			Expect(recorder.Code).To(Equal(http.StatusBadRequest))
			Expect(recorder.Body.String()).To(ContainSubstring("missing required query parameter"))
		})
	})

	Context("when DevMode is enabled", func() {
		BeforeEach(func() {
			handler.config.DevMode = true
		})

		It("should bypass authentication and allow unrestricted access", func() {
			handler.HandleQueryWithTenant(res, req)

			recorder := res.Writer.(*httptest.ResponseRecorder)
			Expect(recorder.Code).To(Equal(http.StatusOK))

			var response map[string]any
			err := json.Unmarshal(recorder.Body.Bytes(), &response)
			Expect(err).NotTo(HaveOccurred())
			Expect(response["status"]).To(Equal("success"))
		})
	})

	Context("when user is not authenticated and DevMode is disabled", func() {
		It("should return unauthorized error", func() {
			handler.config.DevMode = false
			handler.HandleQueryWithTenant(res, req)

			recorder := res.Writer.(*httptest.ResponseRecorder)
			Expect(recorder.Code).To(Equal(http.StatusUnauthorized))
			Expect(recorder.Body.String()).To(ContainSubstring("Unauthorized"))
		})
	})

	Context("when user is authenticated as admin", func() {
		BeforeEach(func() {
			handler.config.AdminEmail = "admin@example.com"
		})

		It("should bypass tenant filtering for admin user", func() {
			idToken := MockIDToken(map[string]any{
				"email":          "admin@example.com",
				"name":           "Admin User",
				"groups":         []any{"admin-group"},
				"email_verified": true,
			})

			ctx := context.WithValue(req.Context(), helper.IdTokenContextKey, idToken)
			req = req.WithContext(ctx)

			handler.HandleQueryWithTenant(res, req)

			recorder := res.Writer.(*httptest.ResponseRecorder)
			Expect(recorder.Code).To(Equal(http.StatusOK))

			var response map[string]any
			err := json.Unmarshal(recorder.Body.Bytes(), &response)
			Expect(err).NotTo(HaveOccurred())
			Expect(response["status"]).To(Equal("success"))
		})

		It("should enforce tenant filtering for non-admin user", func() {
			idToken := MockIDToken(map[string]any{
				"email":          "user@example.com",
				"name":           "Regular User",
				"groups":         []any{"other-group"},
				"email_verified": true,
			})

			ctx := context.WithValue(req.Context(), helper.IdTokenContextKey, idToken)
			req = req.WithContext(ctx)

			handler.HandleQueryWithTenant(res, req)

			recorder := res.Writer.(*httptest.ResponseRecorder)
			Expect(recorder.Code).To(Equal(http.StatusUnauthorized))
			Expect(recorder.Body.String()).To(ContainSubstring("user has no tenant group"))
		})
	})
})

var _ = Describe("Tenant ID Extraction", func() {
	It("should extract tenant ID from groups with correct prefix", func() {
		groups := []string{"tenant:production", "admin", "developers"}
		tenantID := getTenantIDFromGroups(groups)
		Expect(tenantID).To(Equal("production"))
	})

	It("should return first matching tenant group when multiple exist", func() {
		groups := []string{"tenant:prod", "tenant:staging", "admin"}
		tenantID := getTenantIDFromGroups(groups)
		Expect(tenantID).To(Equal("prod"))
	})

	It("should return empty string when no tenant group exists", func() {
		groups := []string{"admin", "developers", "viewers"}
		tenantID := getTenantIDFromGroups(groups)
		Expect(tenantID).To(BeEmpty())
	})

	It("should return empty string for empty groups", func() {
		groups := []string{}
		tenantID := getTenantIDFromGroups(groups)
		Expect(tenantID).To(BeEmpty())
	})

	It("should handle tenant groups with special characters", func() {
		groups := []string{"tenant:my-tenant-123", "admin"}
		tenantID := getTenantIDFromGroups(groups)
		Expect(tenantID).To(Equal("my-tenant-123"))
	})

	It("should not match partial prefix", func() {
		groups := []string{"tenants:production", "mytenant:test"}
		tenantID := getTenantIDFromGroups(groups)
		Expect(tenantID).To(BeEmpty())
	})
})
