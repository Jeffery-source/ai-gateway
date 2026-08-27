package service

import (
	"fmt"

	"ai_gateway/internal/conversation"
)

type ConversationService struct {
	store conversation.Store
}

func NewConversationService(
	store conversation.Store,
) *ConversationService {
	return &ConversationService{
		store: store,
	}
}

func (s *ConversationService) Create(
	model string,
) (*conversation.Conversation, error) {

	if model == "" {
		return nil, fmt.Errorf("model is required")
	}

	return s.store.Create(model)
}

func (s *ConversationService) Get(
	id string,
) (*conversation.Conversation, error) {

	if id == "" {
		return nil, fmt.Errorf("conversation id is required")
	}

	return s.store.Get(id)
}

func (s *ConversationService) AddMessage(
	id string,
	message conversation.Message,
) error {

	if id == "" {
		return fmt.Errorf("conversation id is required")
	}

	if message.Role == "" {
		return fmt.Errorf("message role is required")
	}

	return s.store.AddMessage(
		id,
		message,
	)
}
