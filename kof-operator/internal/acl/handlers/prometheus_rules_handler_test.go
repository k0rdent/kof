package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"

	"github.com/k0rdent/kof/kof-operator/internal/server"
	"github.com/k0rdent/kof/kof-operator/internal/server/helper"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	ctrl "sigs.k8s.io/controller-runtime"
)

var _ = Describe("HandleRulesWithTenantFiltration", func() {
	buildAlert := func(tenant string, state v1.AlertState) *v1.Alert {
		return &v1.Alert{
			State: state,
			Labels: model.LabelSet{
				model.LabelName(TenantLabelName): model.LabelValue(tenant),
			},
		}
	}

	buildRule := func(name string, ruleType v1.RuleType, alerts ...*v1.Alert) *AlertRule {
		return &AlertRule{
			Type:   ruleType,
			Name:   name,
			Alerts: alerts,
		}
	}

	buildGroup := func(name string, rules ...*AlertRule) *RuleGroup {
		return &RuleGroup{
			Name:  name,
			Rules: rules,
		}
	}

	buildResponse := func(groups ...*RuleGroup) RulesResponse {
		return RulesResponse{
			Status: "success",
			Data: RulesResult{
				Groups: groups,
			},
		}
	}

	var (
		req              *http.Request
		res              *server.Response
		mockPromxy       *httptest.Server
		handler          *PromxyHandler
		logger           = ctrl.Log.WithName("test")
		responseToReturn RulesResponse
	)

	BeforeEach(func() {
		mockPromxy = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			Expect(json.NewEncoder(w).Encode(responseToReturn)).NotTo(HaveOccurred())
		}))

		parsedURL, err := url.Parse(mockPromxy.URL)
		Expect(err).NotTo(HaveOccurred())

		handler = NewHandler(Config{
			Host:   parsedURL.Host,
			Scheme: "http",
		})

		req = httptest.NewRequest(http.MethodGet, "/api/v1/rules", nil)
		res = &server.Response{
			Writer: httptest.NewRecorder(),
			Logger: &logger,
		}
	})

	AfterEach(func() {
		mockPromxy.Close()
	})

	It("filters alerts and marks rule firing when tenant has a firing alert", func() {
		req = httptest.NewRequest(http.MethodGet, "/api/v1/rules", nil)

		alertRule := buildRule("TestRule", v1.RuleTypeAlerting,
			buildAlert("test-tenant", v1.AlertStateFiring),
			buildAlert("other-tenant", v1.AlertStatePending),
		)
		group := buildGroup("TestGroup", alertRule)
		responseToReturn = buildResponse(group)

		idToken := MockIDToken(map[string]any{
			"email":          "user@example.com",
			"name":           "Test User",
			"groups":         []any{"tenant:test-tenant"},
			"email_verified": true,
		})

		ctx := context.WithValue(req.Context(), helper.IdTokenContextKey, idToken)
		req = req.WithContext(ctx)

		handler.ProxyRulesWithTenantFiltration(res, req)

		recorder := res.Writer.(*httptest.ResponseRecorder)
		Expect(recorder.Code).To(Equal(http.StatusOK))

		var rr RulesResponse
		Expect(json.Unmarshal(recorder.Body.Bytes(), &rr)).To(Succeed())

		Expect(rr.Data.Groups).To(HaveLen(1))
		g := rr.Data.Groups[0]
		Expect(g.Rules).To(HaveLen(1))
		rule := g.Rules[0]

		Expect(rule.Type).To(Equal(v1.RuleTypeAlerting))
		Expect(rule.State).To(Equal(v1.AlertStateFiring))
		Expect(rule.Alerts).To(HaveLen(1))
		Expect(string(rule.Alerts[0].Labels[model.LabelName(TenantLabelName)])).To(Equal("test-tenant"))
	})

	It("marks rule pending when tenant has matching non-firing alerts", func() {
		req = httptest.NewRequest(http.MethodGet, "/api/v1/rules", nil)

		alertRule := buildRule("TestRule", v1.RuleTypeAlerting,
			buildAlert("test-tenant", v1.AlertStatePending),
			buildAlert("test-tenant", v1.AlertStatePending),
		)
		group := buildGroup("TestGroup", alertRule)
		responseToReturn = buildResponse(group)

		idToken := MockIDToken(map[string]any{
			"email":          "user@example.com",
			"name":           "Test User",
			"groups":         []any{"tenant:test-tenant"},
			"email_verified": true,
		})

		ctx := context.WithValue(req.Context(), helper.IdTokenContextKey, idToken)
		req = req.WithContext(ctx)

		handler.ProxyRulesWithTenantFiltration(res, req)

		recorder := res.Writer.(*httptest.ResponseRecorder)
		Expect(recorder.Code).To(Equal(http.StatusOK))

		var rr RulesResponse
		Expect(json.Unmarshal(recorder.Body.Bytes(), &rr)).To(Succeed())

		rule := rr.Data.Groups[0].Rules[0]
		Expect(rule.State).To(Equal(v1.AlertStatePending))
		Expect(rule.Alerts).To(HaveLen(2))
	})

	It("marks rule inactive when tenant has no matching alerts", func() {
		req = httptest.NewRequest(http.MethodGet, "/api/v1/rules", nil)

		alertRule := buildRule("TestRule", v1.RuleTypeAlerting,
			buildAlert("other-tenant", v1.AlertStateFiring),
		)
		group := buildGroup("TestGroup", alertRule)
		responseToReturn = buildResponse(group)

		idToken := MockIDToken(map[string]any{
			"email":          "user@example.com",
			"name":           "Test User",
			"groups":         []any{"tenant:test-tenant"},
			"email_verified": true,
		})

		ctx := context.WithValue(req.Context(), helper.IdTokenContextKey, idToken)
		req = req.WithContext(ctx)

		handler.ProxyRulesWithTenantFiltration(res, req)

		recorder := res.Writer.(*httptest.ResponseRecorder)
		Expect(recorder.Code).To(Equal(http.StatusOK))

		var rr RulesResponse
		Expect(json.Unmarshal(recorder.Body.Bytes(), &rr)).To(Succeed())

		rule := rr.Data.Groups[0].Rules[0]
		Expect(rule.State).To(Equal(v1.AlertStateInactive))
		Expect(rule.Alerts).To(BeEmpty())
	})

	It("handles multiple groups and rules", func() {
		req = httptest.NewRequest(http.MethodGet, "/api/v1/rules", nil)

		alertRule1 := buildRule("Rule1", v1.RuleTypeAlerting,
			buildAlert("test-tenant", v1.AlertStateFiring),
		)
		alertRule2 := buildRule("Rule2", v1.RuleTypeAlerting,
			buildAlert("test-tenant", v1.AlertStatePending),
		)
		group1 := buildGroup("Group1", alertRule1, alertRule2)

		alertRule3 := buildRule("Rule3", v1.RuleTypeAlerting,
			buildAlert("other-tenant", v1.AlertStateFiring),
		)
		group2 := buildGroup("Group2", alertRule3)

		responseToReturn = buildResponse(group1, group2)

		idToken := MockIDToken(map[string]any{
			"groups": []any{"tenant:test-tenant"},
		})

		ctx := context.WithValue(req.Context(), helper.IdTokenContextKey, idToken)
		req = req.WithContext(ctx)

		handler.ProxyRulesWithTenantFiltration(res, req)

		recorder := res.Writer.(*httptest.ResponseRecorder)
		Expect(recorder.Code).To(Equal(http.StatusOK))

		var rr RulesResponse
		Expect(json.Unmarshal(recorder.Body.Bytes(), &rr)).To(Succeed())

		Expect(rr.Data.Groups).To(HaveLen(2))
		Expect(rr.Data.Groups[0].Name).To(Equal("Group1"))
		Expect(rr.Data.Groups[0].Rules).To(HaveLen(2))
	})

	It("handles mixed alert states in single rule", func() {
		req = httptest.NewRequest(http.MethodGet, "/api/v1/rules", nil)

		alertRule := buildRule("TestRule", v1.RuleTypeAlerting,
			buildAlert("test-tenant", v1.AlertStateFiring),
			buildAlert("test-tenant", v1.AlertStatePending),
			buildAlert("test-tenant", v1.AlertStateInactive),
		)
		group := buildGroup("TestGroup", alertRule)
		responseToReturn = buildResponse(group)

		idToken := MockIDToken(map[string]any{
			"groups": []any{"tenant:test-tenant"},
		})

		ctx := context.WithValue(req.Context(), helper.IdTokenContextKey, idToken)
		req = req.WithContext(ctx)

		handler.ProxyRulesWithTenantFiltration(res, req)

		recorder := res.Writer.(*httptest.ResponseRecorder)
		Expect(recorder.Code).To(Equal(http.StatusOK))

		var rr RulesResponse
		Expect(json.Unmarshal(recorder.Body.Bytes(), &rr)).To(Succeed())

		rule := rr.Data.Groups[0].Rules[0]
		Expect(rule.Alerts).To(HaveLen(3))
		Expect(rule.State).To(Equal(v1.AlertStateFiring))
	})

	It("returns empty alerts when no alerts match", func() {
		req = httptest.NewRequest(http.MethodGet, "/api/v1/rules", nil)

		alertRule := buildRule("TestRule", v1.RuleTypeAlerting,
			buildAlert("other-tenant", v1.AlertStateFiring),
		)
		group := buildGroup("TestGroup", alertRule)
		responseToReturn = buildResponse(group)

		idToken := MockIDToken(map[string]any{
			"groups": []any{"tenant:test-tenant"},
		})

		ctx := context.WithValue(req.Context(), helper.IdTokenContextKey, idToken)
		req = req.WithContext(ctx)

		handler.ProxyRulesWithTenantFiltration(res, req)

		recorder := res.Writer.(*httptest.ResponseRecorder)
		Expect(recorder.Code).To(Equal(http.StatusOK))

		var rr RulesResponse
		Expect(json.Unmarshal(recorder.Body.Bytes(), &rr)).To(Succeed())

		Expect(rr.Data.Groups).To(HaveLen(1))
		Expect(rr.Data.Groups[0].Rules).To(HaveLen(1))
		Expect(rr.Data.Groups[0].Rules[0].Alerts).To(BeEmpty())
	})

	It("returns the same response for non-alerting rules", func() {
		req = httptest.NewRequest(http.MethodGet, "/api/v1/rules", nil)

		recordingRule := buildRule("RecordingRule", v1.RuleTypeRecording)
		group := buildGroup("TestGroup", recordingRule)
		responseToReturn = buildResponse(group)

		idToken := MockIDToken(map[string]any{
			"groups": []any{"tenant:test-tenant"},
		})

		ctx := context.WithValue(req.Context(), helper.IdTokenContextKey, idToken)
		req = req.WithContext(ctx)

		handler.ProxyRulesWithTenantFiltration(res, req)

		recorder := res.Writer.(*httptest.ResponseRecorder)
		Expect(recorder.Code).To(Equal(http.StatusOK))

		var rr RulesResponse
		Expect(json.Unmarshal(recorder.Body.Bytes(), &rr)).To(Succeed())

		Expect(rr.Data.Groups).To(HaveLen(1))
		Expect(rr.Data.Groups[0].Rules).To(HaveLen(1))
		Expect(rr.Data.Groups[0].Rules[0].Type).To(Equal(v1.RuleTypeRecording))
		Expect(rr.Data.Groups[0].Rules[0].Alerts).To(BeEmpty())
	})
})
