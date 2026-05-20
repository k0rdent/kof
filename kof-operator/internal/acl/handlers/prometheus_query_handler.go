package handlers

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/k0rdent/kof/kof-operator/internal/server"
	"github.com/prometheus-community/prom-label-proxy/injectproxy"
	"github.com/prometheus/prometheus/model/labels"
)

type Claims struct {
	Email    string   `json:"email"`
	Name     string   `json:"name"`
	Groups   []string `json:"groups"`
	Verified bool     `json:"email_verified"`
	TenantID string   `json:"tenant,omitempty"`
}

const (
	TenantGroupPrefix = "tenant:"

	// Query parameter name.
	PrometheusQueryParamName = "query"
	// Match parameter name used in series and labels endpoints.
	PrometheusMatchParamName = "match[]"

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

// PromxyQueryHandler handles Prometheus query API requests with tenant isolation.
type PromxyQueryHandler struct {
	config Config
}

// NewPromxyQueryHandler creates a new handler with the provided configuration.
func NewPromxyQueryHandler(cfg Config) Proxy {
	return &PromxyQueryHandler{config: cfg}
}

func (h *PromxyQueryHandler) AdminEmail() string { return h.config.AdminEmail }
func (h *PromxyQueryHandler) IsDevMode() bool    { return h.config.DevMode }
func (h *PromxyQueryHandler) Schema() string     { return h.config.Scheme }
func (h *PromxyQueryHandler) Host() string       { return h.config.Host }

func (h *PromxyQueryHandler) HandleTenantInjection(res *server.Response, req *http.Request, idToken *oidc.IDToken) {
	var paramName string
	var body io.Reader

	query, err := extractQuery(req, res.Writer)
	if err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			res.Fail(fmt.Sprintf("request body too large: %v", err), http.StatusRequestEntityTooLarge)
			return
		}
		res.Fail(fmt.Sprintf("failed to extract query: %v", err), http.StatusBadRequest)
		return
	}

	switch {
	case query.Has(PrometheusQueryParamName):
		paramName = PrometheusQueryParamName
	case query.Has(PrometheusMatchParamName):
		paramName = PrometheusMatchParamName
	case isQueryEndpoint(req.URL.Path):
		// query-based endpoints (/query, /query_range, etc.) require the 'query' parameter.
		res.Fail(fmt.Sprintf("missing required query parameter: %s", PrometheusQueryParamName), http.StatusBadRequest)
		return
	default:
		// match[]-based endpoints (/series, /labels, /label/*) without a match[] param:
		// inject a dummy selector so the tenant filter can still be applied.
		query.Set(PrometheusMatchParamName, DummyMatchSelector)
		paramName = PrometheusMatchParamName
	}

	tenantID, err := ExtractTenantIDFromToken(idToken)
	if err != nil {
		res.Fail(fmt.Sprintf("failed to extract tenant ID: %v", err), http.StatusUnauthorized)
		return
	}

	modifiedQuery, err := injectTenantIDLabel(tenantID, query.Get(paramName))
	if err != nil {
		res.Fail(fmt.Sprintf("failed to inject tenant ID label into query: %v", err), http.StatusBadRequest)
		return
	}

	query.Set(paramName, modifiedQuery)

	if req.Method == http.MethodPost {
		body = strings.NewReader(query.Encode())
		query = req.URL.Query() // Clear query parameters from URL for POST requests, as they are sent in the body
	}

	path := strings.TrimPrefix(req.URL.Path, "/metrics")
	promxyURL := BuildURL(h.config.Scheme, h.config.Host, path, query.Encode())

	if err := StreamProxyRequest(req.Context(), promxyURL, req.Method, body, res.Writer); err != nil {
		res.Logger.Error(err, "failed to proxy request to promxy")
		http.Error(res.Writer, "unable to make request", http.StatusInternalServerError)
		return
	}
}

// isQueryEndpoint reports whether the request path uses the 'query' PromQL parameter
// (e.g. /query, /query_range, /format_query) rather than 'match[]' (series, labels, label/*).
func isQueryEndpoint(path string) bool {
	return strings.Contains(path, "query")
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
