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

type JaegerTracesHandler struct {
	config Config
}

func NewJaegerTracesHandler(config Config) Proxy {
	return &JaegerTracesHandler{config: config}
}

func (h *JaegerTracesHandler) AdminEmail() string { return h.config.AdminEmail }
func (h *JaegerTracesHandler) IsDevMode() bool    { return h.config.DevMode }
func (h *JaegerTracesHandler) Schema() string     { return h.config.Scheme }
func (h *JaegerTracesHandler) Host() string       { return h.config.Host }
func (h *JaegerTracesHandler) PathPrefix() string { return "" }

func (h *JaegerTracesHandler) HandleTenantInjection(res *server.Response, req *http.Request, idToken *oidc.IDToken) {
	// Extract tenant ID from authenticated user's token
	tenantID, err := ExtractTenantIDFromToken(idToken)
	if err != nil {
		res.Fail(fmt.Sprintf("failed to extract tenant ID: %v", err), http.StatusUnauthorized)
		return
	}

	query := req.URL.Query()
	rawTags := query.Get("tags")
	tags := make(map[string]string)

	if rawTags != "" {
		if err := json.Unmarshal([]byte(rawTags), &tags); err != nil {
			res.Fail(fmt.Sprintf("failed to parse tags: %v", err), http.StatusInternalServerError)
			return
		}
	}

	tags["resource_attr:"+TenantLabelName] = tenantID

	tagsJSON, err := json.Marshal(tags)
	if err != nil {
		res.Fail(fmt.Sprintf("failed to marshal tags: %v", err), http.StatusInternalServerError)
		return
	}

	query.Set("tags", string(tagsJSON))

	path := strings.TrimPrefix(req.URL.Path, "/traces")
	url := BuildURL(h.config.Scheme, h.config.Host, path, query.Encode())

	var body io.Reader
	if req.Method == http.MethodPost {
		body = req.Body
	}

	if err := StreamProxyRequest(req.Context(), url, req.Method, body, res.Writer); err != nil {
		res.Logger.Error(err, "failed to proxy request to traces")
		http.Error(res.Writer, "unable to make request", http.StatusInternalServerError)
		return
	}
}
