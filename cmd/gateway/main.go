package main

import (
	"log"
	"os"

	"ai_gateway/internal/app"
	"ai_gateway/internal/config"
	"ai_gateway/internal/middleware"
	"ai_gateway/internal/router"
	"ai_gateway/internal/server"
)

func main() {

	apiKey := os.Getenv("AI_GATEWAY_API_KEY")

	if apiKey == "" {
		log.Fatal("AI_GATEWAY_API_KEY is required")
	}

	authMiddleware := middleware.NewAPIKeyAuth(
		apiKey,
	)

	configPath := "config.yaml"

	if value := os.Getenv("AI_GATEWAY_CONFIG"); value != "" {
		configPath = value
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf(
			"load config: %v",
			err,
		)
	}

	application, err := app.New(cfg)
	if err != nil {
		log.Fatalf(
			"create application: %v",
			err,
		)
	}

	handler := router.New(
		application.ChatService,
		application.ConversationService,
		authMiddleware,
	)

	srv := server.New(
		cfg.Server.Address(),
		handler,
	)

	addr := cfg.Server.Address()

	log.Printf(
		"AI Gateway listening on %s",
		addr,
	)

	if err := srv.Start(); err != nil {
		log.Fatal(err)
	}
}
