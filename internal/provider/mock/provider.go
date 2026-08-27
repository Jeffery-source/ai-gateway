package mock

import (
	"context"
	"time"

	"ai_gateway/internal/chat"
	"ai_gateway/internal/provider"
)

type Provider struct{}

func NewProvider() *Provider {
	return &Provider{}
}

func (p *Provider) Chat(
	ctx context.Context,
	req *chat.ChatCompletionRequest,
) (*chat.ChatCompletionResponse, error) {

	return &chat.ChatCompletionResponse{
		ID:     "mock-response",
		Object: "chat.completion",
		Model:  req.Model,
		Choices: []chat.Choice{
			{
				Index: 0,
				Message: chat.Message{
					Role:    "assistant",
					Content: "This is a mock response.",
				},
				FinishReason: "stop",
			},
		},
	}, nil
}

func (p *Provider) ChatStream(
	ctx context.Context,
	req *chat.ChatCompletionRequest,
	writer provider.StreamWriter,
) error {

	chunks := []string{
		"This ",
		"is ",
		"a ",
		"mock ",
		"streaming ",
		"response.",
	}

	for _, content := range chunks {

		select {
		case <-ctx.Done():
			return ctx.Err()

		default:
		}

		chunk := &chat.ChatCompletionChunk{
			ID:     "mock-stream",
			Object: "chat.completion.chunk",
			Model:  req.Model,
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

		time.Sleep(50 * time.Millisecond)
	}

	return nil
}
