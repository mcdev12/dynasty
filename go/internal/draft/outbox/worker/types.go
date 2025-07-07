package worker

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type OutboxEvent struct {
	ID        uuid.UUID
	DraftID   uuid.UUID
	EventType string
	Payload   []byte
	CreatedAt time.Time
	SentAt    *time.Time
}

type EventPublisher interface {
	Publish(ctx context.Context, event OutboxEvent) error
}
