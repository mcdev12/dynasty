package outbox

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type JetStreamConfig struct {
	URL             string
	StreamName      string
	SubjectPrefix   string
	MaxReconnects   int
	ReconnectWait   time.Duration
	MaxAge          time.Duration // How long to keep messages
	MaxMsgs         int64         // Max number of messages to keep
	Replicas        int           // Number of replicas for the stream
	DuplicateWindow time.Duration // Window for duplicate detection
}

func DefaultJetStreamConfig() JetStreamConfig {
	return JetStreamConfig{
		URL:             nats.DefaultURL,
		StreamName:      "DRAFT_EVENTS",
		SubjectPrefix:   "draft.events",
		MaxReconnects:   -1, // Infinite
		ReconnectWait:   2 * time.Second,
		MaxAge:          7 * 24 * time.Hour, // 7 days
		MaxMsgs:         -1,                 // No limit
		Replicas:        1,
		DuplicateWindow: 2 * time.Hour,
	}
}

type JetStreamPublisher struct {
	nc     *nats.Conn
	js     jetstream.JetStream
	config JetStreamConfig
	logger *slog.Logger
}

func NewJetStreamPublisher(config JetStreamConfig, logger *slog.Logger) (*JetStreamPublisher, error) {
	opts := []nats.Option{
		nats.MaxReconnects(config.MaxReconnects),
		nats.ReconnectWait(config.ReconnectWait),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			logger.Error("NATS disconnected", slog.String("error", err.Error()))
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			logger.Info("NATS reconnected", slog.String("url", nc.ConnectedUrl()))
		}),
		nats.ErrorHandler(func(nc *nats.Conn, sub *nats.Subscription, err error) {
			logger.Error("NATS error", slog.String("error", err.Error()))
		}),
	}

	nc, err := nats.Connect(config.URL, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}

	p := &JetStreamPublisher{
		nc:     nc,
		js:     js,
		config: config,
		logger: logger,
	}

	// Create or update the stream
	if err := p.ensureStream(context.Background()); err != nil {
		nc.Close()
		return nil, fmt.Errorf("failed to ensure stream: %w", err)
	}

	return p, nil
}

func (p *JetStreamPublisher) ensureStream(ctx context.Context) error {
	streamConfig := jetstream.StreamConfig{
		Name:        p.config.StreamName,
		Description: "Draft event stream for outbox pattern",
		Subjects:    []string{fmt.Sprintf("%s.>", p.config.SubjectPrefix)},
		Retention:   jetstream.LimitsPolicy,
		MaxAge:      p.config.MaxAge,
		MaxMsgs:     p.config.MaxMsgs,
		Storage:     jetstream.FileStorage,
		Replicas:    p.config.Replicas,
		Duplicates:  p.config.DuplicateWindow,
	}

	stream, err := p.js.Stream(ctx, p.config.StreamName)
	if err != nil {
		// Stream doesn't exist, create it
		_, err = p.js.CreateStream(ctx, streamConfig)
		if err != nil {
			return fmt.Errorf("failed to create stream: %w", err)
		}
		p.logger.Info("created JetStream stream", slog.String("name", p.config.StreamName))
	} else {
		// Stream exists, update it
		info, err := stream.Info(ctx)
		if err != nil {
			return fmt.Errorf("failed to get stream info: %w", err)
		}

		// Only update if configuration has changed
		if !isStreamConfigEqual(info.Config, streamConfig) {
			_, err = p.js.UpdateStream(ctx, streamConfig)
			if err != nil {
				return fmt.Errorf("failed to update stream: %w", err)
			}
			p.logger.Info("updated JetStream stream", slog.String("name", p.config.StreamName))
		}
	}

	return nil
}

func (p *JetStreamPublisher) Publish(ctx context.Context, event OutboxEvent) error {
	subject := fmt.Sprintf("%s.%s", p.config.SubjectPrefix, event.EventType)

	// Create message envelope
	envelope := map[string]interface{}{
		"eventId":   event.ID.String(),
		"eventType": event.EventType,
		"draftId":   event.DraftID.String(),
		"timestamp": time.Now().UTC(),
		"payload":   json.RawMessage(event.Payload),
	}

	messageBytes, err := json.Marshal(envelope)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Publish with message ID for deduplication
	pubOpts := []jetstream.PublishOpt{
		jetstream.WithMsgID(event.ID.String()),
		jetstream.WithExpectStream(p.config.StreamName),
	}

	ack, err := p.js.PublishMsg(ctx, &nats.Msg{
		Subject: subject,
		Data:    messageBytes,
		Header: nats.Header{
			"Event-Type": []string{event.EventType},
			"Draft-ID":   []string{event.DraftID.String()},
			"Event-ID":   []string{event.ID.String()},
		},
	}, pubOpts...)

	if err != nil {
		return fmt.Errorf("failed to publish to JetStream: %w", err)
	}

	p.logger.Debug("published to JetStream",
		slog.String("subject", subject),
		slog.String("event_id", event.ID.String()),
		slog.Uint64("sequence", ack.Sequence),
		slog.String("stream", ack.Stream))

	return nil
}

func (p *JetStreamPublisher) Close() error {
	if p != nil && p.nc != nil {
		p.nc.Close()
	}
	return nil
}

// Helper function to compare stream configurations
func isStreamConfigEqual(a, b jetstream.StreamConfig) bool {
	// Compare relevant fields
	return a.Name == b.Name &&
		a.MaxAge == b.MaxAge &&
		a.MaxMsgs == b.MaxMsgs &&
		a.Replicas == b.Replicas &&
		a.Duplicates == b.Duplicates
}
