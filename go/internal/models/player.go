package models

import (
	"time"

	"github.com/google/uuid"
)

// Profile is a marker interface for sport-specific player profiles
type Profile interface {
	SportID() string
}

// Player represents a sports player in the system
type Player struct {
	ID         uuid.UUID  `json:"id"`
	SportID    string     `json:"sport_id"`
	ExternalID string     `json:"external_id"`
	FullName   string     `json:"full_name"`
	TeamID     *uuid.UUID `json:"team_id,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`

	NFLPlayerProfile *NFLPlayerProfile `json:"nfl_player_profile,omitempty"`
}

// NFLPlayerProfile represents NFL-specific player attributes
type NFLPlayerProfile struct {
	PlayerID     uuid.UUID  `json:"player_id"`
	Position     string     `json:"position"`
	Status       string     `json:"status"`
	College      string     `json:"college"`
	JerseyNumber int        `json:"jersey_number"`
	Experience   int        `json:"experience"`
	BirthDate    *time.Time `json:"birth_date,omitempty"` // Keep as pointer for null dates
	HeightCm     int        `json:"height_cm"`
	WeightKg     int        `json:"weight_kg"`
	HeightDesc   string     `json:"height_desc"`
	WeightDesc   string     `json:"weight_desc"`
}

// SportID returns the sport identifier for NFLPlayerProfile
func (p *NFLPlayerProfile) SportID() string {
	return "nfl"
}
