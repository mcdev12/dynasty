package draft

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/mcdev12/dynasty/go/internal/draft/draft/db"
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

func (r *Repository) UpdateDraft(ctx context.Context, id uuid.UUID, req UpdateDraftRequest) (*models.Draft, error) {
	var settingsBytes []byte
	var err error
	if req.Settings != nil {
		settingsBytes, err = json.Marshal(req.Settings)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal draft settings: %w", err)
		}
	}

	var scheduledAt sql.NullTime
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

func (r *Repository) FetchNextDeadline(ctx context.Context) (*NextDeadline, error) {
	row, err := r.queries.FetchNextDeadline(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch next deadline: %w", err)
	}

	var deadline *time.Time
	if row.NextDeadline.Valid {
		deadline = &row.NextDeadline.Time
	}

	return &NextDeadline{
		DraftID:  row.DraftID,
		Deadline: deadline,
	}, nil
}

func (r *Repository) FetchDraftsDueForPick(ctx context.Context, limit int32) ([]uuid.UUID, error) {
	rows, err := r.queries.FetchDraftsDueForPick(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch drafts due for pick: %w", err)
	}
	return rows, nil
}

func (r *Repository) UpdateNextDeadline(ctx context.Context, draftID uuid.UUID, deadline *time.Time) error {
	var deadlineValue sql.NullTime
	if deadline != nil {
		deadlineValue = sql.NullTime{Time: *deadline, Valid: true}
	}

	if err := r.queries.UpdateNextDeadline(ctx, db.UpdateNextDeadlineParams{
		ID:            draftID,
		NextDeadline: deadlineValue,
	}); err != nil {
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

// Helper function to convert DB draft to model
func (r *Repository) dbDraftToModel(dbDraft db.Draft) *models.Draft {
	var settings models.DraftSettings
	if err := json.Unmarshal(dbDraft.Settings, &settings); err != nil {
		// Handle error appropriately, for now we'll use empty settings
		settings = models.DraftSettings{}
	}

	draft := &models.Draft{
		ID:        dbDraft.ID,
		LeagueID:  dbDraft.LeagueID,
		DraftType: models.DraftType(dbDraft.DraftType),
		Status:    models.DraftStatus(dbDraft.Status),
		Settings:  settings,
		CreatedAt: dbDraft.CreatedAt,
		UpdatedAt: dbDraft.UpdatedAt,
	}

	if dbDraft.ScheduledAt.Valid {
		draft.ScheduledAt = &dbDraft.ScheduledAt.Time
	}
	if dbDraft.StartedAt.Valid {
		draft.StartedAt = &dbDraft.StartedAt.Time
	}
	if dbDraft.CompletedAt.Valid {
		draft.CompletedAt = &dbDraft.CompletedAt.Time
	}

	return draft
}