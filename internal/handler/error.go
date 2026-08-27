package handler

import (
	"encoding/json"
	"net/http"

	"ai_gateway/internal/errors"
)

type errorResponse struct {
	Error errorBody `json:"error"`
}

type errorBody struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code"`
}

func WriteError(w http.ResponseWriter, err error) {

	status := http.StatusInternalServerError

	var gatewayErr *errors.Error

	if e, ok := err.(*errors.Error); ok {
		gatewayErr = e
		status = e.HTTPStatus
	} else {
		gatewayErr = &errors.Error{
			HTTPStatus: http.StatusInternalServerError,
			Message:    "Internal Server Error",
			Type:       errors.CodeInternalError,
			Code:       "internal_error",
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	response := errors.Response{
		Error: errors.ErrorBody{
			Message: gatewayErr.Message,
			Type:    gatewayErr.Type,
			Code:    gatewayErr.Code,
		},
	}

	_ = json.NewEncoder(w).Encode(response)
}
