package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/k0rdent/kof/kof-operator/internal/models/response"
	"github.com/k0rdent/kof/kof-operator/internal/server"
)

const BasicInternalErrorMessage = "Something went wrong"

func internalError(res *server.Response, errorMessage string) {
	res.Writer.Header().Set("Content-Type", "application/json")
	res.SetStatus(http.StatusInternalServerError)

	response := response.NewBasicResponse(false, errorMessage)
	text, err := json.Marshal(response)
	if err != nil {
		res.Logger.Error(err, "Failed to marshal basic response")
		text = []byte(BasicInternalErrorMessage)
	}

	if _, err = fmt.Fprintln(res.Writer, text); err != nil {
		res.Logger.Error(err, "Cannot write response")
	}
}

func sendResponse(res *server.Response, data any) {
	jsonResponse, err := json.Marshal(data)
	if err != nil {
		res.Logger.Error(err, "failed to marshal response")
		internalError(res, BasicInternalErrorMessage)
	}

	res.Writer.Header().Set("Content-Type", "application/json")
	res.SetStatus(http.StatusOK)

	if _, err = fmt.Fprintln(res.Writer, string(jsonResponse)); err != nil {
		res.Logger.Error(err, "Cannot write response")
	}
}
