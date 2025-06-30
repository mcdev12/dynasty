package gateway

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
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

// Event Payloads

// PickMadePayload contains data for when a pick is completed
type PickMadePayload struct {
	PickID      string    `json:"pick_id"`
	PlayerID    string    `json:"player_id"`
	TeamID      string    `json:"team_id"`
	Round       int       `json:"round"`
	Pick        int       `json:"pick"`
	OverallPick int       `json:"overall_pick"`
	PlayerName  string    `json:"player_name"`
	TeamName    string    `json:"team_name"`
	PickedAt    time.Time `json:"picked_at"`
}

// PickStartedPayload contains data for when a pick timer begins
type PickStartedPayload struct {
	PickID         string    `json:"pick_id"`
	TeamID         string    `json:"team_id"`
	Round          int       `json:"round"`
	Pick           int       `json:"pick"`
	OverallPick    int       `json:"overall_pick"`
	StartedAt      time.Time `json:"started_at"`
	TimeoutAt      time.Time `json:"timeout_at"`
	TimePerPickSec int       `json:"time_per_pick_sec"`
}

// DraftStartedPayload contains data for when a draft begins
type DraftStartedPayload struct {
	DraftID     string    `json:"draft_id"`
	DraftType   string    `json:"draft_type"`
	StartedAt   time.Time `json:"started_at"`
	TotalRounds int       `json:"total_rounds"`
	TotalPicks  int       `json:"total_picks"`
}

// DraftPausedPayload contains data for when a draft is paused
type DraftPausedPayload struct {
	DraftID  string    `json:"draft_id"`
	PausedAt time.Time `json:"paused_at"`
	Reason   string    `json:"reason,omitempty"`
}

// DraftResumedPayload contains data for when a draft is resumed
type DraftResumedPayload struct {
	DraftID   string    `json:"draft_id"`
	ResumedAt time.Time `json:"resumed_at"`
}

// DraftCompletedPayload contains data for when a draft ends
type DraftCompletedPayload struct {
	DraftID     string    `json:"draft_id"`
	CompletedAt time.Time `json:"completed_at"`
	Duration    string    `json:"duration"`
	TotalPicks  int       `json:"total_picks"`
}

// TimerTickPayload contains periodic timer updates (optional)
type TimerTickPayload struct {
	PickID           string    `json:"pick_id"`
	TeamID           string    `json:"team_id"`
	TimeRemainingSec int       `json:"time_remaining_sec"`
	TickedAt         time.Time `json:"ticked_at"`
}

// Helper functions to create events

// NewPickMadeEvent creates a new PickMade event
func NewPickMadeEvent(draftID uuid.UUID, payload PickMadePayload) (*DraftEvent, error) {
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
func NewPickStartedEvent(draftID uuid.UUID, payload PickStartedPayload) (*DraftEvent, error) {
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
func NewDraftStartedEvent(draftID uuid.UUID, payload DraftStartedPayload) (*DraftEvent, error) {
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
func NewDraftCompletedEvent(draftID uuid.UUID, payload DraftCompletedPayload) (*DraftEvent, error) {
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
		var payload PickMadePayload
		if err := json.Unmarshal(event.Data, &payload); err != nil {
			return nil, err
		}
		return payload, nil

	case EventTypePickStarted:
		var payload PickStartedPayload
		if err := json.Unmarshal(event.Data, &payload); err != nil {
			return nil, err
		}
		return payload, nil

	case EventTypeDraftStarted:
		var payload DraftStartedPayload
		if err := json.Unmarshal(event.Data, &payload); err != nil {
			return nil, err
		}
		return payload, nil

	case EventTypeDraftPaused:
		var payload DraftPausedPayload
		if err := json.Unmarshal(event.Data, &payload); err != nil {
			return nil, err
		}
		return payload, nil

	case EventTypeDraftResumed:
		var payload DraftResumedPayload
		if err := json.Unmarshal(event.Data, &payload); err != nil {
			return nil, err
		}
		return payload, nil

	case EventTypeDraftCompleted:
		var payload DraftCompletedPayload
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