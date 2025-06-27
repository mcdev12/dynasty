package models

import (
	"github.com/google/uuid"
	"time"
)

type FantasyTeam struct {
	ID        uuid.UUID `json:"id"`
	LeagueID  uuid.UUID `json:"league_id"`
	OwnerID   uuid.UUID `json:"owner_id"`
	Name      string    `json:"name"`
	LogoURL   string    `json:"logo_url"`
	CreatedAt time.Time `json:"created_at"`
}
