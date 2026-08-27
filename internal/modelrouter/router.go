package modelrouter

import (
	"ai_gateway/internal/errors"
	"ai_gateway/internal/provider"
	"fmt"
	"strings"
)

type Router struct {
	providers map[string]provider.ModelProvider
	models    map[string]ModelRoute
}

func New() *Router {
	return &Router{
		providers: make(map[string]provider.ModelProvider),
		models:    make(map[string]ModelRoute),
	}
}

func (r *Router) Register(
	name string,
	p provider.ModelProvider,
) {
	r.providers[name] = p
}

func (r *Router) GetProvider(
	name string,
) (provider.ModelProvider, error) {

	p, ok := r.providers[name]

	if !ok {
		return nil, fmt.Errorf(
			"provider not found: %s",
			name,
		)
	}

	return p, nil
}

type ModelRoute struct {
	Provider string
	Model    string
}

func (r *Router) RegisterModel(
	model string,
	route ModelRoute,
) {
	r.models[model] = route
}

func (r *Router) Resolve(
	model string,
) (provider.ModelProvider, string, error) {

	model = strings.TrimSpace(model)

	route, ok := r.models[model]

	if !ok {
		return nil, "", &errors.Error{
			HTTPStatus: 400,
			Message: fmt.Sprintf(
				"model '%s' is not supported",
				model,
			),
			Type: errors.CodeInvalidRequest,
			Code: "model_not_supported",
		}
	}

	p, ok := r.providers[route.Provider]

	if !ok {
		return nil, "", &errors.Error{
			HTTPStatus: 500,
			Message: fmt.Sprintf(
				"provider '%s' is not configured",
				route.Provider,
			),
			Type: errors.CodeInternalError,
			Code: "provider_not_configured",
		}
	}

	return p, route.Model, nil
}
