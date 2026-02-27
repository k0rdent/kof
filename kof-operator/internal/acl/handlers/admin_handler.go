package handlers

import (
	"net/http"

	"github.com/k0rdent/kof/kof-operator/internal/server"
	"github.com/k0rdent/kof/kof-operator/internal/server/helper"
)

func (h *PromxyHandler) HandleAdminProxy(res *server.Response, req *http.Request) {
	ctx := req.Context()
	defer func() {
		if err := req.Body.Close(); err != nil {
			res.Logger.Error(err, "failed to close request body")
		}
	}()

	if idToken, ok := helper.GetIDToken(ctx); ok {
		if isAdminUser(idToken, h.config.AdminEmail) {
			h.HandleProxyBypass(res, req)
			return
		}
	}

	// Allow unrestricted access in development mode
	if h.config.DevMode {
		h.HandleProxyBypass(res, req)
		return
	}

	res.Fail("Forbidden: admin access required", http.StatusForbidden)
}
