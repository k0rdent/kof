package handlers

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/k0rdent/kof/kof-operator/internal/server"
)

// VlogxyHandler handles Vlogxy API requests with tenant isolation.
type VlogxyHandler struct {
	config Config
}

func NewVlogxyHandler(cfg Config) Proxy {
	return &VlogxyHandler{config: cfg}
}

func (h *VlogxyHandler) AdminEmail() string { return h.config.AdminEmail }
func (h *VlogxyHandler) IsDevMode() bool    { return h.config.DevMode }
func (h *VlogxyHandler) Schema() string     { return h.config.Scheme }
func (h *VlogxyHandler) Host() string       { return h.config.Host }

func (h *VlogxyHandler) HandleTenantInjection(res *server.Response, req *http.Request, idToken *oidc.IDToken) {
	tenantID, err := ExtractTenantIDFromToken(idToken)
	if err != nil {
		res.Fail(fmt.Sprintf("failed to extract tenant ID: %v", err), http.StatusUnauthorized)
		return
	}

	query := req.URL.Query()
	path := strings.TrimPrefix(req.URL.Path, "/vlogxy")

	query.Set("extra_filters", fmt.Sprintf("%s:=\"%s\"", TenantLabelName, tenantID))
	vlogxyURL := BuildURL(h.config.Scheme, h.config.Host, path, query.Encode())

	var body io.Reader
	if req.Method == http.MethodPost {
		body = req.Body
	}

	if err := StreamProxyRequest(req.Context(), vlogxyURL, req.Method, body, res.Writer); err != nil {
		res.Logger.Error(err, "failed to proxy request to vlogxy")
		http.Error(res.Writer, "unable to make request", http.StatusInternalServerError)
		return
	}
}
