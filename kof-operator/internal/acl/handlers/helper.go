package handlers

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/k0rdent/kof/kof-operator/internal/server"
	"github.com/k0rdent/kof/kof-operator/internal/server/helper"
)

const TenantLabelName = "tenant"

const (
	KBytes = 1024
	MBytes = 1024 * KBytes
	// MaxBodySize defines the maximum size of the request body that will be read for tenant injection.
	// This is a safeguard to prevent excessive memory usage when parsing large queries.
	MaxBodySize = 10 * MBytes
)

type Proxy interface {
	// HandleTenantInjection processes the incoming request, extracts tenant information from the authenticated user's token,
	// and modifies the request to include tenant-specific filters before proxying it to the backend service.
	HandleTenantInjection(res *server.Response, req *http.Request, idToken *oidc.IDToken)
	// IsDevMode reports whether development mode is enabled, allowing requests to bypass tenant and admin checks.
	IsDevMode() bool
	// AdminEmail returns the configured admin email for access control checks.
	AdminEmail() string
	// Schema returns the URL scheme (http or https) to be used when constructing the backend service URL.
	Schema() string
	// Host returns the backend service host (including port if necessary) to which the request should be proxied.
	Host() string
}

func ACLProxy(res *server.Response, req *http.Request, proxy Proxy) {
	ctx := req.Context()
	defer func() {
		if err := req.Body.Close(); err != nil {
			res.Logger.Error(err, "failed to close request body")
		}
	}()

	// Check for authenticated user with ID token
	if idToken, ok := helper.GetIDToken(ctx); ok {
		if isAdminUser(idToken, proxy.AdminEmail()) {
			ProxyBypass(res, req, proxy)
			return
		}

		proxy.HandleTenantInjection(res, req, idToken)
		return
	}

	// Allow unrestricted access in development mode
	if proxy.IsDevMode() {
		ProxyBypass(res, req, proxy)
		return
	}

	res.Fail("Unauthorized: authentication required", http.StatusUnauthorized)
}

func AdminProxy(res *server.Response, req *http.Request, proxy Proxy) {
	ctx := req.Context()
	defer func() {
		if err := req.Body.Close(); err != nil {
			res.Logger.Error(err, "failed to close request body")
		}
	}()

	if idToken, ok := helper.GetIDToken(ctx); ok {
		if isAdminUser(idToken, proxy.AdminEmail()) {
			ProxyBypass(res, req, proxy)
			return
		}
	}

	if proxy.IsDevMode() {
		ProxyBypass(res, req, proxy)
		return
	}

	res.Fail("Forbidden: admin access required", http.StatusForbidden)
}

func ProxyBypass(res *server.Response, req *http.Request, proxy Proxy) {
	var body io.Reader

	if req.Method == http.MethodPost {
		body = req.Body
	}

	// Remove the /metrics, /traces and /logs prefixes to construct the correct backend URL path
	pathParts := strings.Split(req.URL.Path, "/")
	path := strings.Join(pathParts[2:], "/")

	targetURL := BuildURL(proxy.Schema(), proxy.Host(), path, req.URL.Query().Encode())

	if err := StreamProxyRequest(req.Context(), targetURL, req.Method, body, res.Writer); err != nil {
		res.Logger.Error(err, "failed to proxy request: "+targetURL)
		http.Error(res.Writer, "unable to make request", http.StatusInternalServerError)
		return
	}
}

func StreamProxyRequest(ctx context.Context, url, method string, body io.Reader, writer http.ResponseWriter) error {
	resp, err := ProxyRequest(ctx, url, method, body)
	if err != nil {
		return fmt.Errorf("failed to proxy request: %w", err)
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("failed to close response body: %v\n", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("received non-OK response: %s", resp.Status)
	}

	writer.Header().Add("Content-Type", "application/json")

	if _, err := io.Copy(writer, resp.Body); err != nil {
		return fmt.Errorf("failed to proxy response body: %w", err)
	}

	return nil
}

// ProxyRequest creates and executes an HTTP request to a backend service.
func ProxyRequest(ctx context.Context, promxyURL, method string, body io.Reader) (*http.Response, error) {
	proxyReq, err := http.NewRequestWithContext(ctx, method, promxyURL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create proxy request: %w", err)
	}

	if method == http.MethodPost {
		proxyReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	proxyResp, err := http.DefaultClient.Do(proxyReq)
	if err != nil {
		return nil, fmt.Errorf("failed to proxy request: %w", err)
	}

	return proxyResp, nil
}

func ExtractTenantIDFromToken(idToken *oidc.IDToken) (string, error) {
	claims := new(Claims)
	if err := idToken.Claims(claims); err != nil {
		return "", fmt.Errorf("failed to parse claims: %w", err)
	}

	if claims.TenantID != "" {
		return claims.TenantID, nil
	}

	tenantID := getTenantIDFromGroups(claims.Groups)
	if tenantID != "" {
		return tenantID, nil
	}

	return "", fmt.Errorf("unauthorized: user has no tenant group (expected %s prefix)", TenantGroupPrefix)
}

func extractQuery(req *http.Request, writer http.ResponseWriter) (url.Values, error) {
	query := req.URL.Query()

	if req.Method == http.MethodPost {
		// Limit request body size to prevent memory exhaustion
		req.Body = http.MaxBytesReader(writer, req.Body, MaxBodySize)

		q, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}

		query, err = url.ParseQuery(string(q))
		if err != nil {
			return nil, err
		}
	}

	return query, nil
}

// getTenantIDFromGroups scans user groups for tenant membership and returns the tenant ID.
// Returns empty string if no tenant group is found.
func getTenantIDFromGroups(groups []string) string {
	for _, group := range groups {
		if id, ok := strings.CutPrefix(group, TenantGroupPrefix); ok && id != "" {
			return id
		}
	}
	return ""
}

// isAdminUser checks if the authenticated user has admin privileges based on email.
// Admins bypass tenant filtering and get unrestricted access to all metrics.
func isAdminUser(idToken *oidc.IDToken, adminEmail string) bool {
	if adminEmail == "" {
		return false
	}

	claims := new(Claims)
	if err := idToken.Claims(claims); err != nil {
		return false
	}

	return claims.Email == adminEmail
}

func BuildURL(scheme, host, path, query string) string {
	return (&url.URL{
		Scheme:   scheme,
		Host:     host,
		Path:     path,
		RawQuery: query,
	}).String()
}
