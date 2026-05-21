package controller

import (
	"context"
	"net/http"
	"strings"

	"github.com/k0rdent/kof/kof-operator/internal/telemetry"
)

func ReloadPromxyConfig(ctx context.Context, endpoint string) error {
	client := &http.Client{Transport: telemetry.NewTransport(nil)}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(""))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	return res.Body.Close()
}
