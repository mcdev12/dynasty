package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/mcdev12/dynasty/go/internal/draft/db"
	"github.com/mcdev12/dynasty/go/internal/models"
)

type Querier interface {
	CreateDraft(ctx context.Context, arg db.CreateDraftParams) (db.Draft, error)
	GetDraft(ctx context.Context, id uuid.UUID) (db.Draft, error)
	UpdateDraftStatus(ctx context.Context, arg db.UpdateDraftStatusParams) (db.Draft, error)
	DeleteDraft(ctx context.Context, id uuid.UUID) error
}

type Repository struct {
	queries Querier
}

func NewRepository(querier Querier) *Repository {
	return &Repository{
		queries: querier,
	}
}

type CreateDraftRequest struct {
	ID          uuid.UUID            `json:"id"`
	LeagueID    uuid.UUID            `json:"league_id"`
	DraftType   models.DraftType     `json:"draft_type"`
	Status      models.DraftStatus   `json:"status"`
	Settings    models.DraftSettings `json:"settings"`
	ScheduledAt *time.Time           `json:"scheduled_at"`
}

type UpdateDraftStatusRequest struct {
	Status models.DraftStatus `json:"status"`
}

func (r *Repository) CreateDraft(ctx context.Context, req CreateDraftRequest) (*models.Draft, error) {
	settingsBytes, err := json.Marshal(req.Settings)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal draft settings: %w", err)
	}

	var scheduledAt sql.NullTime
	if req.ScheduledAt != nil {
		scheduledAt = sql.NullTime{Time: *req.ScheduledAt, Valid: true}
	}

	draft, err := r.queries.CreateDraft(ctx, db.CreateDraftParams{
		ID:          req.ID,
		LeagueID:    req.LeagueID,
		DraftType:   db.DraftType(req.DraftType),
		Status:      db.DraftStatus(req.Status),
		Settings:    settingsBytes,
		ScheduledAt: scheduledAt,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create draft: %w", err)
	}

	return r.dbDraftToModel(draft), nil
}

func (r *Repository) GetDraft(ctx context.Context, id uuid.UUID) (*models.Draft, error) {
	draft, err := r.queries.GetDraft(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get draft: %w", err)
	}

	return r.dbDraftToModel(draft), nil
}

func (r *Repository) UpdateDraftStatus(ctx context.Context, id uuid.UUID, req UpdateDraftStatusRequest) (*models.Draft, error) {
	draft, err := r.queries.UpdateDraftStatus(ctx, db.UpdateDraftStatusParams{
		ID:     id,
		Status: db.DraftStatus(req.Status),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update draft status: %w", err)
	}

	return r.dbDraftToModel(draft), nil
}

func (r *Repository) DeleteDraft(ctx context.Context, id uuid.UUID) error {
	if err := r.queries.DeleteDraft(ctx, id); err != nil {
		return fmt.Errorf("failed to delete draft: %w", err)
	}
	return nil
}

func (r *Repository) dbDraftToModel(dbDraft db.Draft) *models.Draft {
	var settings models.DraftSettings
	if err := json.Unmarshal(dbDraft.Settings, &settings); err != nil {
		// Handle error appropriately - could log or return empty settings
		settings = models.DraftSettings{}
	}

	var scheduledAt, startedAt, completedAt *time.Time
	if dbDraft.ScheduledAt.Valid {
		scheduledAt = &dbDraft.ScheduledAt.Time
	}
	if dbDraft.StartedAt.Valid {
		startedAt = &dbDraft.StartedAt.Time
	}
	if dbDraft.CompletedAt.Valid {
		completedAt = &dbDraft.CompletedAt.Time
	}

	return &models.Draft{
		ID:          dbDraft.ID,
		LeagueID:    dbDraft.LeagueID,
		DraftType:   models.DraftType(dbDraft.DraftType),
		Status:      models.DraftStatus(dbDraft.Status),
		Settings:    settings,
		ScheduledAt: scheduledAt,
		StartedAt:   startedAt,
		CompletedAt: completedAt,
		CreatedAt:   dbDraft.CreatedAt,
		UpdatedAt:   dbDraft.UpdatedAt,
	}
}
