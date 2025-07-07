package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/rs/zerolog/log"
)

// setupNATSConnection creates a NATS connection with JetStream
func setupNATSConnection(natsURL string) (*nats.Conn, jetstream.JetStream, error) {
	opts := []nats.Option{
		nats.MaxReconnects(natsMaxReconnects),
		nats.ReconnectWait(natsReconnectWait),
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

	nc, err := nats.Connect(natsURL, opts...)
	if err != nil {
		return nil, nil, fmt.Errorf("connect to NATS: %w", err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, nil, fmt.Errorf("create JetStream context: %w", err)
	}

	return nc, js, nil
}

// ensureConsumer creates or gets the JetStream consumer
func (o *Orchestrator) ensureConsumer(ctx context.Context) error {
	stream, err := o.js.Stream(ctx, "DRAFT_EVENTS")
	if err != nil {
		return fmt.Errorf("get stream: %w", err)
	}

	consumerConfig := jetstream.ConsumerConfig{
		Name:          consumerName,
		Durable:       consumerName,
		Description:   "Draft orchestrator event consumer with startup replay",
		FilterSubject: "draft.events.>",
		DeliverPolicy: jetstream.DeliverAllPolicy, // Replay all events for recovery
		AckPolicy:     jetstream.AckExplicitPolicy,
		MaxDeliver:    consumerMaxDeliver,
		AckWait:       consumerAckWait,
		MaxAckPending: consumerMaxAckPending,
		ReplayPolicy:  jetstream.ReplayInstantPolicy,
	}

	// Try to get existing consumer
	consumer, err := stream.Consumer(ctx, consumerName)
	if err != nil {
		// Create new consumer
		consumer, err = stream.CreateConsumer(ctx, consumerConfig)
		if err != nil {
			return fmt.Errorf("create consumer: %w", err)
		}
		log.Info().Msg("created JetStream consumer for orchestrator")
	} else {
		log.Info().Msg("using existing JetStream consumer for orchestrator")
	}

	o.consumer = consumer
	return nil
}

// processEvent processes a single JetStream event
func (o *Orchestrator) processEvent(ctx context.Context, msg jetstream.Msg) error {
	// Parse event from message data
	var event DomainEvent
	if err := json.Unmarshal(msg.Data(), &event); err != nil {
		return fmt.Errorf("unmarshal event: %w", err)
	}

	// Parse draft ID
	draftID, err := uuid.Parse(event.DraftID)
	if err != nil {
		return fmt.Errorf("parse draft ID: %w", err)
	}

	log.Debug().
		Str("subject", msg.Subject()).
		Str("draft_id", event.DraftID).
		Str("event_type", event.EventType).
		Msg("processing orchestrator event")

	// Feed event to handler
	return o.HandleDomainEvent(ctx, event.EventType, draftID, event.Payload)
}

// Close gracefully closes the orchestrator
func (o *Orchestrator) Close() error {
	if o.nc != nil {
		o.nc.Close()
	}
	return nil
}