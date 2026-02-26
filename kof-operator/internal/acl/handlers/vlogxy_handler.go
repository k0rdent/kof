package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/k0rdent/kof/kof-operator/internal/server"
	"github.com/k0rdent/kof/kof-operator/internal/server/helper"
)

// VlogxyHandler handles Vlogxy API requests with tenant isolation.
type VlogxyHandler struct {
	config *Config
}

func NewVlogxyHandler(cfg Config) *VlogxyHandler {
	return &VlogxyHandler{config: &cfg}
}

func (h *VlogxyHandler) ProxyLogsWithTenantInjection(res *server.Response, req *http.Request) {
	ctx := req.Context()
	defer func() {
		if err := req.Body.Close(); err != nil {
			res.Logger.Error(err, "failed to close request body")
		}
	}()

	// Check for authenticated user with ID token
	if idToken, ok := helper.GetIDToken(ctx); ok {
		if isAdminUser(idToken, h.config.AdminEmail) {
			h.HandleLogsProxyBypass(res, req)
			return
		}

		h.HandleLogsTenantInjection(res, req, idToken)
		return
	}

	// Allow unrestricted access in development mode
	if h.config.DevMode {
		h.HandleLogsProxyBypass(res, req)
		return
	}

	res.Fail("Unauthorized: authentication required", http.StatusUnauthorized)
}

func (h *VlogxyHandler) HandleLogsProxyBypass(res *server.Response, req *http.Request) {
	query := req.URL.Query()
	path := strings.TrimPrefix(req.URL.Path, "/vlogxy")
	vlogxyURL := BuildURL(h.config.Scheme, h.config.Host, path, query.Encode())

	if err := StreamProxyRequest(req.Context(), vlogxyURL, req.Method, res.Writer); err != nil {
		res.Logger.Error(err, "failed to proxy request to vlogxy")
		http.Error(res.Writer, "unable to make request", http.StatusInternalServerError)
		return
	}
}

func (h *VlogxyHandler) HandleLogsTenantInjection(res *server.Response, req *http.Request, idToken *oidc.IDToken) {
	// Extract tenant ID from authenticated user's token
	tenantID, err := ExtractTenantIDFromToken(idToken)
	if err != nil {
		res.Fail(fmt.Sprintf("failed to extract tenant ID: %v", err), http.StatusUnauthorized)
		return
	}

	query := req.URL.Query()
	path := strings.TrimPrefix(req.URL.Path, "/vlogxy")

	// Inject tenant label into the query
	query.Set("extra_filters", fmt.Sprintf("tenantId:=\"%s\"", tenantID))
	vlogxyURL := BuildURL(h.config.Scheme, h.config.Host, path, query.Encode())

	if err := StreamProxyRequest(req.Context(), vlogxyURL, req.Method, res.Writer); err != nil {
		res.Logger.Error(err, "failed to proxy request to vlogxy")
		http.Error(res.Writer, "unable to make request", http.StatusInternalServerError)
		return
	}
}
