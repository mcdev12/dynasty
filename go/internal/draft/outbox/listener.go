package outbox

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/mcdev12/dynasty/go/internal/draft/db"
)

type RealtimeConfig struct {
	NotifyChannel    string
	FallbackInterval time.Duration // How often to run fallback polling
	BatchSize        int32
	MaxRetries       int
	RetryDelay       time.Duration
	DatabaseURL      string // Connection string for LISTEN connection
}

func DefaultRealtimeConfig() RealtimeConfig {
	return RealtimeConfig{
		NotifyChannel:    "draft_outbox_events",
		FallbackInterval: 30 * time.Second,
		BatchSize:        100,
		MaxRetries:       3,
		RetryDelay:       time.Second,
	}
}

type RealtimeWorker struct {
	db        *sql.DB
	listenDB  *sql.DB // Separate connection for LISTEN
	queries   *db.Queries
	publisher EventPublisher
	config    RealtimeConfig
	logger    *slog.Logger

	mu       sync.Mutex
	running  bool
	stopChan chan struct{}
	wg       sync.WaitGroup

	// Metrics
	eventsProcessed uint64
	lastProcessed   time.Time
}

func NewRealtimeWorker(database *sql.DB, publisher EventPublisher, cfg RealtimeConfig, logger *slog.Logger) (*RealtimeWorker, error) {
	// For now, we'll use the same connection pool
	// In production, you'd pass the connection string separately
	listenDB, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}
	listenDB.SetMaxOpenConns(1)

	// Ensure the listen connection stays open
	listenDB.SetMaxOpenConns(1)
	listenDB.SetMaxIdleConns(1)
	listenDB.SetConnMaxLifetime(0)

	return &RealtimeWorker{
		db:        database,
		listenDB:  listenDB,
		queries:   db.New(database),
		publisher: publisher,
		config:    cfg,
		logger:    logger,
		stopChan:  make(chan struct{}),
	}, nil
}

func (w *RealtimeWorker) Start(ctx context.Context) error {
	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return fmt.Errorf("realtime worker already running")
	}
	w.running = true
	w.mu.Unlock()

	// Start listener goroutine
	w.wg.Add(1)
	go w.runListener(ctx)

	// Start fallback poller goroutine
	w.wg.Add(1)
	go w.runFallbackPoller(ctx)

	w.logger.Info("realtime outbox worker started",
		slog.String("channel", w.config.NotifyChannel),
		slog.Duration("fallback_interval", w.config.FallbackInterval))

	return nil
}

func (w *RealtimeWorker) Stop() error {
	w.mu.Lock()
	if !w.running {
		w.mu.Unlock()
		return fmt.Errorf("realtime worker not running")
	}
	w.running = false
	w.mu.Unlock()

	// Signal all goroutines to stop
	close(w.stopChan)
	
	// Wait for all goroutines to finish
	w.wg.Wait()

	// Close the listen connection if it's different from the main db connection
	// and if it's not nil
	if w.listenDB != nil && w.listenDB != w.db {
		if err := w.listenDB.Close(); err != nil {
			w.logger.Error("failed to close listen connection", slog.String("error", err.Error()))
		}
	}

	w.logger.Info("realtime outbox worker stopped")
	return nil
}

func (w *RealtimeWorker) runListener(ctx context.Context) {
	defer w.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.stopChan:
			return
		default:
			if err := w.listen(ctx); err != nil {
				w.logger.Error("listener error, retrying", slog.String("error", err.Error()))
				select {
				case <-time.After(5 * time.Second):
				case <-ctx.Done():
					return
				case <-w.stopChan:
					return
				}
			}
		}
	}
}

func (w *RealtimeWorker) listen(ctx context.Context) error {
	// Validate that we have a connection string
	if w.config.DatabaseURL == "" {
		return fmt.Errorf("database URL not configured for LISTEN connection")
	}

	// Create a listener
	listener := pq.NewListener(
		w.config.DatabaseURL,
		10*time.Second, // Min reconnect interval
		time.Minute,    // Max reconnect interval
		func(ev pq.ListenerEventType, err error) {
			if err != nil {
				w.logger.Error("listener event", slog.String("error", err.Error()))
			}
		},
	)
	defer func() {
		if err := listener.Close(); err != nil {
			w.logger.Error("failed to close listener", slog.String("error", err.Error()))
		}
	}()

	// Listen to the channel
	if err := listener.Listen(w.config.NotifyChannel); err != nil {
		return fmt.Errorf("failed to listen to channel: %w", err)
	}

	w.logger.Info("listening for PostgreSQL notifications", slog.String("channel", w.config.NotifyChannel))

	// Process any pending events first
	w.processPendingEvents(ctx)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-w.stopChan:
			return nil
		case notification := <-listener.Notify:
			if notification == nil {
				// Notification channel was closed, reconnect
				continue
			}

			// Process the specific event
			eventID, err := uuid.Parse(notification.Extra)
			if err != nil {
				w.logger.Error("invalid event ID in notification",
					slog.String("payload", notification.Extra),
					slog.String("error", err.Error()))
				continue
			}

			w.logger.Debug("received notification", slog.String("event_id", eventID.String()))

			if err := w.processEvent(ctx, eventID); err != nil {
				w.logger.Error("failed to process event",
					slog.String("event_id", eventID.String()),
					slog.String("error", err.Error()))
			}

		case <-time.After(90 * time.Second):
			// Ping to keep connection alive
			if err := listener.Ping(); err != nil {
				return fmt.Errorf("ping failed: %w", err)
			}
		}
	}
}

func (w *RealtimeWorker) processEvent(ctx context.Context, eventID uuid.UUID) error {
	txn, err := w.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = txn.Rollback()
		}
	}()

	qtx := w.queries.WithTx(txn)

	// Fetch the specific event with row lock
	events, err := qtx.FetchUnsentOutbox(ctx, 1)
	if err != nil {
		return fmt.Errorf("failed to fetch event: %w", err)
	}

	// Find our specific event
	var targetEvent *db.FetchUnsentOutboxRow
	for _, event := range events {
		if event.ID == eventID {
			targetEvent = &event
			break
		}
	}

	if targetEvent == nil {
		// Event might have been processed by another worker
		_ = txn.Rollback()
		return nil
	}

	// Process the event
	outboxEvent := OutboxEvent{
		ID:        targetEvent.ID,
		DraftID:   targetEvent.DraftID,
		EventType: targetEvent.EventType,
		Payload:   targetEvent.Payload,
	}

	if err := w.publishWithRetry(ctx, outboxEvent); err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	// Mark as sent
	if err := qtx.MarkOutboxSent(ctx, []uuid.UUID{targetEvent.ID}); err != nil {
		return fmt.Errorf("failed to mark event as sent: %w", err)
	}

	// Commit transaction
	if err := txn.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	w.mu.Lock()
	w.eventsProcessed++
	w.lastProcessed = time.Now()
	w.mu.Unlock()

	w.logger.Info("processed event",
		slog.String("event_id", targetEvent.ID.String()),
		slog.String("event_type", targetEvent.EventType))

	return nil
}

func (w *RealtimeWorker) runFallbackPoller(ctx context.Context) {
	defer w.wg.Done()

	ticker := time.NewTicker(w.config.FallbackInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.stopChan:
			return
		case <-ticker.C:
			w.processPendingEvents(ctx)
		}
	}
}

func (w *RealtimeWorker) processPendingEvents(ctx context.Context) {
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

	// Fetch unsent events
	events, err := qtx.FetchUnsentOutbox(ctx, w.config.BatchSize)
	if err != nil {
		w.logger.Error("failed to fetch unsent events", slog.String("error", err.Error()))
		return
	}

	if len(events) == 0 {
		_ = txn.Rollback()
		return
	}

	w.logger.Debug("processing pending events", slog.Int("count", len(events)))

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
				slog.String("error", err.Error()))
			continue
		}

		successfulIDs = append(successfulIDs, event.ID)
	}

	if len(successfulIDs) > 0 {
		if err := qtx.MarkOutboxSent(ctx, successfulIDs); err != nil {
			w.logger.Error("failed to mark events as sent", slog.String("error", err.Error()))
			return
		}

		w.mu.Lock()
		w.eventsProcessed += uint64(len(successfulIDs))
		w.lastProcessed = time.Now()
		w.mu.Unlock()
	}

	if err := txn.Commit(); err != nil {
		w.logger.Error("failed to commit transaction", slog.String("error", err.Error()))
		return
	}

	if len(successfulIDs) > 0 {
		w.logger.Info("processed pending events",
			slog.Int("total", len(events)),
			slog.Int("successful", len(successfulIDs)))
	}
}

func (w *RealtimeWorker) publishWithRetry(ctx context.Context, event OutboxEvent) error {
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

func (w *RealtimeWorker) Stats() (uint64, time.Time) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.eventsProcessed, w.lastProcessed
}
