package conversation

import (
	"time"
)

type Conversation struct {
	ID        string
	Model     string
	Messages  []Message
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Message struct {
	Role    string
	Content string
}

type Store interface {
	Create(
		model string,
	) (*Conversation, error)

	Get(
		id string,
	) (*Conversation, error)

	AddMessage(
		id string,
		message Message,
	) error
}
