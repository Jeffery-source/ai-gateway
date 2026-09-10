package service

import (
	"context"
	"testing"

	"ai_gateway/internal/chat"
	"ai_gateway/internal/conversation"
	"ai_gateway/internal/modelrouter"
	"ai_gateway/internal/provider"
)

func stringPtr(s string) *string {
	return &s
}

type recordingProvider struct {
	requests []*chat.ChatCompletionRequest
}

var _ provider.ModelProvider = (*recordingProvider)(nil)

func (p *recordingProvider) Chat(
	ctx context.Context,
	req *chat.ChatCompletionRequest,
) (*chat.ChatCompletionResponse, error) {

	copyReq := *req

	copyReq.Messages = append(
		[]chat.Message(nil),
		req.Messages...,
	)

	p.requests = append(
		p.requests,
		&copyReq,
	)

	return &chat.ChatCompletionResponse{
		ID:     "test-response",
		Object: "chat.completion",
		Model:  req.Model,
		Choices: []chat.Choice{
			{
				Index: 0,
				Message: chat.Message{
					Role:    "assistant",
					Content: stringPtr("你好，张三！"),
				},
				FinishReason: "stop",
			},
		},
	}, nil
}

func (p *recordingProvider) ChatStream(
	ctx context.Context,
	req *chat.ChatCompletionRequest,
	writer provider.StreamWriter,
) error {

	chunks := []string{
		"你好，",
		"张三！",
		"很高兴",
		"认识你。",
	}

	for _, content := range chunks {

		select {
		case <-ctx.Done():
			return ctx.Err()

		default:
		}

		chunk := &chat.ChatCompletionChunk{
			ID:     "test-stream",
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
	}

	return nil
}
func TestChatServiceConversation(t *testing.T) {

	// 1. 创建 Provider
	mockProvider := &recordingProvider{}

	// 2. 创建 Model Router
	modelRouter := modelrouter.New()

	modelRouter.Register(
		"mock",
		mockProvider,
	)

	modelRouter.RegisterModel(
		"qwen3",
		modelrouter.ModelRoute{
			Provider: "mock",
			Model:    "qwen3",
		},
	)

	// 3. 创建 Conversation
	conversationStore := conversation.NewMemoryStore()

	conversationService := NewConversationService(
		conversationStore,
	)

	// 4. 创建 ChatService
	chatService := NewChatService(
		modelRouter,
		conversationService,
	)

	// 5. 创建 Conversation
	conv, err := conversationService.Create(
		"qwen3",
	)

	if err != nil {
		t.Fatalf(
			"create conversation failed: %v",
			err,
		)
	}

	// 6. 第一次请求
	firstReq := &chat.ChatCompletionRequest{
		Model:          "qwen3",
		ConversationID: conv.ID,
		Messages: []chat.Message{
			{
				Role:    "user",
				Content: stringPtr("我叫张三"),
			},
		},
	}

	_, err = chatService.Chat(
		context.Background(),
		firstReq,
	)

	if err != nil {
		t.Fatalf(
			"first chat failed: %v",
			err,
		)
	}

	// 7. 第二次请求
	secondReq := &chat.ChatCompletionRequest{
		Model:          "qwen3",
		ConversationID: conv.ID,
		Messages: []chat.Message{
			{
				Role:    "user",
				Content: stringPtr("我叫什么？"),
			},
		},
	}

	_, err = chatService.Chat(
		context.Background(),
		secondReq,
	)

	if err != nil {
		t.Fatalf(
			"second chat failed: %v",
			err,
		)
	}

	// 8. Provider 应该收到两次请求
	if len(mockProvider.requests) != 2 {
		t.Fatalf(
			"expected 2 provider requests, got %d",
			len(mockProvider.requests),
		)
	}

	// 9. 检查第二次请求
	secondProviderRequest :=
		mockProvider.requests[1]

	if len(secondProviderRequest.Messages) != 3 {
		t.Fatalf(
			"expected 3 messages in second request, got %d",
			len(secondProviderRequest.Messages),
		)
	}

	// 第一条
	if secondProviderRequest.Messages[0].Role != "user" {
		t.Fatalf(
			"expected first message role user, got %s",
			secondProviderRequest.Messages[0].Role,
		)
	}

	if secondProviderRequest.Messages[0].Content != stringPtr("我叫张三") {
		t.Fatalf(
			"expected first message content '我叫张三', got %s",
			secondProviderRequest.Messages[0].Content,
		)
	}

	// 第二条
	if secondProviderRequest.Messages[1].Role != "assistant" {
		t.Fatalf(
			"expected second message role assistant, got %s",
			secondProviderRequest.Messages[1].Role,
		)
	}

	if secondProviderRequest.Messages[1].Content != stringPtr("你好，张三！") {
		t.Fatalf(
			"expected second message content '你好，张三！', got %s",
			secondProviderRequest.Messages[1].Content,
		)
	}

	// 第三条
	if secondProviderRequest.Messages[2].Role != "user" {
		t.Fatalf(
			"expected third message role user, got %s",
			secondProviderRequest.Messages[2].Role,
		)
	}

	if secondProviderRequest.Messages[2].Content != stringPtr("我叫什么？") {
		t.Fatalf(
			"expected third message content '我叫什么？', got %s",
			secondProviderRequest.Messages[2].Content,
		)
	}
}

func TestChatServiceConversationStream(t *testing.T) {

	mockProvider := &recordingProvider{}

	modelRouter := modelrouter.New()

	modelRouter.Register(
		"mock",
		mockProvider,
	)

	modelRouter.RegisterModel(
		"qwen3",
		modelrouter.ModelRoute{
			Provider: "mock",
			Model:    "qwen3",
		},
	)

	conversationStore := conversation.NewMemoryStore()

	conversationService := NewConversationService(
		conversationStore,
	)

	chatService := NewChatService(
		modelRouter,
		conversationService,
	)

	conv, err := conversationService.Create(
		"qwen3",
	)

	if err != nil {
		t.Fatalf(
			"create conversation failed: %v",
			err,
		)
	}

	req := &chat.ChatCompletionRequest{
		Model:          "qwen3",
		ConversationID: conv.ID,
		Messages: []chat.Message{
			{
				Role:    "user",
				Content: stringPtr("你好"),
			},
		},
		Stream: true,
	}

	var received string

	err = chatService.ChatStream(
		context.Background(),
		req,
		func(chunk *chat.ChatCompletionChunk) error {

			for _, choice := range chunk.Choices {
				received += choice.Delta.Content
			}

			return nil
		},
	)

	if err != nil {
		t.Fatalf(
			"chat stream failed: %v",
			err,
		)
	}

	expected := "你好，张三！很高兴认识你。"

	if received != expected {
		t.Fatalf(
			"expected streamed content %q, got %q",
			expected,
			received,
		)
	}

	got, err := conversationService.Get(
		conv.ID,
	)

	if err != nil {
		t.Fatalf(
			"get conversation failed: %v",
			err,
		)
	}

	if len(got.Messages) != 2 {
		t.Fatalf(
			"expected 2 messages, got %d",
			len(got.Messages),
		)
	}

	if got.Messages[0].Role != "user" {
		t.Fatalf(
			"expected first message role user, got %s",
			got.Messages[0].Role,
		)
	}

	if got.Messages[0].Content != "你好" {
		t.Fatalf(
			"expected first message content %q, got %q",
			"你好",
			got.Messages[0].Content,
		)
	}

	if got.Messages[1].Role != "assistant" {
		t.Fatalf(
			"expected second message role assistant, got %s",
			got.Messages[1].Role,
		)
	}

	if got.Messages[1].Content != expected {
		t.Fatalf(
			"expected assistant content %q, got %q",
			expected,
			got.Messages[1].Content,
		)
	}
}
