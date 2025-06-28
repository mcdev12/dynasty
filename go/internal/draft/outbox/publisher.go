package outbox

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
)

// MockPublisher is a simple in-memory publisher for development/testing
type MockPublisher struct {
	logger *slog.Logger
}

func NewMockPublisher(logger *slog.Logger) *MockPublisher {
	return &MockPublisher{logger: logger}
}

func (p *MockPublisher) Publish(ctx context.Context, event OutboxEvent) error {
	p.logger.Info("publishing event",
		slog.String("event_id", event.ID.String()),
		slog.String("event_type", event.EventType),
		slog.String("draft_id", event.DraftID.String()))
	return nil
}

// KafkaPublisher publishes events to Apache Kafka
type KafkaPublisher struct {
	// TODO: Add Kafka producer client
	topicPrefix string
	logger      *slog.Logger
}

func NewKafkaPublisher(topicPrefix string, logger *slog.Logger) *KafkaPublisher {
	return &KafkaPublisher{
		topicPrefix: topicPrefix,
		logger:      logger,
	}
}

func (p *KafkaPublisher) Publish(ctx context.Context, event OutboxEvent) error {
	topic := fmt.Sprintf("%s.draft.%s", p.topicPrefix, event.EventType)
	
	// Create the message envelope
	envelope := map[string]interface{}{
		"eventId":   event.ID.String(),
		"eventType": event.EventType,
		"draftId":   event.DraftID.String(),
		"timestamp": event.CreatedAt,
		"payload":   json.RawMessage(event.Payload),
	}

	messageBytes, err := json.Marshal(envelope)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// TODO: Implement actual Kafka publishing
	// producer.Produce(&kafka.Message{
	//     TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
	//     Key:            []byte(event.DraftID.String()),
	//     Value:          messageBytes,
	// }, nil)

	p.logger.Debug("would publish to Kafka",
		slog.String("topic", topic),
		slog.String("key", event.DraftID.String()),
		slog.Int("size", len(messageBytes)))

	return nil
}

// NATSPublisher publishes events to NATS/NATS Streaming
type NATSPublisher struct {
	// TODO: Add NATS connection
	subject string
	logger  *slog.Logger
}

func NewNATSPublisher(subject string, logger *slog.Logger) *NATSPublisher {
	return &NATSPublisher{
		subject: subject,
		logger:  logger,
	}
}

func (p *NATSPublisher) Publish(ctx context.Context, event OutboxEvent) error {
	subject := fmt.Sprintf("%s.draft.%s", p.subject, event.EventType)

	// Create the message envelope
	envelope := map[string]interface{}{
		"eventId":   event.ID.String(),
		"eventType": event.EventType,
		"draftId":   event.DraftID.String(),
		"timestamp": event.CreatedAt,
		"payload":   json.RawMessage(event.Payload),
	}

	messageBytes, err := json.Marshal(envelope)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// TODO: Implement actual NATS publishing
	// nc.Publish(subject, messageBytes)

	p.logger.Debug("would publish to NATS",
		slog.String("subject", subject),
		slog.Int("size", len(messageBytes)))

	return nil
}

// RabbitMQPublisher publishes events to RabbitMQ
type RabbitMQPublisher struct {
	// TODO: Add RabbitMQ channel
	exchange string
	logger   *slog.Logger
}

func NewRabbitMQPublisher(exchange string, logger *slog.Logger) *RabbitMQPublisher {
	return &RabbitMQPublisher{
		exchange: exchange,
		logger:   logger,
	}
}

func (p *RabbitMQPublisher) Publish(ctx context.Context, event OutboxEvent) error {
	routingKey := fmt.Sprintf("draft.%s", event.EventType)

	// Create the message envelope
	envelope := map[string]interface{}{
		"eventId":   event.ID.String(),
		"eventType": event.EventType,
		"draftId":   event.DraftID.String(),
		"timestamp": event.CreatedAt,
		"payload":   json.RawMessage(event.Payload),
	}

	messageBytes, err := json.Marshal(envelope)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// TODO: Implement actual RabbitMQ publishing
	// ch.Publish(
	//     p.exchange,
	//     routingKey,
	//     false,
	//     false,
	//     amqp.Publishing{
	//         ContentType: "application/json",
	//         Body:        messageBytes,
	//     },
	// )

	p.logger.Debug("would publish to RabbitMQ",
		slog.String("exchange", p.exchange),
		slog.String("routing_key", routingKey),
		slog.Int("size", len(messageBytes)))

	return nil
}