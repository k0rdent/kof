package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/k0rdent/kof/kof-operator/internal/server"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
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

// MetricsRulesHandler handles Prometheus rules API requests with tenant isolation.
type MetricsRulesHandler struct {
	config Config
}

func NewMetricsRulesHandler(cfg Config) Proxy {
	return &MetricsRulesHandler{config: cfg}
}

func (h *MetricsRulesHandler) AdminEmail() string { return h.config.AdminEmail }
func (h *MetricsRulesHandler) IsDevMode() bool    { return h.config.DevMode }
func (h *MetricsRulesHandler) Schema() string     { return h.config.Scheme }
func (h *MetricsRulesHandler) Host() string       { return h.config.Host }
func (h *MetricsRulesHandler) PathPrefix() string { return "" }

func (h *MetricsRulesHandler) HandleTenantInjection(res *server.Response, req *http.Request, idToken *oidc.IDToken) {
	ctx := req.Context()

	tenantID, err := ExtractTenantIDFromToken(idToken)
	if err != nil {
		res.Fail(fmt.Sprintf("failed to extract tenant ID: %v", err), http.StatusUnauthorized)
		return
	}

	path := strings.TrimPrefix(req.URL.Path, "/metrics")
	backendURL := BuildURL(h.config.Scheme, h.config.Host, path, req.URL.Query().Encode())
	resp, err := ProxyRequest(ctx, backendURL, req.Method, nil)
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

			matchingAlerts := filterAlertsByTenant(rule.Alerts, tenantID)
			isFiring := slices.ContainsFunc(matchingAlerts, func(alert *v1.Alert) bool {
				return alert.State == v1.AlertStateFiring
			})

			if len(matchingAlerts) == 0 {
				rule.State = v1.AlertStateInactive
			} else if isFiring {
				rule.State = v1.AlertStateFiring
			} else {
				rule.State = v1.AlertStatePending
			}

			rule.Alerts = matchingAlerts
		}
	}

	res.SendObj(ruleResponse, http.StatusOK)
}
