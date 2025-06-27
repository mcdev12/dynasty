package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Roster struct {
	ID              uuid.UUID       `json:"id"`
	FantasyTeamID   uuid.UUID       `json:"fantasy_team_id"`
	PlayerID        uuid.UUID       `json:"player_id"`
	Position        RosterPosition  `json:"position"`
	AcquiredAt      time.Time       `json:"acquired_at"`
	AcquisitionType AcquisitionType `json:"acquisition_type"`
	KeeperData      json.RawMessage `json:"keeper_data"`
}

// RosterPosition represents the position a player has on a roster
type RosterPosition string

const (
	RosterPositionStarter RosterPosition = "STARTER"
	RosterPositionBench   RosterPosition = "BENCH"
	RosterPositionIR      RosterPosition = "IR"
	RosterPositionTaxi    RosterPosition = "TAXI"
)

// AcquisitionType represents how a player was acquired
type AcquisitionType string

const (
	AcquisitionTypeDraft     AcquisitionType = "DRAFT"
	AcquisitionTypeWaiver    AcquisitionType = "WAIVER"
	AcquisitionTypeTrade     AcquisitionType = "TRADE"
	AcquisitionTypeFreeAgent AcquisitionType = "FREE_AGENT"
	AcquisitionTypeKeeper    AcquisitionType = "KEEPER"
)
