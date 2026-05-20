package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/k0rdent/kof/kof-operator/internal/server"
)

type JaegerTraceResponse struct {
	Data   []*JaegerTrace `json:"data"`
	Errors any            `json:"errors"`
	Limit  int            `json:"limit"`
	Offset int            `json:"offset"`
	Total  int            `json:"total"`
}

type JaegerTrace struct {
	TraceID   string             `json:"traceID"`
	Spans     json.RawMessage    `json:"spans"`
	Processes map[string]Process `json:"processes"`
}

type Process struct {
	ServiceName string     `json:"serviceName"`
	Tags        []KeyValue `json:"tags"`
}

type KeyValue struct {
	Key   string `json:"key"`
	Type  string `json:"type"`
	Value any    `json:"value"`
}

type JaegerTraceHandler struct {
	config Config
}

func NewJaegerTraceHandler(config Config) Proxy {
	return &JaegerTraceHandler{config: config}
}

func (h *JaegerTraceHandler) AdminEmail() string { return h.config.AdminEmail }
func (h *JaegerTraceHandler) IsDevMode() bool    { return h.config.DevMode }
func (h *JaegerTraceHandler) Schema() string     { return h.config.Scheme }
func (h *JaegerTraceHandler) Host() string       { return h.config.Host }

func (h *JaegerTraceHandler) HandleTenantInjection(res *server.Response, req *http.Request, idToken *oidc.IDToken) {
	// Extract tenant ID from authenticated user's token
	tenantID, err := ExtractTenantIDFromToken(idToken)
	if err != nil {
		res.Fail(fmt.Sprintf("failed to extract tenant ID: %v", err), http.StatusUnauthorized)
		return
	}

	path := strings.TrimPrefix(req.URL.Path, "/traces")
	backendURL := BuildURL(h.config.Scheme, h.config.Host, path, req.URL.Query().Encode())

	var body io.Reader
	if req.Method == http.MethodPost {
		body = req.Body
	}

	vtResp, err := ProxyRequest(req.Context(), backendURL, req.Method, body)
	if err != nil {
		res.Logger.Error(err, "failed to proxy request to traces")
		http.Error(res.Writer, "unable to make request", http.StatusInternalServerError)
		return
	}

	defer func() {
		if err := vtResp.Body.Close(); err != nil {
			res.Logger.Error(err, "failed to close response body")
		}
	}()

	if vtResp.StatusCode != http.StatusOK {
		res.Fail(fmt.Sprintf("received non-OK response: %s", vtResp.Status), vtResp.StatusCode)
		return
	}

	result := new(JaegerTraceResponse)

	rawBody, err := io.ReadAll(vtResp.Body)
	if err != nil {
		res.Logger.Error(err, "failed to read response body")
		http.Error(res.Writer, "unable to read response", http.StatusInternalServerError)
		return
	}

	if err := json.Unmarshal(rawBody, result); err != nil {
		res.Logger.Error(err, "failed to unmarshal response body")
		http.Error(res.Writer, "unable to parse response", http.StatusInternalServerError)
		return
	}

	filteredData := make([]*JaegerTrace, 0, len(result.Data))
	for _, trace := range result.Data {
		if hasProcessTagWithValue(trace, TenantLabelName, tenantID) {
			filteredData = append(filteredData, trace)
		}
	}

	result.Data = filteredData
	result.Total = len(filteredData)
	if err := json.NewEncoder(res.Writer).Encode(result); err != nil {
		res.Logger.Error(err, "failed to encode response")
		http.Error(res.Writer, "unable to encode response", http.StatusInternalServerError)
		return
	}
}

func hasProcessTagWithValue(trace *JaegerTrace, key string, value any) bool {
	for _, proc := range trace.Processes {
		for _, tag := range proc.Tags {
			if tag.Key == key && tag.Value == value {
				return true
			}
		}
	}
	return false
}
