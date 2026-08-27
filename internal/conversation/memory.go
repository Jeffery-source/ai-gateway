package conversation

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

func newID() string {

	b := make([]byte, 16)

	_, _ = rand.Read(b)

	return "conv_" + hex.EncodeToString(b)
}

type MemoryStore struct {
	mu            sync.RWMutex
	conversations map[string]*Conversation
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		conversations: make(
			map[string]*Conversation,
		),
	}
}

func (s *MemoryStore) Create(
	model string,
) (*Conversation, error) {

	now := time.Now()

	conversation := &Conversation{
		ID:        newID(),
		Model:     model,
		Messages:  make([]Message, 0),
		CreatedAt: now,
		UpdatedAt: now,
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.conversations[conversation.ID] = conversation

	return conversation, nil
}

func (s *MemoryStore) Get(
	id string,
) (*Conversation, error) {

	s.mu.RLock()
	defer s.mu.RUnlock()

	conversation, ok := s.conversations[id]

	if !ok {
		return nil, fmt.Errorf(
			"conversation not found: %s",
			id,
		)
	}

	return conversation, nil
}

func (s *MemoryStore) AddMessage(
	id string,
	message Message,
) error {

	s.mu.Lock()
	defer s.mu.Unlock()

	conversation, ok := s.conversations[id]

	if !ok {
		return fmt.Errorf(
			"conversation not found: %s",
			id,
		)
	}

	conversation.Messages = append(
		conversation.Messages,
		message,
	)

	conversation.UpdatedAt = time.Now()

	return nil
}
