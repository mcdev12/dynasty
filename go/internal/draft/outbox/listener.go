package outbox

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/google/uuid"
	"github.com/lib/pq"
	draftdb "github.com/mcdev12/dynasty/go/internal/draft/db"
	"github.com/rs/zerolog/log"
	"time"
)

type ListenerConfig struct {
	DatabaseURL      string        // Postgres DSN for LISTEN/NOTIFY
	NotifyChannel    string        // Channel name to LISTEN on
	FallbackInterval time.Duration // How often to poll for missed events
	MaxRetries       int
	RetryDelay       time.Duration
	PingInterval     time.Duration
	BatchSize        int32 // Max events to fetch per batch
}

func DefaultListenerConfig() ListenerConfig {
	return ListenerConfig{
		DatabaseURL:      "",
		NotifyChannel:    "draft_outbox_events",
		FallbackInterval: 30 * time.Second,
		MaxRetries:       5,
		RetryDelay:       200 * time.Millisecond,
		PingInterval:     90 * time.Second,
		BatchSize:        100,
	}
}

// Publisher is an interface that defines our publisher.
type Publisher interface {
	Publish(ctx context.Context, event OutboxEvent) error
}

type Listener struct {
	db        *sql.DB
	queries   *draftdb.Queries
	listener  *pq.Listener
	publisher Publisher
	cfg       ListenerConfig
}

// TODO worker pool if we need to in the future
func NewListener(dbConn *sql.DB, publisher Publisher, cfg ListenerConfig) (*Listener, error) {
	l := pq.NewListener(
		cfg.DatabaseURL,
		10*time.Second,
		time.Minute,
		func(ev pq.ListenerEventType, err error) {
			if err != nil {
				log.Error().Err(err).Msg("listener event")
			}
		},
	)
	if err := l.Listen(cfg.NotifyChannel); err != nil {
		return nil, fmt.Errorf("failed to listen to channel: %w", err)
	}

	log.Info().
		Str("channel", cfg.NotifyChannel).
		Msg("listening for notifications")

	return &Listener{
		db:        dbConn,
		queries:   draftdb.New(dbConn),
		listener:  l,
		publisher: publisher,
		cfg:       cfg,
	}, nil
}

func (l *Listener) Start(ctx context.Context) error {
	log.Info().
		Str("channel", l.cfg.NotifyChannel).
		Dur("ping_interval", l.cfg.PingInterval).
		Dur("fallback_interval", l.cfg.FallbackInterval).
		Msg("listener started")

	pingTicker := time.NewTicker(l.cfg.PingInterval)
	fallbackTicker := time.NewTicker(l.cfg.FallbackInterval)
	defer pingTicker.Stop()
	defer fallbackTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("listener shutting down")
			return l.Stop()
		case note := <-l.listener.Notify:
			if note == nil {
				// nil notification means channel connection was lost so reconnect
				continue
			}
			err := l.handleNotification(ctx, note.Extra)
			if err != nil {
				log.Error().Err(err).Msg("failed to handle notification")
			}
		case <-fallbackTicker.C:
			err := l.processUnsent(ctx)
			if err != nil {
				log.Error().Err(err).Msg("failed to process unsent events")
			}
		case <-pingTicker.C:
			if err := l.listener.Ping(); err != nil {
				log.Error().Err(err).Msg("failed to ping listener")
			}
		}
	}
}

func (l *Listener) Stop() error {
	return l.listener.Close()
}

// handleNotification handles a pg listen notification. Extra is the payload on the note.
// It fetches the outbox event from the db, constructs an event and then publishes it.
func (l *Listener) handleNotification(ctx context.Context, extra string) error {
	id, err := uuid.Parse(extra)
	if err != nil {
		log.Error().Err(err).Msg("invalid event ID in notification")
		return fmt.Errorf("invalid event ID in notification: %w", err)
	}

	row, err := l.queries.FetchOutboxByID(ctx, id)
	if err != nil {
		log.Error().Err(err).Msg("failed to fetch outbox event")
		return fmt.Errorf("failed to fetch outbox event: %w", err)
	}

	event := OutboxEvent{
		ID:        row.ID,
		DraftID:   row.DraftID,
		EventType: row.EventType,
		Payload:   row.Payload,
	}

	err = l.publishWithRetry(ctx, event)
	if err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}

	if err := l.queries.MarkOutboxSent(ctx, id); err != nil {
		log.Error().Err(err).Str("event_id", id.String()).Msg("failed to mark outbox event as sent")
		return err
	}

	log.Info().Str("event_id", id.String()).Msg("published and marked event as sent")
	return nil
}

// TODO Fix int32 type on batch size
// processUnsent processes unsent message in our draft outbox.
func (l *Listener) processUnsent(ctx context.Context) error {
	unsent, err := l.queries.FetchUnsentOutbox(ctx, l.cfg.BatchSize)
	if err != nil {
		log.Error().Err(err).Msg("failed to fetch unsent outbox events")
		return fmt.Errorf("failed to fetch unsent outbox events: %w", err)
	}

	for _, event := range unsent {
		outboxEvent := OutboxEvent{
			ID:        event.ID,
			DraftID:   event.DraftID,
			EventType: event.EventType,
			Payload:   event.Payload,
		}

		err := l.publishWithRetry(ctx, outboxEvent)
		if err != nil {
			log.Error().Err(err).Str("event_id", event.ID.String()).Msg("failed to publish event")
			continue
		}

		if err := l.queries.MarkOutboxSent(ctx, outboxEvent.ID); err != nil {
			log.Error().Err(err).Str("event_id", outboxEvent.ID.String()).Msg("failed to mark outbox event as sent")
			continue
		}
	}
	return nil

}

// publishWithRetry attempts to publish an outbox event with a given retry delay and max retries.
func (l *Listener) publishWithRetry(ctx context.Context, event OutboxEvent) error {
	var lastErr error

	for attempt := 0; attempt <= l.cfg.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := l.cfg.RetryDelay * time.Duration(attempt)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}

		// Try to publish
		if err := l.publisher.Publish(ctx, event); err != nil {
			lastErr = err
			log.Error().
				Err(err).
				Int("attempt", attempt+1).
				Str("event_id", event.ID.String()).
				Msg("failed to publish, retrying")
			continue
		}

		if err := l.queries.MarkOutboxSent(ctx, event.ID); err != nil {
			log.Error().Err(err).Str("event_id", event.ID.String()).Msg("failed to mark outbox event as sent")
			return err
		}

		if attempt > 0 {
			log.Info().
				Int("attempt", attempt+1).
				Str("event_id", event.ID.String()).
				Msg("publish succeeded after retry")
		}
		return nil
	}

	// All attempts exhausted
	return fmt.Errorf("publish failed after %d attempts: %w", l.cfg.MaxRetries+1, lastErr)
}
