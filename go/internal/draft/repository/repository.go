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

type Repository struct {
	queries *db.Queries
}

func NewRepository(queries *db.Queries) *Repository {
	return &Repository{
		queries: queries,
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

type UpdateDraftRequest struct {
	Settings    *models.DraftSettings `json:"settings"`
	ScheduledAt *time.Time           `json:"scheduled_at"`
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

type NextDeadline struct {
	DraftID  uuid.UUID  `json:"draft_id"`
	Deadline *time.Time `json:"deadline"`
}

func (r *Repository) FetchNextDeadline(ctx context.Context) (*NextDeadline, error) {
	row, err := r.queries.FetchNextDeadline(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch next deadline: %w", err)
	}

	var dt *time.Time
	if row.NextDeadline.Valid {
		t := row.NextDeadline.Time
		dt = &t
	}

	return &NextDeadline{
		DraftID:  row.DraftID,
		Deadline: dt,
	}, nil
}

func (r *Repository) FetchDraftsDueForPick(ctx context.Context, limit int32) ([]uuid.UUID, error) {
	rows, err := r.queries.FetchDraftsDueForPick(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch due deadlines: %w", err)
	}

	return rows, nil
}

func (r *Repository) UpdateNextDeadline(ctx context.Context, draftID uuid.UUID, deadline *time.Time) error {
	var nd sql.NullTime
	if deadline != nil {
		nd = sql.NullTime{
			Time:  *deadline,
			Valid: true,
		}
	} else {
		nd = sql.NullTime{
			Valid: false,
		}
	}
	err := r.queries.UpdateNextDeadline(ctx, db.UpdateNextDeadlineParams{
		ID:           draftID,
		NextDeadline: nd,
	})
	if err != nil {
		return fmt.Errorf("failed to update next deadline: %w", err)
	}
	return nil
}

func (r *Repository) ClearNextDeadline(ctx context.Context, id uuid.UUID) error {
	if err := r.queries.ClearNextDeadline(ctx, id); err != nil {
		return fmt.Errorf("failed to clear next deadline: %w", err)
	}
	return nil
}

func (r *Repository) UpdateDraft(ctx context.Context, id uuid.UUID, req UpdateDraftRequest) (*models.Draft, error) {
	var settingsBytes []byte
	var scheduledAt sql.NullTime
	
	// Handle settings update
	if req.Settings != nil {
		var err error
		settingsBytes, err = json.Marshal(req.Settings)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal draft settings: %w", err)
		}
	}
	
	// Handle scheduled_at update
	if req.ScheduledAt != nil {
		scheduledAt = sql.NullTime{Time: *req.ScheduledAt, Valid: true}
	}
	
	draft, err := r.queries.UpdateDraft(ctx, db.UpdateDraftParams{
		ID:          id,
		Settings:    settingsBytes,
		ScheduledAt: scheduledAt,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update draft: %w", err)
	}
	
	return r.dbDraftToModel(draft), nil
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
