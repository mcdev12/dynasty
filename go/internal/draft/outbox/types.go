package outbox

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// OutboxEvent represents an outbox event for the application layer
type OutboxEvent struct {
	ID        uuid.UUID       `json:"id"`
	DraftID   uuid.UUID       `json:"draft_id"`
	EventType string          `json:"event_type"`
	Payload   json.RawMessage `json:"payload"`
	CreatedAt time.Time       `json:"created_at"`
	SentAt    *time.Time      `json:"sent_at,omitempty"`
}