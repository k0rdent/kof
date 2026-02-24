package handlers

import (
	"fmt"
	"net/http"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/k0rdent/kof/kof-operator/internal/server"
	"github.com/k0rdent/kof/kof-operator/internal/server/helper"
	"github.com/prometheus-community/prom-label-proxy/injectproxy"
	"github.com/prometheus/prometheus/model/labels"
)

type Claims struct {
	Email    string   `json:"email"`
	Name     string   `json:"name"`
	Groups   []string `json:"groups"`
	Verified bool     `json:"email_verified"`
}

const (
	TenantGroupPrefix = "tenant:"

	// Query parameter name.
	PrometheusQueryParamName = "query"
	// Match parameter name used in series and labels endpoints.
	PrometheusMatchParamName = "match[]"

	TenantLabelName = "tenantId"

	// DummyMatchSelector is used when match[] parameter is missing.
	// The prom-label-proxy requires a valid PromQL selector to inject tenant labels.
	// This placeholder will be replaced with the actual tenant matcher during injection.
	DummyMatchSelector = "{__name__=~\".+\"}"
)

type Config struct {
	Host       string
	Scheme     string
	DevMode    bool
	AdminEmail string
}

// PromxyHandler handles Prometheus API requests with tenant isolation.
type PromxyHandler struct {
	config *Config
}

// NewHandler creates a new handler with the provided configuration.
func NewHandler(cfg Config) *PromxyHandler {
	return &PromxyHandler{config: &cfg}
}

// ProxyQueryWithTenantInjection intercepts metric queries and injects tenant labels based on user identity.
// In DevMode, it bypasses tenant injection for admin access.
func (h *PromxyHandler) ProxyQueryWithTenantInjection(res *server.Response, req *http.Request) {
	ctx := req.Context()
	defer func() {
		if err := req.Body.Close(); err != nil {
			res.Logger.Error(err, "failed to close request body")
		}
	}()

	// Check for authenticated user with ID token
	if idToken, ok := helper.GetIDToken(ctx); ok {
		if isAdminUser(idToken, h.config.AdminEmail) {
			h.HandleProxyBypass(res, req)
			return
		}

		h.handleTenantInjection(res, req, idToken, PrometheusQueryParamName)
		return
	}

	// Allow unrestricted access in development mode
	if h.config.DevMode {
		h.HandleProxyBypass(res, req)
		return
	}

	res.Fail("Unauthorized: authentication required", http.StatusUnauthorized)
}

// HandleProxyBypass forwards requests directly to Promxy without tenant filtering.
// This should only be used in development environments.
func (h *PromxyHandler) HandleProxyBypass(res *server.Response, req *http.Request) {
	query := req.URL.Query()
	promxyURL := BuildURL(h.config.Scheme, h.config.Host, req.URL.Path, query.Encode())

	statusCode, err := StreamProxyRequest(req.Context(), promxyURL, req.Method, res.Writer)
	if err != nil {
		res.Logger.Error(err, "failed to proxy request to promxy")
		http.Error(res.Writer, "unable to make request", statusCode)
		return
	}

	res.SetStatus(statusCode)
}

// handleTenantInjection extracts tenant ID from the ID token and injects it into the query.
func (h *PromxyHandler) handleTenantInjection(res *server.Response, req *http.Request, idToken *oidc.IDToken, paramName string) {
	query := req.URL.Query()

	// Extract tenant ID from authenticated user's token
	tenantID, err := ExtractTenantIDFromToken(idToken)
	if err != nil {
		res.Fail(fmt.Sprintf("failed to extract tenant ID: %v", err), http.StatusUnauthorized)
		return
	}

	if !query.Has(paramName) {
		res.Fail(fmt.Sprintf("missing required query parameter: %s", paramName), http.StatusBadRequest)
		return
	}

	// Inject tenant label into the query
	modifiedQuery, err := injectTenantIDLabel(tenantID, query.Get(paramName))
	if err != nil {
		res.Fail(fmt.Sprintf("failed to inject tenant ID label into query: %v", err), http.StatusBadRequest)
		return
	}

	// Forward modified query to Promxy
	query.Set(paramName, modifiedQuery)
	promxyURL := BuildURL(h.config.Scheme, h.config.Host, req.URL.Path, query.Encode())

	statusCode, err := StreamProxyRequest(req.Context(), promxyURL, req.Method, res.Writer)
	if err != nil {
		res.Logger.Error(err, "failed to proxy request to promxy")
		http.Error(res.Writer, "unable to make request", statusCode)
		return
	}

	res.SetStatus(statusCode)
}

// injectTenantIDLabel adds a tenant label matcher to a PromQL query using prom-label-proxy.
// This ensures queries only access metrics belonging to the specified tenant.
func injectTenantIDLabel(tenantID, originalQuery string) (string, error) {
	enforcer := injectproxy.NewPromQLEnforcer(
		false,
		&labels.Matcher{
			Name:  TenantLabelName,
			Value: tenantID,
			Type:  labels.MatchEqual,
		},
	)
	return enforcer.Enforce(originalQuery)
}
