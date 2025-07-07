package outbox

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/mcdev12/dynasty/go/internal/draft/outbox/db"
	"github.com/mcdev12/dynasty/go/internal/draft/outbox/worker"
)

type Repository struct {
	queries *db.Queries
}

func NewRepository(queries *db.Queries) *Repository {
	return &Repository{
		queries: queries,
	}
}

func (r *Repository) InsertOutboxPickMade(ctx context.Context, draftID uuid.UUID, payload []byte) error {
	err := r.queries.InsertOutboxPickMade(ctx, db.InsertOutboxPickMadeParams{
		ID:      uuid.New(),
		DraftID: draftID,
		Payload: payload,
	})
	if err != nil {
		return fmt.Errorf("failed to insert PickMade outbox event: %w", err)
	}
	return nil
}

func (r *Repository) InsertOutboxPickStarted(ctx context.Context, draftID uuid.UUID, payload []byte) error {
	err := r.queries.InsertOutboxPickStarted(ctx, db.InsertOutboxPickStartedParams{
		ID:      uuid.New(),
		DraftID: draftID,
		Payload: payload,
	})
	if err != nil {
		return fmt.Errorf("failed to insert PickStarted outbox event: %w", err)
	}
	return nil
}

func (r *Repository) InsertOutboxDraftStarted(ctx context.Context, draftID uuid.UUID, payload []byte) error {
	err := r.queries.InsertOutboxDraftStarted(ctx, db.InsertOutboxDraftStartedParams{
		ID:      uuid.New(),
		DraftID: draftID,
		Payload: payload,
	})
	if err != nil {
		return fmt.Errorf("failed to insert DraftStarted outbox event: %w", err)
	}
	return nil
}

func (r *Repository) InsertOutboxDraftPaused(ctx context.Context, draftID uuid.UUID, payload []byte) error {
	err := r.queries.InsertOutboxDraftPaused(ctx, db.InsertOutboxDraftPausedParams{
		ID:      uuid.New(),
		DraftID: draftID,
		Payload: payload,
	})
	if err != nil {
		return fmt.Errorf("failed to insert DraftPaused outbox event: %w", err)
	}
	return nil
}

func (r *Repository) InsertOutboxDraftResumed(ctx context.Context, draftID uuid.UUID, payload []byte) error {
	err := r.queries.InsertOutboxDraftResumed(ctx, db.InsertOutboxDraftResumedParams{
		ID:      uuid.New(),
		DraftID: draftID,
		Payload: payload,
	})
	if err != nil {
		return fmt.Errorf("failed to insert DraftResumed outbox event: %w", err)
	}
	return nil
}

func (r *Repository) InsertOutboxDraftCompleted(ctx context.Context, draftID uuid.UUID, payload []byte) error {
	err := r.queries.InsertOutboxDraftCompleted(ctx, db.InsertOutboxDraftCompletedParams{
		ID:      uuid.New(),
		DraftID: draftID,
		Payload: payload,
	})
	if err != nil {
		return fmt.Errorf("failed to insert DraftCompleted outbox event: %w", err)
	}
	return nil
}

func (r *Repository) FetchUnsentOutbox(ctx context.Context, limit int32) ([]worker.OutboxEvent, error) {
	rows, err := r.queries.FetchUnsentOutbox(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch unsent outbox events: %w", err)
	}

	events := make([]worker.OutboxEvent, len(rows))
	for i, row := range rows {
		events[i] = worker.OutboxEvent{
			ID:        row.ID,
			DraftID:   row.DraftID,
			EventType: row.EventType,
			Payload:   []byte(row.Payload),
		}
	}

	return events, nil
}

func (r *Repository) MarkOutboxSent(ctx context.Context, id uuid.UUID) error {
	err := r.queries.MarkOutboxSent(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to mark outbox event as sent: %w", err)
	}
	return nil
}

func (r *Repository) FetchOutboxByID(ctx context.Context, id uuid.UUID) (*worker.OutboxEvent, error) {
	row, err := r.queries.FetchOutboxByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("outbox event not found or already sent")
		}
		return nil, fmt.Errorf("failed to fetch outbox event by ID: %w", err)
	}

	return &worker.OutboxEvent{
		ID:        row.ID,
		DraftID:   row.DraftID,
		EventType: row.EventType,
		Payload:   []byte(row.Payload),
	}, nil
}
