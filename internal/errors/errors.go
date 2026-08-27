package errors

type Code string

const (
	CodeInvalidRequest   Code = "invalid_request"
	CodeAuthentication   Code = "authentication_error"
	CodePermissionDenied Code = "permission_denied"
	CodeNotFound         Code = "not_found"
	CodeRateLimit        Code = "rate_limit"
	CodeProviderError    Code = "provider_error"
	CodeProviderTimeout  Code = "provider_timeout"
	CodeInternalError    Code = "internal_error"
)

type Error struct {
	HTTPStatus int
	Message    string
	Type       Code
	Code       string
}

func (e *Error) Error() string {
	return e.Message
}

type Response struct {
	Error ErrorBody `json:"error"`
}

type ErrorBody struct {
	Message string `json:"message"`
	Type    Code   `json:"type"`
	Code    string `json:"code,omitempty"`
}
