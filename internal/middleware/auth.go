package middleware

import (
	"encoding/json"
	"net/http"
	"strings"

	"ai_gateway/internal/errors"
)

type APIKeyAuth struct {
	apiKey string
}

func NewAPIKeyAuth(apiKey string) *APIKeyAuth {
	return &APIKeyAuth{
		apiKey: apiKey,
	}
}

func (a *APIKeyAuth) Middleware(
	next http.Handler,
) http.Handler {

	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {

			auth := r.Header.Get("Authorization")

			if auth == "" {
				WriteAuthError(
					w,
					"missing authorization header",
				)
				return
			}

			const prefix = "Bearer "

			if !strings.HasPrefix(auth, prefix) {
				WriteAuthError(
					w,
					"invalid authorization header",
				)
				return
			}

			token := strings.TrimSpace(
				strings.TrimPrefix(auth, prefix),
			)

			if token == "" {
				WriteAuthError(
					w,
					"missing API key",
				)
				return
			}

			if token != a.apiKey {
				WriteAuthError(
					w,
					"invalid API key",
				)
				return
			}

			next.ServeHTTP(w, r)
		},
	)
}

func WriteAuthError(
	w http.ResponseWriter,
	message string,
) {

	w.Header().Set(
		"Content-Type",
		"application/json",
	)

	w.WriteHeader(http.StatusUnauthorized)

	response := map[string]interface{}{
		"error": map[string]interface{}{
			"message": message,
			"type":    string(errors.CodeAuthentication),
			"code":    "invalid_api_key",
		},
	}

	_ = json.NewEncoder(w).Encode(response)
}
