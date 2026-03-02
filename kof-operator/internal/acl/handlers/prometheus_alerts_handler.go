package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/k0rdent/kof/kof-operator/internal/server"
	"github.com/k0rdent/kof/kof-operator/internal/server/helper"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
)

type AlertsResponse struct {
	Status string       `json:"status"`
	Data   AlertsResult `json:"data"`
}

type AlertsResult struct {
	Alerts []*v1.Alert `json:"alerts"`
}

func (h *PromxyHandler) ProxyAlertsWithTenantFiltration(res *server.Response, req *http.Request) {
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

		h.handleAlertsTenantFiltration(res, req, idToken)
		return
	}

	// Allow unrestricted access in development mode
	if h.config.DevMode {
		h.HandleProxyBypass(res, req)
		return
	}

	res.Fail("Unauthorized: authentication required", http.StatusUnauthorized)
}

func (h *PromxyHandler) handleAlertsTenantFiltration(res *server.Response, req *http.Request, idToken *oidc.IDToken) {
	ctx := req.Context()

	// Extract tenant ID from authenticated user's token
	tenantID, err := ExtractTenantIDFromToken(idToken)
	if err != nil {
		res.Fail(fmt.Sprintf("failed to extract tenant ID: %v", err), http.StatusUnauthorized)
		return
	}

	url := BuildURL(h.config.Scheme, h.config.Host, req.URL.Path, req.URL.Query().Encode())
	resp, err := ProxyRequest(ctx, url, req.Method, nil)
	if err != nil {
		res.Fail(fmt.Sprintf("failed to proxy request: %v", err), http.StatusInternalServerError)
		return
	}

	defer func() {
		if err := resp.Body.Close(); err != nil {
			res.Logger.Error(err, "failed to close response body")
		}
	}()

	if resp.StatusCode != http.StatusOK {
		res.Fail(fmt.Sprintf("received non-OK response: %s", resp.Status), resp.StatusCode)
		return
	}

	alerts := new(AlertsResponse)
	if err := json.NewDecoder(resp.Body).Decode(alerts); err != nil {
		res.Fail(fmt.Sprintf("failed to decode alerts: %v", err), http.StatusInternalServerError)
		return
	}

	filteredAlerts := filterAlertsByTenant(alerts.Data.Alerts, tenantID)
	alerts.Data.Alerts = filteredAlerts
	res.SendObj(alerts, http.StatusOK)
}

func filterAlertsByTenant(alerts []*v1.Alert, tenantID string) []*v1.Alert {
	matchingAlerts := make([]*v1.Alert, 0, len(alerts))

	if len(alerts) == 0 {
		return matchingAlerts
	}

	for _, alert := range alerts {
		if alert == nil || len(alert.Labels) == 0 {
			continue
		}

		if val, ok := alert.Labels[TenantLabelName]; ok && string(val) == tenantID {
			matchingAlerts = append(matchingAlerts, alert)
		}
	}
	return matchingAlerts
}
