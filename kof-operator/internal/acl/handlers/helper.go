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

const TenantLabelName = "tenant"

const (
	KBytes = 1024
	MBytes = 1024 * KBytes
	// MaxBodySize defines the maximum size of the request body that will be read for tenant injection.
	// This is a safeguard to prevent excessive memory usage when parsing large queries.
	MaxBodySize = 10 * MBytes
)

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

// ProxyRequest creates and executes an HTTP request to Promxy.
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

func extractQuery(res *http.Request, writer http.ResponseWriter) (url.Values, error) {
	query := res.URL.Query()

	if res.Method == http.MethodPost {
		// Limit request body size to prevent memory exhaustion
		res.Body = http.MaxBytesReader(writer, res.Body, MaxBodySize)

		q, err := io.ReadAll(res.Body)
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
