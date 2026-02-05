package handlers

import (
	"net/http"

	"github.com/k0rdent/kof/kof-operator/internal/server"
	"github.com/k0rdent/kof/kof-operator/internal/server/helper"
)

func PrometheusLabelHandler(res *server.Response, req *http.Request) {
	ctx := req.Context()
	defer func() {
		if err := req.Body.Close(); err != nil {
			res.Logger.Error(err, "failed to close request body")
		}
	}()

	// Check for authenticated user with ID token
	if idToken, ok := helper.GetIDToken(ctx); ok {
		query := req.URL.Query()

		if !query.Has(GrafanaMatchParamName) {
			query.Set(GrafanaMatchParamName, "{tenantId=\"hello\"}")
		}

		req.URL.RawQuery = query.Encode()
		handleTenantInjection(res, req, idToken, GrafanaMatchParamName)
		return
	}

	// Allow unrestricted access in development mode
	if DevMode {
		handleAdminBypass(res, req)
		return
	}

	res.Fail("Unauthorized: authentication required", http.StatusUnauthorized)
}
