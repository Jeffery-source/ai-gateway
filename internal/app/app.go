package app

import (
	"fmt"

	"ai_gateway/internal/config"
	"ai_gateway/internal/conversation"
	"ai_gateway/internal/modelrouter"
	"ai_gateway/internal/provider/factory"
	"ai_gateway/internal/service"
)

type App struct {
	Config              *config.Config
	ModelRouter         *modelrouter.Router
	ChatService         *service.ChatService
	ConversationService *service.ConversationService
}

func New(cfg *config.Config) (*App, error) {

	modelRouter := modelrouter.New()

	for name, providerConfig := range cfg.Providers {

		p, err := factory.Create(
			name,
			providerConfig,
		)

		if err != nil {
			return nil, fmt.Errorf(
				"create provider '%s': %w",
				name,
				err,
			)
		}

		modelRouter.Register(
			name,
			p,
		)
	}

	for modelName, modelConfig := range cfg.Models {

		modelRouter.RegisterModel(
			modelName,
			modelrouter.ModelRoute{
				Provider: modelConfig.Provider,
				Model:    modelConfig.Model,
			},
		)
	}

	conversationStore := conversation.NewMemoryStore()

	conversationService := service.NewConversationService(
		conversationStore,
	)

	chatService := service.NewChatService(
		modelRouter,
		conversationService,
	)

	return &App{
		Config:              cfg,
		ModelRouter:         modelRouter,
		ChatService:         chatService,
		ConversationService: conversationService,
	}, nil
}
