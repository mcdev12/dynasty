package models

import (
	"github.com/google/uuid"
	"time"
)

// LeagueType represents the type of league
type LeagueType string

const (
	LeagueTypeRedraft LeagueType = "REDRAFT"
	LeagueTypeKeeper  LeagueType = "KEEPER"
	LeagueTypeDynasty LeagueType = "DYNASTY"
)

type LeagueStatus string

const (
	LeagueStatusPending   LeagueStatus = "PENDING"
	LeagueStatusActive    LeagueStatus = "ACTIVE"
	LeagueStatusCompleted LeagueStatus = "COMPLETED"
	LeagueStatusCancelled LeagueStatus = "CANCELLED"
)

// League represents a fantasy sports league
type League struct {
	ID             uuid.UUID    `json:"id"`
	Name           string       `json:"name"`
	SportID        string       `json:"sport_id"`
	LeagueType     LeagueType   `json:"league_type"`
	CommissionerID uuid.UUID    `json:"commissioner_id"`
	LeagueSettings interface{}  `json:"league_settings"` // JSONB stored as interface{}
	Status         LeagueStatus `json:"league_status"`
	Season         string       `json:"season"`
	CreatedAt      time.Time    `json:"created_at"`
	UpdatedAt      time.Time    `json:"updated_at"`
}
