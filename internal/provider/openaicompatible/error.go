package openaicompatible

import (
	"context"
	"net/http"

	"ai_gateway/internal/errors"
)

func mapProviderRequestError(err error) error {

	if err == nil {
		return nil
	}

	if err == context.Canceled {
		return err
	}

	if err == context.DeadlineExceeded {
		return &errors.Error{
			HTTPStatus: http.StatusGatewayTimeout,
			Message:    "Provider request timeout",
			Type:       errors.CodeProviderTimeout,
			Code:       "provider_timeout",
		}
	}

	return &errors.Error{
		HTTPStatus: http.StatusBadGateway,
		Message:    "Provider request failed",
		Type:       errors.CodeProviderError,
		Code:       "provider_unavailable",
	}
}

func mapProviderError(
	status int,
	message string,
	code string,
) error {

	switch status {

	case http.StatusUnauthorized:
		return &errors.Error{
			HTTPStatus: http.StatusUnauthorized,
			Message:    message,
			Type:       errors.CodeAuthentication,
			Code:       code,
		}

	case http.StatusForbidden:
		return &errors.Error{
			HTTPStatus: http.StatusForbidden,
			Message:    message,
			Type:       errors.CodePermissionDenied,
			Code:       code,
		}

	case http.StatusNotFound:
		return &errors.Error{
			HTTPStatus: http.StatusNotFound,
			Message:    message,
			Type:       errors.CodeNotFound,
			Code:       code,
		}

	case http.StatusTooManyRequests:
		return &errors.Error{
			HTTPStatus: http.StatusTooManyRequests,
			Message:    message,
			Type:       errors.CodeRateLimit,
			Code:       code,
		}

	case http.StatusRequestTimeout:
		return &errors.Error{
			HTTPStatus: http.StatusGatewayTimeout,
			Message:    message,
			Type:       errors.CodeProviderTimeout,
			Code:       code,
		}

	default:
		return &errors.Error{
			HTTPStatus: http.StatusBadGateway,
			Message:    message,
			Type:       errors.CodeProviderError,
			Code:       code,
		}
	}
}
