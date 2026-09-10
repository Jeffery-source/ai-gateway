package provider

import (
	"context"

	"ai_gateway/internal/chat"
)

func stringPtr(s string) *string {
	return &s
}

func stringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

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
