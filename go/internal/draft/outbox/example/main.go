package main

import (
	"context"
	"database/sql"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq"
	"github.com/mcdev12/dynasty/go/internal/draft/outbox"
)

func main() {
	// Setup logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	// Connect to database
	db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal("failed to connect to database:", err)
	}
	defer db.Close()

	// Create publisher based on environment
	var publisher outbox.EventPublisher
	
	switch os.Getenv("PUBLISHER_TYPE") {
	case "kafka":
		publisher = outbox.NewKafkaPublisher("dynasty", logger)
	case "nats":
		publisher = outbox.NewNATSPublisher("dynasty", logger)
	case "rabbitmq":
		publisher = outbox.NewRabbitMQPublisher("dynasty", logger)
	default:
		// Use mock publisher for development
		publisher = outbox.NewMockPublisher(logger)
	}

	// Wrap publisher with metrics if needed
	if os.Getenv("ENABLE_METRICS") == "true" {
		metrics := outbox.NewPrometheusMetrics()
		publisher = outbox.NewMetricPublisher(publisher, metrics)
	}

	// Configure worker
	config := outbox.Config{
		PollInterval: 5 * time.Second,
		BatchSize:    100,
		MaxRetries:   3,
		RetryDelay:   time.Second,
	}

	// Override from environment if needed
	if interval := os.Getenv("OUTBOX_POLL_INTERVAL"); interval != "" {
		if d, err := time.ParseDuration(interval); err == nil {
			config.PollInterval = d
		}
	}

	// Create worker
	worker := outbox.NewWorker(db, publisher, config, logger)

	// Optionally wrap with metrics
	var relay outbox.OutboxRelay = worker
	if os.Getenv("ENABLE_METRICS") == "true" {
		metrics := outbox.NewPrometheusMetrics()
		relay = outbox.NewWorkerWithMetrics(worker, metrics)
	}

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start worker
	if err := relay.Start(ctx); err != nil {
		log.Fatal("failed to start outbox worker:", err)
	}

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	logger.Info("outbox worker started, press Ctrl+C to stop")

	// Wait for shutdown signal
	<-sigChan
	logger.Info("shutdown signal received")

	// Stop worker gracefully
	if err := relay.Stop(); err != nil {
		logger.Error("failed to stop outbox worker", slog.String("error", err.Error()))
	}

	logger.Info("outbox worker stopped")
}