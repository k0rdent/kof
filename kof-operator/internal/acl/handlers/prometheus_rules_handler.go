package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/k0rdent/kof/kof-operator/internal/server"
	"github.com/k0rdent/kof/kof-operator/internal/server/helper"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

type RulesResponse struct {
	Status string      `json:"status"`
	Data   RulesResult `json:"data"`
}

type RulesResult struct {
	Groups []*RuleGroup `json:"groups"`
}

type RuleGroup struct {
	EvaluationTime float64      `json:"evaluationTime"`
	File           string       `json:"file"`
	Interval       float64      `json:"interval"`
	LastEvaluation string       `json:"lastEvaluation"`
	Limit          int          `json:"limit"`
	Name           string       `json:"name"`
	Rules          []*AlertRule `json:"rules"`
}

type AlertRule struct {
	State          v1.AlertState     `json:"state"`
	Name           string            `json:"name"`
	Query          string            `json:"query"`
	Duration       float64           `json:"duration"`
	Labels         map[string]string `json:"labels"`
	Annotations    map[string]string `json:"annotations"`
	Alerts         []*v1.Alert       `json:"alerts"`
	Health         string            `json:"health"`
	EvaluationTime float64           `json:"evaluationTime"`
	LastEvaluation string            `json:"lastEvaluation"`
	Type           v1.RuleType       `json:"type"`
}

func (h *PromxyHandler) ProxyRulesWithTenantFiltration(res *server.Response, req *http.Request) {
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

		h.handleRulesTenantFiltration(res, req, idToken)
		return
	}

	// Allow unrestricted access in development mode
	if h.config.DevMode {
		h.HandleProxyBypass(res, req)
		return
	}

	res.Fail("Unauthorized: authentication required", http.StatusUnauthorized)
}

func (h *PromxyHandler) handleRulesTenantFiltration(res *server.Response, req *http.Request, idToken *oidc.IDToken) {
	ctx := req.Context()

	// Extract tenant ID from authenticated user's token
	tenantID, err := ExtractTenantIDFromToken(idToken)
	if err != nil {
		res.Fail(fmt.Sprintf("failed to extract tenant ID: %v", err), http.StatusUnauthorized)
		return
	}

	url := BuildURL(h.config.Scheme, h.config.Host, req.URL.Path, req.URL.Query().Encode())
	resp, err := ProxyRequest(ctx, url, req.Method)
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

	ruleResponse := new(RulesResponse)
	if err := json.NewDecoder(resp.Body).Decode(ruleResponse); err != nil {
		res.Fail(fmt.Sprintf("failed to decode rules: %v", err), http.StatusInternalServerError)
		return
	}

	// Filter alerts based on tenant ID
	for _, group := range ruleResponse.Data.Groups {
		for _, rule := range group.Rules {
			if rule.Type != v1.RuleTypeAlerting {
				continue
			}

			isFiring := false
			filteredAlerts := make([]*v1.Alert, 0, len(rule.Alerts))

			for _, alert := range rule.Alerts {
				if len(alert.Labels) == 0 {
					continue
				}

				if val, ok := alert.Labels[model.LabelName(TenantLabelName)]; ok && string(val) == tenantID {
					if alert.State == v1.AlertStateFiring {
						isFiring = true
					}

					filteredAlerts = append(filteredAlerts, alert)
				}
			}

			if len(filteredAlerts) == 0 {
				rule.State = v1.AlertStateInactive
			} else if isFiring {
				rule.State = v1.AlertStateFiring
			} else {
				rule.State = v1.AlertStatePending
			}

			rule.Alerts = filteredAlerts
		}
	}

	res.SendObj(ruleResponse, http.StatusOK)
}
