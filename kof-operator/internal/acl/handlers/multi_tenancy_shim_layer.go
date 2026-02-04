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
	TenantGroupPrefix     = "tenant:"
	GrafanaQueryParamName = "query"

	TenantLabelName = "tenantId"
)

var (
	PromxyHost string
	DevMode    bool
)

func NotFoundHandler(res *server.Response, req *http.Request) {
	res.Writer.Header().Set("Content-Type", "text/plain")
	res.SetStatus(http.StatusNotFound)
	_, err := fmt.Fprintln(res.Writer, "404 - Page not found")
	if err != nil {
		res.Logger.Error(err, "Cannot write response")
	}
}

func MetricsTenantInjectionHandler(res *server.Response, req *http.Request) {
	ctx := req.Context()
	defer func() {
		if err := req.Body.Close(); err != nil {
			res.Logger.Error(err, "failed to close request body")
		}
	}()

	if idToken, ok := helper.GetIDToken(ctx); ok {
		handleTenantInjection(idToken, res, req)
		return
	}

	if DevMode {
		handleAdminBypass(res, req)
		return
	}

	res.Fail("Unauthorized: authentication required", http.StatusUnauthorized)
}

func handleAdminBypass(res *server.Response, req *http.Request) {
	query := req.URL.Query()
	promxyURL := buildPromxyURL(PromxyHost, req.URL.Path, query.Encode())

	resp, statusCode, err := forwardProxyResponse(res, promxyURL)
	if err != nil {
		res.Fail(fmt.Sprintf("failed to proxy request to promxy: %v", err), http.StatusInternalServerError)
		return
	}

	res.SendJson(string(resp), statusCode)
}

func handleTenantInjection(idToken *oidc.IDToken, res *server.Response, req *http.Request) {
	query := req.URL.Query()

	tenantID, err := extractTenantIDFromToken(idToken)
	if err != nil {
		res.Fail(fmt.Sprintf("failed to extract tenant ID: %v", err), http.StatusUnauthorized)
		return
	}

	originalQuery := query.Get(GrafanaQueryParamName)
	modifiedQuery, err := injectTenantIDLabel(tenantID, originalQuery)
	if err != nil {
		res.Fail(fmt.Sprintf("failed to inject tenant ID label into query: %v", err), http.StatusBadRequest)
		return
	}

	query.Set(GrafanaQueryParamName, modifiedQuery)
	promxyURL := buildPromxyURL(PromxyHost, req.URL.Path, query.Encode())

	resp, statusCode, err := forwardProxyResponse(res, promxyURL)
	if err != nil {
		res.Fail(fmt.Sprintf("failed to proxy request to promxy: %v", err), http.StatusInternalServerError)
		return
	}

	res.SendJson(string(resp), statusCode)
}

func extractTenantIDFromToken(idToken *oidc.IDToken) (string, error) {
	claims := new(Claims)
	if err := idToken.Claims(claims); err != nil {
		return "", fmt.Errorf("failed to parse claims: %w", err)
	}

	tenantID := getTenantIDFromGroups(claims.Groups)
	if tenantID == "" {
		return "", fmt.Errorf("unauthorized: missing tenant group")
	}

	return tenantID, nil
}

func getTenantIDFromGroups(groups []string) string {
	prefixLen := len(TenantGroupPrefix)
	for _, group := range groups {
		if len(group) > prefixLen && group[:prefixLen] == TenantGroupPrefix {
			return group[prefixLen:]
		}
	}
	return ""
}

func injectTenantIDLabel(tenantID, originalQuery string) (string, error) {
	enforcer := injectproxy.NewPromQLEnforcer(false,
		&labels.Matcher{
			Name:  TenantLabelName,
			Value: tenantID,
			Type:  labels.MatchEqual,
		},
	)
	return enforcer.Enforce(originalQuery)
}

func buildPromxyURL(host, path, query string) string {
	proxyURL := &url.URL{
		Scheme:   "http",
		Host:     host,
		Path:     path,
		RawQuery: query,
	}
	return proxyURL.String()
}

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

func forwardProxyResponse(res *server.Response, promxyURL string) ([]byte, int, error) {
	promxyResp, err := proxyRequestToPromxy(promxyURL)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to proxy request to promxy: %w", err)
	}

	defer func() {
		if err := promxyResp.Body.Close(); err != nil {
			res.Logger.Error(err, "failed to close promxy response body")
		}
	}()

	promxyRespBody, err := io.ReadAll(promxyResp.Body)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read promxy response: %w", err)
	}

	return promxyRespBody, promxyResp.StatusCode, nil
}
