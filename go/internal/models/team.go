package models

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