package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/mcdev12/dynasty/go/internal/dbconfig"
	"github.com/mcdev12/dynasty/go/internal/draft/orchestrator"
	"github.com/mcdev12/dynasty/go/internal/draft/outbox"
	outboxdb "github.com/mcdev12/dynasty/go/internal/draft/outbox/db"
	"github.com/mcdev12/dynasty/go/internal/genproto/draft/v1/draftv1connect"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Warn().Err(err).Msg("could not load .env file")
	}

	// Setup logging
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	// Get configuration
	draftServiceURL := getEnv("DRAFT_SERVICE_URL", "http://localhost:8080")
	natsURL := getEnv("NATS_URL", nats.DefaultURL)
	batchSize := int32(100)

	// Database configuration
	dbCfg := dbconfig.NewConfigFromEnv()

	// Connect to database
	db, err := sql.Open("postgres", dbCfg.DSN())
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		log.Fatal().Err(err).Msg("failed to ping database")
	}

	log.Info().
		Str("database", dbCfg.Database).
		Str("draft_service_url", draftServiceURL).
		Str("nats_url", natsURL).
		Msg("starting draft orchestrator")

	// Setup HTTP client for gRPC Connect
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Create gRPC service clients
	draftServiceClient := draftv1connect.NewDraftServiceClient(httpClient, draftServiceURL)
	draftPickServiceClient := draftv1connect.NewDraftPickServiceClient(httpClient, draftServiceURL)

	// Setup outbox app for event emission (PickStarted events)
	outboxQueries := outboxdb.New(db)
	outboxRepo := outbox.NewRepository(outboxQueries)
	outboxApp := outbox.NewApp(outboxRepo)

	// Create autopick strategy
	randStrat := orchestrator.NewRandomStrategy(draftPickServiceClient)

	// Create orchestrator
	orch := orchestrator.NewOrchestrator(
		draftServiceClient,
		draftPickServiceClient,
		outboxApp,
		randStrat,
		batchSize,
	)

	// Context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start orchestrator scheduler in background
	go func() {
		log.Info().Msg("starting orchestrator scheduler")
		if err := orch.RunScheduler(ctx); err != nil {
			log.Error().Err(err).Msg("orchestrator scheduler failed")
		}
	}()

	// Set up NATS event subscription  
	eventConsumer, err := setupEventConsumer(natsURL, orch)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to setup event consumer")
	}
	defer eventConsumer.Close()

	// Start consuming events
	go func() {
		log.Info().Msg("starting NATS event consumer")
		if err := eventConsumer.Start(ctx); err != nil {
			log.Error().Err(err).Msg("NATS event consumer failed")
		}
	}()

	// Add health check endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Start HTTP server for health checks
	server := &http.Server{
		Addr:         ":8082", // Different port from main service
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		log.Info().Str("addr", server.Addr).Msg("health check server starting")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error().Err(err).Msg("health check server failed")
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	sig := <-sigChan

	log.Info().Str("signal", sig.String()).Msg("received shutdown signal")

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	// Shutdown HTTP server
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("health check server shutdown failed")
	}

	// Cancel orchestrator context
	cancel()

	// Give orchestrator time to clean up
	time.Sleep(2 * time.Second)

	log.Info().Msg("draft orchestrator shutdown complete")
}

// OrchestratorEventConsumer handles NATS events for the orchestrator
type OrchestratorEventConsumer struct {
	nc           *nats.Conn
	js           jetstream.JetStream
	consumer     jetstream.Consumer
	orchestrator *orchestrator.Orchestrator
}

// setupEventConsumer creates a NATS event consumer for the orchestrator
func setupEventConsumer(natsURL string, orch *orchestrator.Orchestrator) (*OrchestratorEventConsumer, error) {
	opts := []nats.Option{
		nats.MaxReconnects(-1), // Infinite reconnects
		nats.ReconnectWait(2 * time.Second),
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
		return nil, fmt.Errorf("connect to NATS: %w", err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("create JetStream context: %w", err)
	}

	ec := &OrchestratorEventConsumer{
		nc:           nc,
		js:           js,
		orchestrator: orch,
	}

	// Create consumer
	if err := ec.ensureConsumer(context.Background()); err != nil {
		nc.Close()
		return nil, fmt.Errorf("ensure consumer: %w", err)
	}

	return ec, nil
}

// ensureConsumer creates or gets the JetStream consumer for orchestrator
func (ec *OrchestratorEventConsumer) ensureConsumer(ctx context.Context) error {
	stream, err := ec.js.Stream(ctx, "DRAFT_EVENTS")
	if err != nil {
		return fmt.Errorf("get stream: %w", err)
	}

	consumerConfig := jetstream.ConsumerConfig{
		Name:          "draft-orchestrator",
		Durable:       "draft-orchestrator",
		Description:   "Draft orchestrator event consumer",
		FilterSubject: "draft.events.>",
		DeliverPolicy: jetstream.DeliverLastPerSubjectPolicy,
		AckPolicy:     jetstream.AckExplicitPolicy,
		MaxDeliver:    5,
		AckWait:       30 * time.Second,
		MaxAckPending: 100,
		ReplayPolicy:  jetstream.ReplayInstantPolicy,
	}

	// Try to get existing consumer
	consumer, err := stream.Consumer(ctx, "draft-orchestrator")
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

	ec.consumer = consumer
	return nil
}

// Start begins consuming events from NATS
func (ec *OrchestratorEventConsumer) Start(ctx context.Context) error {
	log.Info().Msg("starting orchestrator NATS event consumer")

	messageCh := make(chan jetstream.Msg, 100)
	
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

	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("orchestrator event consumer shutting down")
			return nil
		case msg := <-messageCh:
			if err := ec.processMessage(ctx, msg); err != nil {
				log.Error().
					Err(err).
					Str("subject", msg.Subject()).
					Msg("failed to process message")
				if nakErr := msg.Nak(); nakErr != nil {
					log.Error().Err(nakErr).Msg("failed to NAK message")
				}
			} else {
				if ackErr := msg.Ack(); ackErr != nil {
					log.Error().Err(ackErr).Msg("failed to ACK message")
				}
			}
		}
	}
}

// processMessage processes a single NATS message
func (ec *OrchestratorEventConsumer) processMessage(ctx context.Context, msg jetstream.Msg) error {
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
		Msg("processing orchestrator event")

	// Parse draft ID
	draftID, err := uuid.Parse(envelope.DraftID)
	if err != nil {
		return fmt.Errorf("parse draft ID: %w", err)
	}

	// Handle the domain event
	if err := ec.orchestrator.HandleDomainEvent(ctx, envelope.EventType, draftID, envelope.Payload); err != nil {
		return fmt.Errorf("handle domain event: %w", err)
	}

	log.Info().
		Str("event_type", envelope.EventType).
		Str("draft_id", envelope.DraftID).
		Msg("processed orchestrator event")

	return nil
}

// Close gracefully shuts down the event consumer
func (ec *OrchestratorEventConsumer) Close() error {
	log.Info().Msg("stopping orchestrator event consumer")
	if ec.nc != nil {
		ec.nc.Close()
	}
	return nil
}

func parseUUID(s string) (uuid.UUID, error) {
	return uuid.Parse(s)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}