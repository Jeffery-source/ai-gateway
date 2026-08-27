package conversation

import "testing"

func TestMemoryStoreCreateAndGet(t *testing.T) {
	store := NewMemoryStore()

	conversation, err := store.Create("qwen3")
	if err != nil {
		t.Fatalf("create conversation failed: %v", err)
	}

	if conversation.ID == "" {
		t.Fatal("conversation ID is empty")
	}

	if conversation.Model != "qwen3" {
		t.Fatalf(
			"expected model qwen3, got %s",
			conversation.Model,
		)
	}

	got, err := store.Get(conversation.ID)
	if err != nil {
		t.Fatalf("get conversation failed: %v", err)
	}

	if got.ID != conversation.ID {
		t.Fatalf(
			"expected ID %s, got %s",
			conversation.ID,
			got.ID,
		)
	}
}

func TestMemoryStoreAddMessage(t *testing.T) {
	store := NewMemoryStore()

	conversation, err := store.Create("qwen3")
	if err != nil {
		t.Fatalf("create conversation failed: %v", err)
	}

	err = store.AddMessage(
		conversation.ID,
		Message{
			Role:    "user",
			Content: "你好",
		},
	)

	if err != nil {
		t.Fatalf("add message failed: %v", err)
	}

	got, err := store.Get(conversation.ID)
	if err != nil {
		t.Fatalf("get conversation failed: %v", err)
	}

	if len(got.Messages) != 1 {
		t.Fatalf(
			"expected 1 message, got %d",
			len(got.Messages),
		)
	}

	if got.Messages[0].Role != "user" {
		t.Fatalf(
			"expected role user, got %s",
			got.Messages[0].Role,
		)
	}

	if got.Messages[0].Content != "你好" {
		t.Fatalf(
			"expected content 你好, got %s",
			got.Messages[0].Content,
		)
	}
}

func TestMemoryStoreGetNotFound(t *testing.T) {
	store := NewMemoryStore()

	_, err := store.Get("conv_not_found")

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestMemoryStoreMultipleMessages(t *testing.T) {
	store := NewMemoryStore()

	conversation, err := store.Create("qwen3")
	if err != nil {
		t.Fatalf("create conversation failed: %v", err)
	}

	messages := []Message{
		{
			Role:    "user",
			Content: "我叫张三",
		},
		{
			Role:    "assistant",
			Content: "你好，张三",
		},
		{
			Role:    "user",
			Content: "我叫什么？",
		},
		{
			Role:    "assistant",
			Content: "你叫张三",
		},
	}

	for _, message := range messages {
		if err := store.AddMessage(
			conversation.ID,
			message,
		); err != nil {
			t.Fatalf(
				"add message failed: %v",
				err,
			)
		}
	}

	got, err := store.Get(conversation.ID)
	if err != nil {
		t.Fatalf("get conversation failed: %v", err)
	}

	if len(got.Messages) != 4 {
		t.Fatalf(
			"expected 4 messages, got %d",
			len(got.Messages),
		)
	}

	for i, expected := range messages {
		actual := got.Messages[i]

		if actual.Role != expected.Role {
			t.Fatalf(
				"message %d role: expected %s, got %s",
				i,
				expected.Role,
				actual.Role,
			)
		}

		if actual.Content != expected.Content {
			t.Fatalf(
				"message %d content: expected %s, got %s",
				i,
				expected.Content,
				actual.Content,
			)
		}
	}
}
