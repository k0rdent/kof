package handlers

import (
	"fmt"
	"net/http"

	"github.com/k0rdent/kof/kof-operator/internal/server"
)

func NotFoundHandler(res *server.Response, req *http.Request) {
	res.Writer.Header().Set("Content-Type", "text/plain")
	res.SetStatus(http.StatusNotFound)
	_, err := fmt.Fprintln(res.Writer, "404 - Page not found")
	if err != nil {
		res.Logger.Error(err, "Cannot write response")
	}
}
