package teams

import (
	"time"

	"github.com/google/uuid"
)

// Team represents a sports team in the system
type Team struct {
	ID              uuid.UUID  `json:"id"`
	SportID         string     `json:"sport_id"`
	ExternalID      string     `json:"external_id"`
	Name            string     `json:"name"`
	Code            string     `json:"code"`
	City            string     `json:"city"`
	Coach           *string    `json:"coach,omitempty"`
	Owner           *string    `json:"owner,omitempty"`
	Stadium         *string    `json:"stadium,omitempty"`
	EstablishedYear *int       `json:"established_year,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

// CreateTeamRequest represents the data needed to create a new team
type CreateTeamRequest struct {
	SportID         string  `json:"sport_id" validate:"required"`
	ExternalID      string  `json:"external_id" validate:"required"`
	Name            string  `json:"name" validate:"required"`
	Code            string  `json:"code" validate:"required"`
	City            string  `json:"city" validate:"required"`
	Coach           *string `json:"coach,omitempty"`
	Owner           *string `json:"owner,omitempty"`
	Stadium         *string `json:"stadium,omitempty"`
	EstablishedYear *int    `json:"established_year,omitempty"`
}

// UpdateTeamRequest represents the data that can be updated for a team
type UpdateTeamRequest struct {
	Name            *string `json:"name,omitempty"`
	Code            *string `json:"code,omitempty"`
	City            *string `json:"city,omitempty"`
	Coach           *string `json:"coach,omitempty"`
	Owner           *string `json:"owner,omitempty"`
	Stadium         *string `json:"stadium,omitempty"`
	EstablishedYear *int    `json:"established_year,omitempty"`
}

// TeamFilter represents filtering options for team queries
type TeamFilter struct {
	SportID *string `json:"sport_id,omitempty"`
	City    *string `json:"city,omitempty"`
	Code    *string `json:"code,omitempty"`
}

// TeamSortBy represents sorting options for team queries
type TeamSortBy string

const (
	TeamSortByName            TeamSortBy = "name"
	TeamSortByCity            TeamSortBy = "city"
	TeamSortByEstablishedYear TeamSortBy = "established_year"
	TeamSortByCreatedAt       TeamSortBy = "created_at"
)

// PaginationParams represents pagination parameters
type PaginationParams struct {
	Limit  int `json:"limit" validate:"min=1,max=100"`
	Offset int `json:"offset" validate:"min=0"`
}

// TeamListResponse represents a paginated list of teams
type TeamListResponse struct {
	Teams      []Team `json:"teams"`
	Total      int    `json:"total"`
	Limit      int    `json:"limit"`
	Offset     int    `json:"offset"`
	HasMore    bool   `json:"has_more"`
}
