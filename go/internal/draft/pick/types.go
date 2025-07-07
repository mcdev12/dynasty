package pick

import (
	"github.com/google/uuid"
)

// CreateDraftPickRequest represents a request to create a new draft pick
type CreateDraftPickRequest struct {
	ID            uuid.UUID  `json:"id"`
	DraftID       uuid.UUID  `json:"draft_id"`
	Round         int        `json:"round"`
	Pick          int        `json:"pick"`
	OverallPick   int        `json:"overall_pick"`
	TeamID        uuid.UUID  `json:"team_id"`
	PlayerID      *uuid.UUID `json:"player_id,omitempty"`
	AuctionAmount *float64   `json:"auction_amount,omitempty"`
	KeeperPick    bool       `json:"keeper_pick"`
}

// UpdateDraftPickPlayerRequest represents a request to update a draft pick's player
type UpdateDraftPickPlayerRequest struct {
	PlayerID      uuid.UUID `json:"player_id"`
	AuctionAmount *float64  `json:"auction_amount,omitempty"`
	KeeperPick    bool      `json:"keeper_pick"`
}

// MakePickRequest represents a request to make a draft pick
type MakePickRequest struct {
	PickID      uuid.UUID `json:"pick_id"`
	PlayerID    uuid.UUID `json:"player_id"`
	DraftID     uuid.UUID `json:"draft_id"`
	TeamID      uuid.UUID `json:"team_id"`
	OverallPick int       `json:"overall_pick"`
}

// Slot represents a claimed pick slot for auto-pick
type Slot struct {
	PickID      uuid.UUID `json:"pick_id"`
	TeamID      uuid.UUID `json:"team_id"`
	OverallPick int       `json:"overall_pick"`
}

// AvailablePlayer represents a player available for draft
type AvailablePlayer struct {
	ID       uuid.UUID `json:"id"`
	FullName string    `json:"full_name"`
	TeamID   uuid.UUID `json:"team_id"`
}