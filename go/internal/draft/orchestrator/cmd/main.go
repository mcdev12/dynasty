package main

import (
	"context"
	"database/sql"
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
	"github.com/mcdev12/dynasty/go/internal/genproto/draft/v1/draftv1connect"
	"github.com/nats-io/nats.go"
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

	// Create autopick strategy
	randStrat := orchestrator.NewRandomStrategy(draftPickServiceClient)

	// Create orchestrator
	orch, err := orchestrator.NewOrchestrator(
		draftServiceClient,
		draftPickServiceClient,
		randStrat,
		natsURL,
	)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create orchestrator")
	}

	// Context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start orchestrator scheduler in background (includes JetStream consumer)
	go func() {
		log.Info().Msg("starting orchestrator scheduler with JetStream consumer")
		if err := orch.RunScheduler(ctx); err != nil {
			log.Error().Err(err).Msg("orchestrator scheduler failed")
		}
	}()
	defer orch.Close()

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

func parseUUID(s string) (uuid.UUID, error) {
	return uuid.Parse(s)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}