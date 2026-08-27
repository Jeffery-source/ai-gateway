package provider

import (
	"context"

	"ai_gateway/internal/chat"
)

type ModelProvider interface {
	Chat(
		ctx context.Context,
		req *chat.ChatCompletionRequest,
	) (*chat.ChatCompletionResponse, error)

	ChatStream(
		ctx context.Context,
		req *chat.ChatCompletionRequest,
		writer StreamWriter,
	) error
}

type StreamWriter func(chunk *chat.ChatCompletionChunk) error
