package gateway

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/mcdev12/dynasty/go/internal/draft/events"
)

// DraftEvent represents the base structure for all draft events
type DraftEvent struct {
	ID        string          `json:"id"`         // Event UUID
	DraftID   string          `json:"draft_id"`   // Draft UUID  
	Type      EventType       `json:"type"`       // Event type
	Timestamp time.Time       `json:"timestamp"`  // Event creation time
	Data      json.RawMessage `json:"data"`       // Event-specific payload
}

// EventType represents the type of draft event
type EventType string

const (
	EventTypePickMade      EventType = "PickMade"
	EventTypePickStarted   EventType = "PickStarted"
	EventTypeDraftStarted  EventType = "DraftStarted"
	EventTypeDraftPaused   EventType = "DraftPaused"
	EventTypeDraftResumed  EventType = "DraftResumed"
	EventTypeDraftCompleted EventType = "DraftCompleted"
	EventTypeTimerTick     EventType = "TimerTick"
)

// Event Payloads are now in the events package to avoid cyclic imports

// TimerTickPayload contains periodic timer updates (optional)
type TimerTickPayload struct {
	PickID           string    `json:"pick_id"`
	TeamID           string    `json:"team_id"`
	TimeRemainingSec int       `json:"time_remaining_sec"`
	TickedAt         time.Time `json:"ticked_at"`
}

// Helper functions to create events

// NewPickMadeEvent creates a new PickMade event
func NewPickMadeEvent(draftID uuid.UUID, payload events.PickMadePayload) (*DraftEvent, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	return &DraftEvent{
		ID:        uuid.New().String(),
		DraftID:   draftID.String(),
		Type:      EventTypePickMade,
		Timestamp: time.Now(),
		Data:      data,
	}, nil
}

// NewPickStartedEvent creates a new PickStarted event
func NewPickStartedEvent(draftID uuid.UUID, payload events.PickStartedPayload) (*DraftEvent, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	return &DraftEvent{
		ID:        uuid.New().String(),
		DraftID:   draftID.String(),
		Type:      EventTypePickStarted,
		Timestamp: time.Now(),
		Data:      data,
	}, nil
}

// NewDraftStartedEvent creates a new DraftStarted event
func NewDraftStartedEvent(draftID uuid.UUID, payload events.DraftStartedPayload) (*DraftEvent, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	return &DraftEvent{
		ID:        uuid.New().String(),
		DraftID:   draftID.String(),
		Type:      EventTypeDraftStarted,
		Timestamp: time.Now(),
		Data:      data,
	}, nil
}

// NewDraftCompletedEvent creates a new DraftCompleted event
func NewDraftCompletedEvent(draftID uuid.UUID, payload events.DraftCompletedPayload) (*DraftEvent, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	return &DraftEvent{
		ID:        uuid.New().String(),
		DraftID:   draftID.String(),
		Type:      EventTypeDraftCompleted,
		Timestamp: time.Now(),
		Data:      data,
	}, nil
}

// ParseEventPayload parses event data into the appropriate payload struct
func ParseEventPayload(event *DraftEvent) (interface{}, error) {
	switch event.Type {
	case EventTypePickMade:
		var payload events.PickMadePayload
		if err := json.Unmarshal(event.Data, &payload); err != nil {
			return nil, err
		}
		return payload, nil

	case EventTypePickStarted:
		var payload events.PickStartedPayload
		if err := json.Unmarshal(event.Data, &payload); err != nil {
			return nil, err
		}
		return payload, nil

	case EventTypeDraftStarted:
		var payload events.DraftStartedPayload
		if err := json.Unmarshal(event.Data, &payload); err != nil {
			return nil, err
		}
		return payload, nil

	case EventTypeDraftPaused:
		var payload events.DraftPausedPayload
		if err := json.Unmarshal(event.Data, &payload); err != nil {
			return nil, err
		}
		return payload, nil

	case EventTypeDraftResumed:
		var payload events.DraftResumedPayload
		if err := json.Unmarshal(event.Data, &payload); err != nil {
			return nil, err
		}
		return payload, nil

	case EventTypeDraftCompleted:
		var payload events.DraftCompletedPayload
		if err := json.Unmarshal(event.Data, &payload); err != nil {
			return nil, err
		}
		return payload, nil

	case EventTypeTimerTick:
		var payload TimerTickPayload
		if err := json.Unmarshal(event.Data, &payload); err != nil {
			return nil, err
		}
		return payload, nil

	default:
		return nil, nil // Unknown event type
	}
}