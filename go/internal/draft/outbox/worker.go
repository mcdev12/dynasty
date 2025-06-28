package outbox

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/mcdev12/dynasty/go/internal/draft/db"
)

type Config struct {
	PollInterval time.Duration
	BatchSize    int32
	MaxRetries   int
	RetryDelay   time.Duration
}

func DefaultConfig() Config {
	return Config{
		PollInterval: 5 * time.Second,
		BatchSize:    100,
		MaxRetries:   3,
		RetryDelay:   time.Second,
	}
}

type Worker struct {
	db        *sql.DB
	queries   *db.Queries
	publisher EventPublisher
	config    Config
	logger    *slog.Logger

	mu       sync.Mutex
	running  bool
	stopChan chan struct{}
	wg       sync.WaitGroup
}

func NewWorker(database *sql.DB, publisher EventPublisher, cfg Config, logger *slog.Logger) *Worker {
	return &Worker{
		db:        database,
		queries:   db.New(database),
		publisher: publisher,
		config:    cfg,
		logger:    logger,
		stopChan:  make(chan struct{}),
	}
}

func (w *Worker) Start(ctx context.Context) error {
	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return fmt.Errorf("outbox worker already running")
	}
	w.running = true
	w.mu.Unlock()

	w.wg.Add(1)
	go w.run(ctx)

	w.logger.Info("outbox worker started",
		slog.Duration("poll_interval", w.config.PollInterval),
		slog.Int("batch_size", int(w.config.BatchSize)))

	return nil
}

func (w *Worker) Stop() error {
	w.mu.Lock()
	if !w.running {
		w.mu.Unlock()
		return fmt.Errorf("outbox worker not running")
	}
	w.running = false
	w.mu.Unlock()

	close(w.stopChan)
	w.wg.Wait()

	w.logger.Info("outbox worker stopped")
	return nil
}

func (w *Worker) run(ctx context.Context) {
	defer w.wg.Done()

	ticker := time.NewTicker(w.config.PollInterval)
	defer ticker.Stop()

	// Process immediately on start
	w.processOutbox(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.stopChan:
			return
		case <-ticker.C:
			w.processOutbox(ctx)
		}
	}
}

func (w *Worker) processOutbox(ctx context.Context) {
	txn, err := w.db.BeginTx(ctx, nil)
	if err != nil {
		w.logger.Error("failed to begin transaction", slog.String("error", err.Error()))
		return
	}
	defer func() {
		if err != nil {
			_ = txn.Rollback()
		}
	}()

	qtx := w.queries.WithTx(txn)

	// Fetch unsent events with row locking
	events, err := qtx.FetchUnsentOutbox(ctx, w.config.BatchSize)
	if err != nil {
		w.logger.Error("failed to fetch unsent events", slog.String("error", err.Error()))
		return
	}

	if len(events) == 0 {
		_ = txn.Rollback()
		return
	}

	w.logger.Debug("processing outbox events", slog.Int("count", len(events)))

	// Process each event
	var successfulIDs []uuid.UUID
	for _, event := range events {
		outboxEvent := OutboxEvent{
			ID:        event.ID,
			DraftID:   event.DraftID,
			EventType: event.EventType,
			Payload:   event.Payload,
		}

		if err := w.publishWithRetry(ctx, outboxEvent); err != nil {
			w.logger.Error("failed to publish event",
				slog.String("event_id", event.ID.String()),
				slog.String("event_type", event.EventType),
				slog.String("error", err.Error()))
			continue
		}

		successfulIDs = append(successfulIDs, event.ID)
	}

	// Mark successful events as sent
	if len(successfulIDs) > 0 {
		if err := qtx.MarkOutboxSent(ctx, successfulIDs); err != nil {
			w.logger.Error("failed to mark events as sent", slog.String("error", err.Error()))
			return
		}
	}

	// Commit transaction
	if err := txn.Commit(); err != nil {
		w.logger.Error("failed to commit transaction", slog.String("error", err.Error()))
		return
	}

	w.logger.Info("processed outbox events",
		slog.Int("total", len(events)),
		slog.Int("successful", len(successfulIDs)))
}

func (w *Worker) publishWithRetry(ctx context.Context, event OutboxEvent) error {
	var lastErr error

	for attempt := 0; attempt <= w.config.MaxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(w.config.RetryDelay * time.Duration(attempt)):
			}
		}

		if err := w.publisher.Publish(ctx, event); err != nil {
			lastErr = err
			w.logger.Warn("failed to publish event, retrying",
				slog.String("event_id", event.ID.String()),
				slog.Int("attempt", attempt+1),
				slog.String("error", err.Error()))
			continue
		}

		return nil
	}

	return fmt.Errorf("failed after %d attempts: %w", w.config.MaxRetries+1, lastErr)
}
