package service

import (
	"testing"

	"ai_gateway/internal/conversation"
)

func TestConversationServiceCreate(t *testing.T) {
	store := conversation.NewMemoryStore()

	service := NewConversationService(store)

	result, err := service.Create("qwen3")

	if err != nil {
		t.Fatalf(
			"create conversation failed: %v",
			err,
		)
	}

	if result.ID == "" {
		t.Fatal("conversation ID is empty")
	}

	if result.Model != "qwen3" {
		t.Fatalf(
			"expected model qwen3, got %s",
			result.Model,
		)
	}
}

func TestConversationServiceCreateWithoutModel(t *testing.T) {
	store := conversation.NewMemoryStore()

	service := NewConversationService(store)

	_, err := service.Create("")

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
