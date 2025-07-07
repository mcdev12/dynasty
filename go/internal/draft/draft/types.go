package draft

import (
	"time"

	"github.com/google/uuid"
	"github.com/mcdev12/dynasty/go/internal/models"
)

// CreateDraftRequest represents a request to create a new draft
type CreateDraftRequest struct {
	ID          uuid.UUID            `json:"id"`
	LeagueID    uuid.UUID            `json:"league_id"`
	DraftType   models.DraftType     `json:"draft_type"`
	Status      models.DraftStatus   `json:"status"`
	Settings    models.DraftSettings `json:"settings"`
	ScheduledAt *time.Time           `json:"scheduled_at"`
}

// UpdateDraftStatusRequest represents a request to update draft status
type UpdateDraftStatusRequest struct {
	Status models.DraftStatus `json:"status"`
}

// UpdateDraftRequest represents a request to update draft settings/schedule
type UpdateDraftRequest struct {
	Settings    *models.DraftSettings `json:"settings"`
	ScheduledAt *time.Time            `json:"scheduled_at"`
}

// NextDeadline represents the next deadline for a draft
type NextDeadline struct {
	DraftID  uuid.UUID  `json:"draft_id"`
	Deadline *time.Time `json:"deadline"`
}