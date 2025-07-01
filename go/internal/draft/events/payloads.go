package events

import (
	"time"
)

// Event payload types that are shared between draft and gateway packages

// PickStartedPayload is the payload for a PickStarted event
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

// PickMadePayload is the payload for a PickMade event
type PickMadePayload struct {
	PickID      string    `json:"pick_id"`
	TeamID      string    `json:"team_id"`
	TeamName    string    `json:"team_name"`
	PlayerID    string    `json:"player_id"`
	PlayerName  string    `json:"player_name"`
	Round       int       `json:"round"`
	Pick        int       `json:"pick"`
	OverallPick int       `json:"overall_pick"`
	MadeAt      time.Time `json:"made_at"`
}

// DraftStartedPayload is the payload for a DraftStarted event
type DraftStartedPayload struct {
	DraftID     string    `json:"draft_id"`
	DraftType   string    `json:"draft_type"`
	StartedAt   time.Time `json:"started_at"`
	TotalRounds int       `json:"total_rounds"`
	TotalPicks  int       `json:"total_picks"`
}

// DraftCompletedPayload is the payload for a DraftCompleted event
type DraftCompletedPayload struct {
	DraftID     string    `json:"draft_id"`
	CompletedAt time.Time `json:"completed_at"`
	Duration    string    `json:"duration"`
	TotalPicks  int       `json:"total_picks"`
}

// DraftPausedPayload is the payload for a DraftPaused event
type DraftPausedPayload struct {
	DraftID  string    `json:"draft_id"`
	PausedAt time.Time `json:"paused_at"`
	Reason   string    `json:"reason"`
}

// DraftResumedPayload is the payload for a DraftResumed event
type DraftResumedPayload struct {
	DraftID   string    `json:"draft_id"`
	ResumedAt time.Time `json:"resumed_at"`
}