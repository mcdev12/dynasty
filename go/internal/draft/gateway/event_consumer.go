package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/rs/zerolog/log"
)

// JetStreamConsumerConfig holds configuration for the JetStream consumer
type JetStreamConsumerConfig struct {
	URL               string
	StreamName        string
	ConsumerName      string
	SubjectFilter     string        // e.g., "draft.events.>"
	MaxDeliver        int           // Max delivery attempts
	AckWait           time.Duration // How long to wait for ack
	MaxAckPending     int           // Max messages pending ack
	MaxReconnects     int
	ReconnectWait     time.Duration
}

// DefaultJetStreamConsumerConfig returns default JetStream consumer configuration
func DefaultJetStreamConsumerConfig() JetStreamConsumerConfig {
	return JetStreamConsumerConfig{
		URL:           nats.DefaultURL,
		StreamName:    "DRAFT_EVENTS",
		ConsumerName:  "draft-gateway",
		SubjectFilter: "draft.events.>",
		MaxDeliver:    5,
		AckWait:       30 * time.Second,
		MaxAckPending: 100,
		MaxReconnects: -1, // Infinite
		ReconnectWait: 2 * time.Second,
	}
}

// EventConsumer consumes events from JetStream and broadcasts to WebSocket clients
type EventConsumer struct {
	connectionManager *ConnectionManager
	nc                *nats.Conn
	js                jetstream.JetStream
	consumer          jetstream.Consumer
	config            JetStreamConsumerConfig
}

// NewEventConsumer creates a new JetStream event consumer
func NewEventConsumer(cm *ConnectionManager, config JetStreamConsumerConfig) (*EventConsumer, error) {
	opts := []nats.Option{
		nats.MaxReconnects(config.MaxReconnects),
		nats.ReconnectWait(config.ReconnectWait),
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

	nc, err := nats.Connect(config.URL, opts...)
	if err != nil {
		return nil, fmt.Errorf("connect to NATS: %w", err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("create JetStream context: %w", err)
	}

	ec := &EventConsumer{
		connectionManager: cm,
		nc:                nc,
		js:                js,
		config:            config,
	}

	// Create or get consumer
	if err := ec.ensureConsumer(context.Background()); err != nil {
		nc.Close()
		return nil, fmt.Errorf("ensure consumer: %w", err)
	}

	return ec, nil
}

// ensureConsumer creates or gets the JetStream consumer
func (ec *EventConsumer) ensureConsumer(ctx context.Context) error {
	stream, err := ec.js.Stream(ctx, ec.config.StreamName)
	if err != nil {
		return fmt.Errorf("get stream: %w", err)
	}

	// Consumer configuration
	consumerConfig := jetstream.ConsumerConfig{
		Name:           ec.config.ConsumerName,
		Durable:        ec.config.ConsumerName, // Make it durable
		Description:    "Draft gateway WebSocket consumer",
		FilterSubject:  ec.config.SubjectFilter,
		DeliverPolicy:  jetstream.DeliverLastPerSubjectPolicy, // Start with latest per subject
		AckPolicy:      jetstream.AckExplicitPolicy,
		MaxDeliver:     ec.config.MaxDeliver,
		AckWait:        ec.config.AckWait,
		MaxAckPending:  ec.config.MaxAckPending,
		ReplayPolicy:   jetstream.ReplayInstantPolicy,
	}

	// Try to get existing consumer
	consumer, err := stream.Consumer(ctx, ec.config.ConsumerName)
	if err != nil {
		// Create new consumer
		consumer, err = stream.CreateConsumer(ctx, consumerConfig)
		if err != nil {
			return fmt.Errorf("create consumer: %w", err)
		}
		log.Info().
			Str("consumer", ec.config.ConsumerName).
			Str("stream", ec.config.StreamName).
			Msg("created JetStream consumer")
	} else {
		log.Info().
			Str("consumer", ec.config.ConsumerName).
			Str("stream", ec.config.StreamName).
			Msg("using existing JetStream consumer")
	}

	ec.consumer = consumer
	return nil
}

// Start begins consuming events from JetStream
func (ec *EventConsumer) Start(ctx context.Context) error {
	log.Info().
		Str("consumer", ec.config.ConsumerName).
		Str("stream", ec.config.StreamName).
		Msg("starting JetStream event consumer")

	// Create message handler
	messageCh := make(chan jetstream.Msg, 100)
	
	// Start consumer
	consumeCtx, err := ec.consumer.Consume(func(msg jetstream.Msg) {
		select {
		case messageCh <- msg:
		case <-ctx.Done():
			msg.Nak()
		}
	})
	if err != nil {
		return fmt.Errorf("start consumer: %w", err)
	}
	defer consumeCtx.Stop()

	// Process messages
	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("event consumer shutting down")
			return nil
		case msg := <-messageCh:
			if err := ec.processMessage(ctx, msg); err != nil {
				log.Error().
					Err(err).
					Str("subject", msg.Subject()).
					Msg("failed to process message")
				// Negative acknowledge to retry
				if nakErr := msg.Nak(); nakErr != nil {
					log.Error().Err(nakErr).Msg("failed to NAK message")
				}
			} else {
				// Acknowledge successful processing
				if ackErr := msg.Ack(); ackErr != nil {
					log.Error().Err(ackErr).Msg("failed to ACK message")
				}
			}
		}
	}
}

// processMessage processes a single JetStream message
func (ec *EventConsumer) processMessage(ctx context.Context, msg jetstream.Msg) error {
	// Parse the event envelope
	var envelope struct {
		EventID   string          `json:"eventId"`
		EventType string          `json:"eventType"`
		DraftID   string          `json:"draftId"`
		Timestamp time.Time       `json:"timestamp"`
		Payload   json.RawMessage `json:"payload"`
	}

	if err := json.Unmarshal(msg.Data(), &envelope); err != nil {
		return fmt.Errorf("unmarshal event envelope: %w", err)
	}

	log.Debug().
		Str("event_id", envelope.EventID).
		Str("draft_id", envelope.DraftID).
		Str("event_type", envelope.EventType).
		Str("subject", msg.Subject()).
		Msg("processing JetStream event")

	// Parse draft ID
	draftID, err := uuid.Parse(envelope.DraftID)
	if err != nil {
		return fmt.Errorf("parse draft ID: %w", err)
	}

	// Convert to WebSocket event
	wsEvent, err := ec.convertToWebSocketEvent(envelope.EventID, envelope.EventType, envelope.DraftID, envelope.Payload)
	if err != nil {
		return fmt.Errorf("convert to WebSocket event: %w", err)
	}

	// Broadcast to connected clients
	ec.connectionManager.BroadcastToDraft(draftID, wsEvent)

	log.Info().
		Str("event_id", envelope.EventID).
		Str("draft_id", envelope.DraftID).
		Str("event_type", envelope.EventType).
		Msg("event broadcasted to WebSocket clients")

	return nil
}

// convertToWebSocketEvent converts a JetStream event to WebSocket event format
func (ec *EventConsumer) convertToWebSocketEvent(eventID, eventType, draftID string, payload json.RawMessage) (*DraftEvent, error) {
	// Map event types
	var wsEventType EventType
	switch eventType {
	case "PickMade":
		wsEventType = EventTypePickMade
	case "PickStarted":
		wsEventType = EventTypePickStarted
	case "DraftStarted":
		wsEventType = EventTypeDraftStarted
	case "DraftCompleted":
		wsEventType = EventTypeDraftCompleted
	case "DraftPaused":
		wsEventType = EventTypeDraftPaused
	case "DraftResumed":
		wsEventType = EventTypeDraftResumed
	default:
		return nil, fmt.Errorf("unknown event type: %s", eventType)
	}

	wsEvent := &DraftEvent{
		ID:        eventID,
		DraftID:   draftID,
		Type:      wsEventType,
		Timestamp: time.Now(),
		Data:      payload,
	}

	return wsEvent, nil
}

// Stop gracefully shuts down the event consumer
func (ec *EventConsumer) Stop() error {
	log.Info().Msg("stopping event consumer")
	
	if ec.nc != nil {
		ec.nc.Close()
	}
	
	return nil
}

// GetConsumerInfo returns information about the consumer
func (ec *EventConsumer) GetConsumerInfo(ctx context.Context) (*jetstream.ConsumerInfo, error) {
	return ec.consumer.Info(ctx)
}