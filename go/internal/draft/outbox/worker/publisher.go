package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/rs/zerolog/log"
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
}

func NewJetStreamPublisher(cfg JetStreamConfig) (*JetStreamPublisher, error) {
	// Configure zerolog console and level if needed
	// log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
	// zerolog.SetGlobalLevel(zerolog.DebugLevel)

	opts := []nats.Option{
		nats.MaxReconnects(cfg.MaxReconnects),
		nats.ReconnectWait(cfg.ReconnectWait),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			log.Error().Err(err).Msg("NATS disconnected")
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			log.Info().Str("url", nc.ConnectedUrl()).Msg("NATS reconnected")
		}),
		nats.ErrorHandler(func(nc *nats.Conn, sub *nats.Subscription, err error) {
			log.Error().Err(err).Msg("NATS error")
		}),
	}

	nc, err := nats.Connect(cfg.URL, opts...)
	if err != nil {
		return nil, fmt.Errorf("connect to NATS: %w", err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("create JetStream context: %w", err)
	}

	p := &JetStreamPublisher{nc: nc, js: js, config: cfg}

	if err := p.ensureStream(context.Background()); err != nil {
		nc.Close()
		return nil, fmt.Errorf("ensure stream: %w", err)
	}

	return p, nil
}

func (p *JetStreamPublisher) ensureStream(ctx context.Context) error {
	sc := jetstream.StreamConfig{
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
		// Create new stream
		if _, err = p.js.CreateStream(ctx, sc); err != nil {
			return fmt.Errorf("create stream: %w", err)
		}
		log.Info().
			Str("stream", p.config.StreamName).
			Msg("created JetStream stream")
	} else {
		// Update existing if needed
		info, err := stream.Info(ctx)
		if err != nil {
			return fmt.Errorf("get stream info: %w", err)
		}
		if !isStreamConfigEqual(info.Config, sc) {
			if _, err = p.js.UpdateStream(ctx, sc); err != nil {
				return fmt.Errorf("update stream: %w", err)
			}
			log.Info().
				Str("stream", p.config.StreamName).
				Msg("updated JetStream stream")
		}
	}
	return nil
}

func (p *JetStreamPublisher) Publish(ctx context.Context, event OutboxEvent) error {
	subject := fmt.Sprintf("%s.%s", p.config.SubjectPrefix, event.EventType)

	env := map[string]interface{}{
		"eventId":   event.ID.String(),
		"eventType": event.EventType,
		"draftId":   event.DraftID.String(),
		"timestamp": time.Now().UTC(),
		"payload":   json.RawMessage(event.Payload),
	}
	data, err := json.Marshal(env)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	ack, err := p.js.PublishMsg(ctx, &nats.Msg{
		Subject: subject,
		Data:    data,
		Header: nats.Header{
			"Event-Type": []string{event.EventType},
			"Draft-ID":   []string{event.DraftID.String()},
			"Event-ID":   []string{event.ID.String()},
		},
	},
		jetstream.WithMsgID(event.ID.String()),
		jetstream.WithExpectStream(p.config.StreamName),
	)
	if err != nil {
		return fmt.Errorf("publish to JetStream: %w", err)
	}

	log.Info().
		Str("subject", subject).
		Str("event_id", event.ID.String()).
		Uint64("sequence", ack.Sequence).
		Str("stream", ack.Stream).
		Msg("published to JetStream")

	return nil
}

func (p *JetStreamPublisher) Close() error {
	if p.nc != nil {
		p.nc.Close()
	}
	return nil
}

func isStreamConfigEqual(a, b jetstream.StreamConfig) bool {
	return a.Name == b.Name &&
		a.MaxAge == b.MaxAge &&
		a.MaxMsgs == b.MaxMsgs &&
		a.Replicas == b.Replicas &&
		a.Duplicates == b.Duplicates
}
