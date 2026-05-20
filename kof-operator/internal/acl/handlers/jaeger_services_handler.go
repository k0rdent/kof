package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/k0rdent/kof/kof-operator/internal/server"
)

type JaegerServiceResponse struct {
	Data   []string `json:"data"`
	Errors string   `json:"errors,omitempty"`
	Limit  int      `json:"limit"`
	Offset int      `json:"offset"`
	Total  int      `json:"total"`
}

type JaegerServicesHandler struct {
	config Config
}

func NewJaegerServicesHandler(config Config) Proxy {
	return &JaegerServicesHandler{config: config}
}

func (h *JaegerServicesHandler) AdminEmail() string { return h.config.AdminEmail }
func (h *JaegerServicesHandler) IsDevMode() bool    { return h.config.DevMode }
func (h *JaegerServicesHandler) Schema() string     { return h.config.Scheme }
func (h *JaegerServicesHandler) Host() string       { return h.config.Host }

func (h *JaegerServicesHandler) HandleTenantInjection(res *server.Response, req *http.Request, idToken *oidc.IDToken) {
	tenantID, err := ExtractTenantIDFromToken(idToken)
	if err != nil {
		res.Fail(fmt.Sprintf("failed to extract tenant ID: %v", err), http.StatusUnauthorized)
		return
	}

	var body io.Reader
	if req.Method == http.MethodPost {
		body = req.Body
	}

	query := req.URL.Query()
	query.Set("extra_filters", fmt.Sprintf("\"resource_attr:%s\":=\"%s\"", TenantLabelName, tenantID))
	query.Set("query", `* | uniq by ("resource_attr:service.name")`)

	newPath := "/select/logsql/query"
	backendURL := BuildURL(h.config.Scheme, h.config.Host, newPath, query.Encode())

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

	serviceNames := make([]string, 0)
	dec := json.NewDecoder(vtResp.Body)
	for {
		var entry map[string]string

		if err := dec.Decode(&entry); err != nil {
			if err == io.EOF {
				break
			}
			res.Logger.Error(err, "failed to decode response body")
			http.Error(res.Writer, "unable to decode response", http.StatusInternalServerError)
			return
		}

		if serviceName, ok := entry["resource_attr:service.name"]; ok {
			serviceNames = append(serviceNames, serviceName)
		}
	}

	response := JaegerServiceResponse{
		Data:  serviceNames,
		Total: len(serviceNames),
	}

	if err := json.NewEncoder(res.Writer).Encode(response); err != nil {
		res.Logger.Error(err, "failed to encode response")
		http.Error(res.Writer, "unable to encode response", http.StatusInternalServerError)
		return
	}
}
