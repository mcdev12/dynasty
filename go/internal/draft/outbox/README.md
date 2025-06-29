# Draft Outbox Pattern Implementation

This package implements the transactional outbox pattern for reliable event publishing in the draft system.

## Overview

The outbox pattern ensures that database changes and event publishing are atomic by:
1. Writing events to an outbox table in the same transaction as business data
2. Having a separate worker process that reads from the outbox and publishes events
3. Marking events as sent after successful publishing

## Real-Time Implementation

This package now supports real-time event processing using PostgreSQL LISTEN/NOTIFY combined with NATS JetStream:

### Architecture
- **PostgreSQL Trigger**: Sends notifications immediately when events are inserted
- **Real-time Worker**: Listens for notifications and processes events instantly
- **NATS JetStream**: Provides persistent, reliable message delivery with deduplication
- **Fallback Polling**: Ensures no events are missed during disconnections

## Components

### Core Types
- `OutboxEvent`: Represents an event in the outbox
- `EventPublisher`: Interface for publishing events to external systems
- `OutboxRelay`: Interface for the worker that processes the outbox

### Workers

#### Polling Worker
The `Worker` polls the outbox table for unsent events and publishes them:
- Configurable polling interval and batch size
- Automatic retry with exponential backoff
- Row-level locking to prevent duplicate processing
- Graceful shutdown support

#### Real-time Worker
The `RealtimeWorker` provides sub-second latency event processing:
- Listens to PostgreSQL notifications via LISTEN/NOTIFY
- Processes events immediately upon insertion
- Falls back to periodic polling for reliability
- Maintains separate connection for notifications
- Health check and metrics support

### Publishers
Multiple publisher implementations are provided:
- `MockPublisher`: For development/testing
- `KafkaPublisher`: Publishes to Apache Kafka (requires Kafka client)
- `NATSPublisher`: Publishes to NATS (requires NATS client)
- `RabbitMQPublisher`: Publishes to RabbitMQ (requires AMQP client)
- `JetStreamPublisher`: Publishes to NATS JetStream with persistence and deduplication

### Metrics
Optional metrics collection via:
- `MetricsCollector`: Interface for collecting metrics
- `PrometheusMetrics`: Prometheus implementation (requires prometheus client)
- `MetricPublisher`: Decorator that adds metrics to any publisher

## Usage

### In Repository (Dual Write)
```go
func (dp *DraftPickRepository) MakePick(ctx context.Context, pickID, playerID, draftID, teamID uuid.UUID, overall int32) error {
    txn, err := dp.db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    defer func() {
        if err != nil {
            _ = txn.Rollback()
        }
    }()

    // 1. Update business data
    // 2. Insert event to outbox
    // 3. Commit transaction
}
```

### Running the Polling Worker
```go
// Create publisher
publisher := outbox.NewKafkaPublisher("dynasty", logger)

// Configure worker
config := outbox.Config{
    PollInterval: 5 * time.Second,
    BatchSize:    100,
    MaxRetries:   3,
    RetryDelay:   time.Second,
}

// Create and start worker
worker := outbox.NewWorker(db, publisher, config, logger)
err := worker.Start(ctx)
```

### Running the Real-time Worker
```go
// Create JetStream publisher
jsConfig := outbox.DefaultJetStreamConfig()
publisher, err := outbox.NewJetStreamPublisher(jsConfig, logger)
if err != nil {
    log.Fatal(err)
}

// Configure real-time worker
rtConfig := outbox.DefaultRealtimeConfig()
worker, err := outbox.NewRealtimeWorker(db, publisher, rtConfig, logger)
if err != nil {
    log.Fatal(err)
}

// Start worker
err = worker.Start(ctx)
```

See `example/main.go` and `example/realtime_main.go` for complete examples.

## Configuration

### Environment variables for polling worker:
- `DATABASE_URL`: PostgreSQL connection string
- `PUBLISHER_TYPE`: kafka, nats, rabbitmq, or mock (default)
- `ENABLE_METRICS`: true/false
- `OUTBOX_POLL_INTERVAL`: Duration string (e.g., "10s")

### Environment variables for real-time worker:
- `DATABASE_URL`: PostgreSQL connection string
- `NATS_URL`: NATS server URL (default: nats://localhost:4222)
- `FALLBACK_INTERVAL`: How often to run fallback polling (default: 30s)
- `ENABLE_HEALTH_CHECK`: Enable health check endpoint (true/false)
- `HEALTH_PORT`: Port for health check server (default: 8080)

## Event Schema

Events are published with the following envelope:
```json
{
    "eventId": "uuid",
    "eventType": "PickMade",
    "draftId": "uuid",
    "timestamp": "2024-01-01T00:00:00Z",
    "payload": {
        "PickID": "uuid",
        "PlayerID": "uuid",
        "TeamID": "uuid",
        "Overall": 1
    }
}
```