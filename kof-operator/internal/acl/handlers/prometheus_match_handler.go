package handlers

import (
	"net/http"

	"github.com/k0rdent/kof/kof-operator/internal/server"
	"github.com/k0rdent/kof/kof-operator/internal/server/helper"
)

// HandleMatchWithTenant handles Prometheus API endpoints that use match[] parameter.
// This includes /api/v1/series, /api/v1/labels, /api/v1/label/*, and /api/v1/rules.
// It ensures tenant isolation by injecting tenant labels into match[] selectors.
func (h *Handler) HandleMatchWithTenant(res *server.Response, req *http.Request) {
	ctx := req.Context()
	defer func() {
		if err := req.Body.Close(); err != nil {
			res.Logger.Error(err, "failed to close request body")
		}
	}()

	// Check for authenticated user with ID token
	if idToken, ok := helper.GetIDToken(ctx); ok {
		// Check if user is admin - admins get unrestricted access
		if h.isAdminUser(idToken) {
			h.HandleProxyBypass(res, req)
			return
		}

		query := req.URL.Query()

		// Ensure match[] parameter exists for prom-label-proxy to work.
		// If absent, set a generic selector that matches all metrics.
		if !query.Has(GrafanaMatchParamName) {
			query.Set(GrafanaMatchParamName, DummyMatchSelector)
		}

		req.URL.RawQuery = query.Encode()
		h.handleTenantInjection(res, req, idToken, GrafanaMatchParamName)
		return
	}

	// Allow unrestricted access in development mode
	if h.config.DevMode {
		h.HandleProxyBypass(res, req)
		return
	}

	res.Fail("Unauthorized: authentication required", http.StatusUnauthorized)
}
