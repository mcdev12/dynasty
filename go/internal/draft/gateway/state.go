package gateway

import (
	"time"

	"github.com/google/uuid"
)

// DraftState represents the current state of a draft for synchronization
type DraftState struct {
	DraftID        string     `json:"draft_id"`
	Status         string     `json:"status"`
	CurrentPick    *PickState `json:"current_pick,omitempty"`
	TotalPicks     int        `json:"total_picks"`
	CompletedPicks int        `json:"completed_picks"`
	StartedAt      *time.Time `json:"started_at,omitempty"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
	PausedAt       *time.Time `json:"paused_at,omitempty"`
}

// PickState represents the current pick being made
type PickState struct {
	PickID           string    `json:"pick_id"`
	TeamID           string    `json:"team_id"`
	Round            int       `json:"round"`
	Pick             int       `json:"pick"`
	OverallPick      int       `json:"overall_pick"`
	StartedAt        time.Time `json:"started_at"`
	TimeoutAt        time.Time `json:"timeout_at"`
	TimeRemainingSec int       `json:"time_remaining_sec"`
}

// CalculateTimeRemaining calculates the remaining time for a pick
func (p *PickState) CalculateTimeRemaining() int {
	if p.TimeoutAt.IsZero() {
		return 0
	}

	remaining := int(time.Until(p.TimeoutAt).Seconds())
	if remaining < 0 {
		return 0
	}
	return remaining
}

// CalculateTimeRemainingWithClockSync calculates remaining time with server clock sync
func (p *PickState) CalculateTimeRemainingWithClockSync(serverTime time.Time) int {
	if p.TimeoutAt.IsZero() {
		return 0
	}

	remaining := int(p.TimeoutAt.Sub(serverTime).Seconds())
	if remaining < 0 {
		return 0
	}
	return remaining
}

// UpdateTimeRemaining updates the time remaining field
func (p *PickState) UpdateTimeRemaining() {
	p.TimeRemainingSec = p.CalculateTimeRemaining()
}

// Pick represents a completed pick for history
type Pick struct {
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

// PickHistoryResponse represents the response for pick history API
type PickHistoryResponse struct {
	Picks      []Pick  `json:"picks"`
	NextCursor *string `json:"next_cursor,omitempty"`
	HasMore    bool    `json:"has_more"`
}

// StateUpdateRequest represents a request to update draft state
type StateUpdateRequest struct {
	Status      *string    `json:"status,omitempty"`
	CurrentPick *PickState `json:"current_pick,omitempty"`
}

// DraftStateManager manages the current state of drafts in memory
type DraftStateManager struct {
	states map[uuid.UUID]*DraftState
}

// NewDraftStateManager creates a new state manager
func NewDraftStateManager() *DraftStateManager {
	return &DraftStateManager{
		states: make(map[uuid.UUID]*DraftState),
	}
}

// GetState returns the current state for a draft
func (dsm *DraftStateManager) GetState(draftID uuid.UUID) *DraftState {
	return dsm.states[draftID]
}

// UpdateState updates the state for a draft
func (dsm *DraftStateManager) UpdateState(draftID uuid.UUID, state *DraftState) {
	dsm.states[draftID] = state
}

// RemoveState removes the state for a draft (e.g., when completed)
func (dsm *DraftStateManager) RemoveState(draftID uuid.UUID) {
	delete(dsm.states, draftID)
}

// ProcessEvent updates the draft state based on an incoming event
func (dsm *DraftStateManager) ProcessEvent(event *DraftEvent) error {
	draftID, err := uuid.Parse(event.DraftID)
	if err != nil {
		return err
	}

	state := dsm.GetState(draftID)
	if state == nil {
		// Initialize state if it doesn't exist
		state = &DraftState{
			DraftID: event.DraftID,
		}
	}

	switch event.Type {
	case EventTypeDraftStarted:
		payload, err := ParseEventPayload(event)
		if err != nil {
			return err
		}
		p := payload.(DraftStartedPayload)
		state.Status = "IN_PROGRESS"
		state.TotalPicks = p.TotalPicks
		state.StartedAt = &p.StartedAt

	case EventTypePickStarted:
		payload, err := ParseEventPayload(event)
		if err != nil {
			return err
		}
		p := payload.(PickStartedPayload)
		state.CurrentPick = &PickState{
			PickID:      p.PickID,
			TeamID:      p.TeamID,
			Round:       p.Round,
			Pick:        p.Pick,
			OverallPick: p.OverallPick,
			StartedAt:   p.StartedAt,
			TimeoutAt:   p.TimeoutAt,
		}
		state.CurrentPick.UpdateTimeRemaining()

	case EventTypePickMade:
		state.CompletedPicks++
		state.CurrentPick = nil // Clear current pick

	case EventTypeDraftPaused:
		payload, err := ParseEventPayload(event)
		if err != nil {
			return err
		}
		p := payload.(DraftPausedPayload)
		state.Status = "PAUSED"
		state.PausedAt = &p.PausedAt

	case EventTypeDraftResumed:
		state.Status = "IN_PROGRESS"
		state.PausedAt = nil

	case EventTypeDraftCompleted:
		payload, err := ParseEventPayload(event)
		if err != nil {
			return err
		}
		p := payload.(DraftCompletedPayload)
		state.Status = "COMPLETED"
		state.CompletedAt = &p.CompletedAt
		state.CurrentPick = nil
	}

	dsm.UpdateState(draftID, state)
	return nil
}
