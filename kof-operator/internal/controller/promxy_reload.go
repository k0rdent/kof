package controller

import (
	"net/http"
	"strings"
)

func ReloadPromxyConfig(endpoint string) error {
	res, err := http.Post(endpoint, "application/json", strings.NewReader(""))
	if err != nil {
		return err
	}
	return res.Body.Close()
}
