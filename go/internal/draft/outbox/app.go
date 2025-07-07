package outbox

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/mcdev12/dynasty/go/internal/draft/outbox/worker"
	"github.com/rs/zerolog/log"
)

// OutboxRepository defines what the app layer needs from the repository
type OutboxRepository interface {
	InsertOutboxPickMade(ctx context.Context, draftID uuid.UUID, payload []byte) error
	InsertOutboxPickStarted(ctx context.Context, draftID uuid.UUID, payload []byte) error
	InsertOutboxDraftStarted(ctx context.Context, draftID uuid.UUID, payload []byte) error
	InsertOutboxDraftPaused(ctx context.Context, draftID uuid.UUID, payload []byte) error
	InsertOutboxDraftResumed(ctx context.Context, draftID uuid.UUID, payload []byte) error
	InsertOutboxDraftCompleted(ctx context.Context, draftID uuid.UUID, payload []byte) error
	FetchUnsentOutbox(ctx context.Context, limit int32) ([]worker.OutboxEvent, error)
	MarkOutboxSent(ctx context.Context, id uuid.UUID) error
	FetchOutboxByID(ctx context.Context, id uuid.UUID) (*worker.OutboxEvent, error)
}

// App handles outbox business logic
type App struct {
	repo OutboxRepository
}

// NewApp creates a new outbox App
func NewApp(repo OutboxRepository) *App {
	return &App{
		repo: repo,
	}
}

// InsertPickMadeEvent inserts a PickMade event into the outbox
func (a *App) InsertPickMadeEvent(ctx context.Context, draftID uuid.UUID, payload []byte) error {
	if err := a.validateEventPayload(payload); err != nil {
		return fmt.Errorf("invalid PickMade payload: %w", err)
	}

	if err := a.repo.InsertOutboxPickMade(ctx, draftID, payload); err != nil {
		return fmt.Errorf("failed to insert PickMade event: %w", err)
	}

	log.Info().
		Str("draft_id", draftID.String()).
		Str("event_type", "PickMade").
		Msg("outbox event inserted")

	return nil
}

// InsertPickStartedEvent inserts a PickStarted event into the outbox
func (a *App) InsertPickStartedEvent(ctx context.Context, draftID uuid.UUID, payload []byte) error {
	if err := a.validateEventPayload(payload); err != nil {
		return fmt.Errorf("invalid PickStarted payload: %w", err)
	}

	if err := a.repo.InsertOutboxPickStarted(ctx, draftID, payload); err != nil {
		return fmt.Errorf("failed to insert PickStarted event: %w", err)
	}

	log.Info().
		Str("draft_id", draftID.String()).
		Str("event_type", "PickStarted").
		Msg("outbox event inserted")

	return nil
}

// InsertDraftStartedEvent inserts a DraftStarted event into the outbox
func (a *App) InsertDraftStartedEvent(ctx context.Context, draftID uuid.UUID, payload []byte) error {
	if err := a.validateEventPayload(payload); err != nil {
		return fmt.Errorf("invalid DraftStarted payload: %w", err)
	}

	if err := a.repo.InsertOutboxDraftStarted(ctx, draftID, payload); err != nil {
		return fmt.Errorf("failed to insert DraftStarted event: %w", err)
	}

	log.Info().
		Str("draft_id", draftID.String()).
		Str("event_type", "DraftStarted").
		Msg("outbox event inserted")

	return nil
}

// InsertDraftPausedEvent inserts a DraftPaused event into the outbox
func (a *App) InsertDraftPausedEvent(ctx context.Context, draftID uuid.UUID, payload []byte) error {
	if err := a.validateEventPayload(payload); err != nil {
		return fmt.Errorf("invalid DraftPaused payload: %w", err)
	}

	if err := a.repo.InsertOutboxDraftPaused(ctx, draftID, payload); err != nil {
		return fmt.Errorf("failed to insert DraftPaused event: %w", err)
	}

	log.Info().
		Str("draft_id", draftID.String()).
		Str("event_type", "DraftPaused").
		Msg("outbox event inserted")

	return nil
}

// InsertDraftResumedEvent inserts a DraftResumed event into the outbox
func (a *App) InsertDraftResumedEvent(ctx context.Context, draftID uuid.UUID, payload []byte) error {
	if err := a.validateEventPayload(payload); err != nil {
		return fmt.Errorf("invalid DraftResumed payload: %w", err)
	}

	if err := a.repo.InsertOutboxDraftResumed(ctx, draftID, payload); err != nil {
		return fmt.Errorf("failed to insert DraftResumed event: %w", err)
	}

	log.Info().
		Str("draft_id", draftID.String()).
		Str("event_type", "DraftResumed").
		Msg("outbox event inserted")

	return nil
}

// InsertDraftCompletedEvent inserts a DraftCompleted event into the outbox
func (a *App) InsertDraftCompletedEvent(ctx context.Context, draftID uuid.UUID, payload []byte) error {
	if err := a.validateEventPayload(payload); err != nil {
		return fmt.Errorf("invalid DraftCompleted payload: %w", err)
	}

	if err := a.repo.InsertOutboxDraftCompleted(ctx, draftID, payload); err != nil {
		return fmt.Errorf("failed to insert DraftCompleted event: %w", err)
	}

	log.Info().
		Str("draft_id", draftID.String()).
		Str("event_type", "DraftCompleted").
		Msg("outbox event inserted")

	return nil
}

// Alias methods to match orchestrator interface expectations
func (a *App) InsertOutboxDraftStarted(ctx context.Context, draftID uuid.UUID, payload []byte) error {
	return a.InsertDraftStartedEvent(ctx, draftID, payload)
}

func (a *App) InsertOutboxDraftCompleted(ctx context.Context, draftID uuid.UUID, payload []byte) error {
	return a.InsertDraftCompletedEvent(ctx, draftID, payload)
}

func (a *App) InsertOutboxDraftPaused(ctx context.Context, draftID uuid.UUID, payload []byte) error {
	return a.InsertDraftPausedEvent(ctx, draftID, payload)
}

func (a *App) InsertOutboxDraftResumed(ctx context.Context, draftID uuid.UUID, payload []byte) error {
	return a.InsertDraftResumedEvent(ctx, draftID, payload)
}

func (a *App) InsertOutboxPickStarted(ctx context.Context, draftID uuid.UUID, payload []byte) error {
	return a.InsertPickStartedEvent(ctx, draftID, payload)
}

// FetchUnsentEvents fetches unsent outbox events
func (a *App) FetchUnsentEvents(ctx context.Context, limit int32) ([]worker.OutboxEvent, error) {
	if limit <= 0 {
		return nil, fmt.Errorf("limit must be greater than 0")
	}

	events, err := a.repo.FetchUnsentOutbox(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch unsent events: %w", err)
	}

	if len(events) > 0 {
		log.Debug().
			Int("count", len(events)).
			Msg("fetched unsent outbox events")
	}

	return events, nil
}

// MarkEventSent marks an outbox event as sent
func (a *App) MarkEventSent(ctx context.Context, eventID uuid.UUID) error {
	if err := a.repo.MarkOutboxSent(ctx, eventID); err != nil {
		return fmt.Errorf("failed to mark event as sent: %w", err)
	}

	log.Debug().
		Str("event_id", eventID.String()).
		Msg("marked outbox event as sent")

	return nil
}

// GetEventByID fetches a specific outbox event by ID
func (a *App) GetEventByID(ctx context.Context, eventID uuid.UUID) (*worker.OutboxEvent, error) {
	event, err := a.repo.FetchOutboxByID(ctx, eventID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch event by ID: %w", err)
	}

	return event, nil
}

// ProcessUnsentEvents processes all unsent events in batches
func (a *App) ProcessUnsentEvents(ctx context.Context, batchSize int32, processor func(event worker.OutboxEvent) error) error {
	events, err := a.FetchUnsentEvents(ctx, batchSize)
	if err != nil {
		return fmt.Errorf("failed to fetch unsent events: %w", err)
	}

	processedCount := 0
	errorCount := 0

	for _, event := range events {
		if err := processor(event); err != nil {
			log.Error().
				Err(err).
				Str("event_id", event.ID.String()).
				Str("event_type", event.EventType).
				Msg("failed to process event")
			errorCount++
			continue
		}

		if err := a.MarkEventSent(ctx, event.ID); err != nil {
			log.Error().
				Err(err).
				Str("event_id", event.ID.String()).
				Msg("failed to mark event as sent after processing")
			errorCount++
			continue
		}

		processedCount++
	}

	if processedCount > 0 || errorCount > 0 {
		log.Info().
			Int("processed", processedCount).
			Int("errors", errorCount).
			Int("total", len(events)).
			Msg("processed unsent events batch")
	}

	return nil
}

// validateEventPayload validates that the event payload is not empty
func (a *App) validateEventPayload(payload []byte) error {
	if len(payload) == 0 {
		return fmt.Errorf("event payload cannot be empty")
	}
	return nil
}