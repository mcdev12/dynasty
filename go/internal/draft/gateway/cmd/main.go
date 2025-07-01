package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/mcdev12/dynasty/go/internal/dbconfig"
	"github.com/mcdev12/dynasty/go/internal/draft"
	draftdb "github.com/mcdev12/dynasty/go/internal/draft/db"
	"github.com/mcdev12/dynasty/go/internal/draft/gateway"
	"github.com/mcdev12/dynasty/go/internal/draft/repository"
	"github.com/mcdev12/dynasty/go/internal/leagues"
	leaguedb "github.com/mcdev12/dynasty/go/internal/leagues/db"
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
	port := getEnv("GATEWAY_PORT", "8081")
	natsURL := getEnv("NATS_URL", "nats://localhost:4222")

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
		Str("nats_url", natsURL).
		Str("port", port).
		Msg("starting draft gateway")

	// Setup draft app for state provider
	draftApp := setupDraftApp(db)

	// Create gateway configuration
	gatewayConfig := gateway.Config{
		ConnectionConfig: gateway.DefaultConnectionConfig(),
		JetStreamConfig: gateway.JetStreamConsumerConfig{
			URL:           natsURL,
			StreamName:    "DRAFT_EVENTS",
			ConsumerName:  "draft-gateway",
			SubjectFilter: "draft.events.>",
			MaxDeliver:    5,
			AckWait:       30 * time.Second,
			MaxAckPending: 100,
			MaxReconnects: -1,
			ReconnectWait: 2 * time.Second,
		},
	}

	// Create state provider
	stateProvider := gateway.NewDraftStateProvider(draftApp)

	// Create gateway service
	gatewayService, err := gateway.NewService(gatewayConfig, stateProvider)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create gateway service")
	}

	// Setup HTTP server
	mux := http.NewServeMux()

	// Test endpoint to verify /api routes work
	mux.HandleFunc("/api/test", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("API routes are working!"))
	})
	
	// Register gateway routes (WebSocket and REST)
	gatewayService.RegisterRoutes(mux)

	// Add health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Add service info
	mux.HandleFunc("/info", func(w http.ResponseWriter, r *http.Request) {
		stats := gatewayService.GetStats()
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"service":"draft-gateway","version":"1.0.0","connections":%d}`,
			stats["total_connections"])
	})
	
	// Debug endpoint to list all routes
	mux.HandleFunc("/debug/routes", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintf(w, "Registered routes:\n")
		fmt.Fprintf(w, "/health\n")
		fmt.Fprintf(w, "/info\n")
		fmt.Fprintf(w, "/ws/draft\n")
		fmt.Fprintf(w, "/ws/stats\n")
		fmt.Fprintf(w, "/api/drafts/active\n")
		fmt.Fprintf(w, "/api/drafts/{id}/state\n")
		fmt.Fprintf(w, "/debug/routes\n")
	})

	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", port),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start gateway service (includes event consumer and connection manager)
	go func() {
		if err := gatewayService.Start(ctx); err != nil {
			log.Error().Err(err).Msg("gateway service failed")
		}
	}()

	// Start HTTP server
	go func() {
		log.Info().Str("addr", server.Addr).Msg("HTTP server starting")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("HTTP server failed")
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
		log.Error().Err(err).Msg("HTTP server shutdown failed")
	}

	// Cancel service context to stop gateway service
	cancel()

	// Give services time to clean up
	time.Sleep(1 * time.Second)

	log.Info().Msg("draft gateway shutdown complete")
}

func setupDraftApp(db *sql.DB) draft.DraftApp {
	// Setup queries
	draftQueries := draftdb.New(db)
	leagueQueries := leaguedb.New(db)

	// Setup repositories
	draftRepo := repository.NewRepository(draftQueries)
	draftPickRepo := repository.NewDraftPickRepository(draftQueries, db)
	leagueRepo := leagues.NewRepository(leagueQueries)

	// Create draft app
	return draft.NewApp(draftRepo, draftPickRepo, leagueRepo)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
