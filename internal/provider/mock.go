package provider

import (
	"context"
	"time"

	"ai_gateway/internal/chat"
	"ai_gateway/internal/utils"
)

type MockProvider struct{}

func NewMockProvider() *MockProvider {
	return &MockProvider{}
}

func (p *MockProvider) Chat(
	ctx context.Context,
	req *chat.ChatCompletionRequest,
) (*chat.ChatCompletionResponse, error) {

	return &chat.ChatCompletionResponse{
		ID:      "chatcmpl-mock",
		Object:  "chat.completion",
		Created: 0,
		Model:   req.Model,
		Choices: []chat.Choice{
			{
				Index: 0,
				Message: chat.Message{
					Role:    "assistant",
					Content: utils.StringPtr("Hello from Mock Provider."),
				},
				FinishReason: "stop",
			},
		},
	}, nil
}

func (p *MockProvider) ChatStream(
	ctx context.Context,
	req *chat.ChatCompletionRequest,
	writer StreamWriter,
) error {

	chunks := []string{
		"Hello ",
		"from ",
		"Mock ",
		"Provider.",
	}

	for _, content := range chunks {

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(500 * time.Millisecond):
			// default:
		}

		chunk := &chat.ChatCompletionChunk{
			ID:      "chatcmpl-mock",
			Object:  "chat.completion.chunk",
			Created: 0,
			Model:   req.Model,
			Choices: []chat.ChunkChoice{
				{
					Index: 0,
					Delta: chat.Delta{
						Content: content,
					},
				},
			},
		}

		if err := writer(chunk); err != nil {
			return err
		}
	}

	finishReason := "stop"

	return writer(
		&chat.ChatCompletionChunk{
			ID:      "chatcmpl-mock",
			Object:  "chat.completion.chunk",
			Created: 0,
			Model:   req.Model,
			Choices: []chat.ChunkChoice{
				{
					Index:        0,
					Delta:        chat.Delta{},
					FinishReason: &finishReason,
				},
			},
		},
	)
}
