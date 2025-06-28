package models

import (
	"github.com/google/uuid"
	"time"
)

// DraftPick represents a single pick in a draft.
type DraftPick struct {
	ID            uuid.UUID  `json:"id"`
	DraftID       uuid.UUID  `json:"draft_id"`
	Round         int        `json:"round"`
	Pick          int        `json:"pick"`         // pick number in the round
	OverallPick   int        `json:"overall_pick"` // pick number overall
	TeamID        uuid.UUID  `json:"team_id"`
	PlayerID      *uuid.UUID `json:"player_id,omitempty"` // nil until picked
	PickedAt      *time.Time `json:"picked_at,omitempty"`
	AuctionAmount *float64   `json:"auction_amount,omitempty"` // auction support
	KeeperPick    bool       `json:"keeper_pick"`              // indicates if used on keeper
}
