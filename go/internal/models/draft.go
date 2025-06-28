package models

import (
	"github.com/google/uuid"
	"time"
)

// DraftType defines the type of draft.
type DraftType string

const (
	DraftTypeSnake   DraftType = "SNAKE"
	DraftTypeAuction DraftType = "AUCTION"
	DraftTypeRookie  DraftType = "ROOKIE"
)

// DraftStatus defines the status of a draft.
type DraftStatus string

const (
	DraftStatusNotStarted DraftStatus = "NOT_STARTED"
	DraftStatusInProgress DraftStatus = "IN_PROGRESS"
	DraftStatusPaused     DraftStatus = "PAUSED"
	DraftStatusCompleted  DraftStatus = "COMPLETED"
	DraftStatusCancelled  DraftStatus = "CANCELLED"
)

// DraftSettings holds JSONB configuration for drafts.
type DraftSettings struct {
	Rounds               int         `json:"rounds"`
	TimePerPickSec       int         `json:"time_per_pick_sec"`
	DraftOrder           []uuid.UUID `json:"draft_order,omitempty"`
	ThirdRoundReversal   bool        `json:"third_round_reversal,omitempty"`
	BudgetPerTeam        *float64    `json:"budget_per_team,omitempty"`         // auction
	MinBidIncrement      *float64    `json:"min_bid_increment,omitempty"`       // auction
	TimePerNominationSec *int        `json:"time_per_nomination_sec,omitempty"` // auction
	// Extend with more settings as needed
}

// Draft represents a draft instance.
type Draft struct {
	ID          uuid.UUID     `json:"id"`
	LeagueID    uuid.UUID     `json:"league_id"`
	DraftType   DraftType     `json:"draft_type"`
	Status      DraftStatus   `json:"status"`
	Settings    DraftSettings `json:"settings"`
	ScheduledAt *time.Time    `json:"scheduled_at,omitempty"`
	StartedAt   *time.Time    `json:"started_at,omitempty"`
	CompletedAt *time.Time    `json:"completed_at,omitempty"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}
