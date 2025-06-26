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
	PlayerID     uuid.UUID `json:"player_id"`
	HeightCm     *int      `json:"height_cm,omitempty"` // Optional - may not have metric data
	WeightKg     *int      `json:"weight_kg,omitempty"` // Optional - may not have metric data
	GroupRole    string    `json:"group_role"`          // 'Offense', 'Defense', 'Special Teams'
	Position     string    `json:"position"`            // 'QB', 'RB', 'WR', etc.
	Age          *int      `json:"age,omitempty"`       // Optional - may not be available
	HeightDesc   string    `json:"height_desc"`         // '6'' 3"'
	WeightDesc   string    `json:"weight_desc"`         // '210 lbs'
	College      *string   `json:"college,omitempty"`   // Optional - some players may not have college
	JerseyNumber int       `json:"jersey_number"`
	SalaryDesc   *string   `json:"salary_desc,omitempty"` // Optional - salary may not be public
	Experience   int       `json:"experience"`            // Years of experience
}

// SportID returns the sport identifier for NFLPlayerProfile
func (p *NFLPlayerProfile) SportID() string {
	return "nfl"
}
