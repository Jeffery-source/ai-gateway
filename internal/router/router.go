package router

import (
	"net/http"

	"ai_gateway/internal/handler"
	"ai_gateway/internal/middleware"

	//"ai_gateway/internal/provider/openai"
	"ai_gateway/internal/service"
)

func New(
	chatService *service.ChatService,
	conversationService *service.ConversationService,
	authMiddleware *middleware.APIKeyAuth,
) http.Handler {

	mux := http.NewServeMux()

	chatHandler := handler.NewChatHandler(
		chatService,
	)
	conversationHandler := handler.NewConversationHandler(
		conversationService,
	)

	mux.HandleFunc(
		"/health",
		handler.Health,
	)

	mux.Handle(
		"/v1/models",
		authMiddleware.Middleware(
			http.HandlerFunc(handler.Models),
		),
	)

	mux.Handle(
		"/v1/chat/completions",
		authMiddleware.Middleware(
			http.HandlerFunc(chatHandler.ChatCompletions),
		),
	)
	mux.Handle(
		"/v1/conversations",
		authMiddleware.Middleware(
			http.HandlerFunc(conversationHandler.Create),
		),
	)

	return mux
}
