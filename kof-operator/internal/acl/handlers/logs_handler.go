package handlers

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/k0rdent/kof/kof-operator/internal/server"
)

// logsHandler handles Logs API requests with tenant isolation.
type logsHandler struct {
	config Config
}

func NewLogsHandler(cfg Config) Proxy {
	return &logsHandler{config: cfg}
}

func (h *logsHandler) AdminEmail() string { return h.config.AdminEmail }
func (h *logsHandler) IsDevMode() bool    { return h.config.DevMode }
func (h *logsHandler) Schema() string     { return h.config.Scheme }
func (h *logsHandler) Host() string       { return h.config.Host }
func (h *logsHandler) PathPrefix() string { return "" }

func (h *logsHandler) HandleTenantInjection(res *server.Response, req *http.Request, idToken *oidc.IDToken) {
	tenantID, err := ExtractTenantIDFromToken(idToken)
	if err != nil {
		res.Fail(fmt.Sprintf("failed to extract tenant ID: %v", err), http.StatusUnauthorized)
		return
	}

	query := req.URL.Query()
	path := strings.TrimPrefix(req.URL.Path, "/logs")

	query.Set("extra_filters", fmt.Sprintf("%s:=\"%s\"", TenantLabelName, tenantID))
	logsURL := BuildURL(h.config.Scheme, h.config.Host, path, query.Encode())

	var body io.Reader
	if req.Method == http.MethodPost {
		body = req.Body
	}

	if err := StreamProxyRequest(req.Context(), logsURL, req.Method, body, res.Writer); err != nil {
		res.Logger.Error(err, "failed to proxy request")
		http.Error(res.Writer, "unable to make request", http.StatusInternalServerError)
		return
	}
}
