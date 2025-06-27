package leagues

import (
	"github.com/google/uuid"
	"github.com/mcdev12/dynasty/go/internal/models"
)

// CreateLeagueRequest represents the data needed to create a new league
type CreateLeagueRequest struct {
	Name           string              `json:"name" validate:"required"`
	SportID        string              `json:"sport_id" validate:"required"`
	LeagueType     models.LeagueType   `json:"league_type" validate:"required"`
	CommissionerID uuid.UUID           `json:"commissioner_id" validate:"required"`
	LeagueSettings interface{}         `json:"league_settings" validate:"required"`
	Status         models.LeagueStatus `json:"status" validate:"required"`
	Season         string              `json:"season" validate:"required"`
}

// UpdateLeagueRequest represents the data that can be updated for a league
type UpdateLeagueRequest struct {
	Name           string              `json:"name" validate:"required"`
	SportID        string              `json:"sport_id" validate:"required"`
	LeagueType     models.LeagueType   `json:"league_type" validate:"required"`
	CommissionerID uuid.UUID           `json:"commissioner_id" validate:"required"`
	LeagueSettings interface{}         `json:"league_settings" validate:"required"`
	Status         models.LeagueStatus `json:"status" validate:"required"`
	Season         string              `json:"season" validate:"required"`
}
