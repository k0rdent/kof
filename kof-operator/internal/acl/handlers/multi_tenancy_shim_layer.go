package handlers

import (
	"fmt"
	"io"
	"net/http"
	"net/url"

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

	// Query parameter names used by Grafana for Prometheus queries.
	GrafanaQueryParamName = "query"
	// Grafana uses match[] parameter for label/series queries.
	GrafanaMatchParamName = "match[]"

	TenantLabelName = "tenantId"

	// DummyMatchSelector is used when match[] parameter is missing.
	// The prom-label-proxy requires a valid PromQL selector to inject tenant labels.
	// This placeholder will be replaced with the actual tenant matcher during injection.
	DummyMatchSelector = "{__name__=~\".+\"}"
)

var (
	PromxyHost string
	DevMode    bool
)

func HandleNotFound(res *server.Response, req *http.Request) {
	res.Writer.Header().Set("Content-Type", "text/plain")
	res.SetStatus(http.StatusNotFound)
	_, err := fmt.Fprintln(res.Writer, "404 - Page not found")
	if err != nil {
		res.Logger.Error(err, "Cannot write response")
	}
}

// HandleQueryWithTenant intercepts metric queries and injects tenant labels based on user identity.
// In DevMode, it bypasses tenant injection for admin access.
func HandleQueryWithTenant(res *server.Response, req *http.Request) {
	ctx := req.Context()
	defer func() {
		if err := req.Body.Close(); err != nil {
			res.Logger.Error(err, "failed to close request body")
		}
	}()

	// Check for authenticated user with ID token
	if idToken, ok := helper.GetIDToken(ctx); ok {
		handleTenantInjection(res, req, idToken, GrafanaQueryParamName)
		return
	}

	// Allow unrestricted access in development mode
	if DevMode {
		HandleProxyBypass(res, req)
		return
	}

	res.Fail("Unauthorized: authentication required", http.StatusUnauthorized)
}

// HandleProxyBypass forwards requests directly to Promxy without tenant filtering.
// This should only be used in development environments.
func HandleProxyBypass(res *server.Response, req *http.Request) {
	query := req.URL.Query()
	promxyURL := buildPromxyURL(PromxyHost, req.URL.Path, query.Encode())

	respBody, statusCode, err := forwardProxyResponse(res, promxyURL)
	if err != nil {
		res.Fail(fmt.Sprintf("failed to proxy request to promxy: %v", err), http.StatusInternalServerError)
		return
	}

	res.SendJson(string(respBody), statusCode)
}

// handleTenantInjection extracts tenant ID from the ID token and injects it into the query.
func handleTenantInjection(res *server.Response, req *http.Request, idToken *oidc.IDToken, paramName string) {
	query := req.URL.Query()

	// Extract tenant ID from authenticated user's token
	tenantID, err := extractTenantIDFromToken(idToken)
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
	promxyURL := buildPromxyURL(PromxyHost, req.URL.Path, query.Encode())

	respBody, statusCode, err := forwardProxyResponse(res, promxyURL)
	if err != nil {
		res.Fail(fmt.Sprintf("failed to proxy request to promxy: %v", err), http.StatusInternalServerError)
		return
	}

	res.SendJson(string(respBody), statusCode)
}

// extractTenantIDFromToken parses the ID token claims and extracts the tenant identifier.
func extractTenantIDFromToken(idToken *oidc.IDToken) (string, error) {
	claims := new(Claims)
	if err := idToken.Claims(claims); err != nil {
		return "", fmt.Errorf("failed to parse claims: %w", err)
	}

	tenantID := getTenantIDFromGroups(claims.Groups)
	if tenantID == "" {
		return "", fmt.Errorf("unauthorized: user has no tenant group (expected %s prefix)", TenantGroupPrefix)
	}

	return tenantID, nil
}

// getTenantIDFromGroups scans user groups for tenant membership and returns the tenant ID.
// Returns empty string if no tenant group is found.
func getTenantIDFromGroups(groups []string) string {
	prefixLen := len(TenantGroupPrefix)
	for _, group := range groups {
		if len(group) > prefixLen && group[:prefixLen] == TenantGroupPrefix {
			return group[prefixLen:]
		}
	}
	return ""
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

// buildPromxyURL constructs a complete URL for proxying requests to Promxy.
func buildPromxyURL(host, path, query string) string {
	return (&url.URL{
		Scheme:   "http",
		Host:     host,
		Path:     path,
		RawQuery: query,
	}).String()
}

// proxyRequestToPromxy creates and executes an HTTP request to Promxy.
func proxyRequestToPromxy(promxyURL string) (*http.Response, error) {
	proxyReq, err := http.NewRequest(http.MethodGet, promxyURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create proxy request: %w", err)
	}

	proxyReq.Header.Set("Content-Type", "application/json")

	proxyResp, err := http.DefaultClient.Do(proxyReq)
	if err != nil {
		return nil, fmt.Errorf("failed to forward request to promxy: %w", err)
	}

	return proxyResp, nil
}

// forwardProxyResponse sends a request to Promxy and returns the response body and status code.
func forwardProxyResponse(res *server.Response, promxyURL string) ([]byte, int, error) {
	promxyResp, err := proxyRequestToPromxy(promxyURL)
	if err != nil {
		return nil, 0, err
	}
	defer func() {
		if err := promxyResp.Body.Close(); err != nil {
			res.Logger.Error(err, "failed to close promxy response body")
		}
	}()

	respBody, err := io.ReadAll(promxyResp.Body)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read promxy response: %w", err)
	}

	return respBody, promxyResp.StatusCode, nil
}
