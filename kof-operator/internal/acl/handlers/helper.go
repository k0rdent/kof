package handlers

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"

	"github.com/coreos/go-oidc/v3/oidc"
)

func StreamProxyRequest(ctx context.Context, url, method string, writer io.Writer) (int, error) {
	resp, err := ProxyRequest(ctx, url, method)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to proxy request: %w", err)
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("failed to close response body: %v\n", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return resp.StatusCode, fmt.Errorf("received non-OK response: %s", resp.Status)
	}

	if _, err := io.Copy(writer, resp.Body); err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to proxy response body: %w", err)
	}

	return resp.StatusCode, nil
}

// ProxyRequest creates and executes an HTTP request to Promxy.
func ProxyRequest(ctx context.Context, promxyURL, method string) (*http.Response, error) {
	proxyReq, err := http.NewRequestWithContext(ctx, method, promxyURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create proxy request: %w", err)
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
