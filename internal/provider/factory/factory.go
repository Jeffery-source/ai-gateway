package factory

import (
	"fmt"

	"ai_gateway/internal/config"
	"ai_gateway/internal/provider"
	"ai_gateway/internal/provider/mock"
	"ai_gateway/internal/provider/openaicompatible"
)

func Create(
	name string,
	cfg config.ProviderConfig,
) (provider.ModelProvider, error) {

	switch cfg.Type {

	case "openai-compatible":

		return openaicompatible.NewProvider(
			cfg.APIKey,
			cfg.BaseURL,
		), nil

	case "mock":

		return mock.NewProvider(), nil

	default:

		return nil, fmt.Errorf(
			"unsupported provider type: %s",
			cfg.Type,
		)
	}
}
