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
	draftdraft "github.com/mcdev12/dynasty/go/internal/draft/draft"
	draftdb "github.com/mcdev12/dynasty/go/internal/draft/draft/db"
	"github.com/mcdev12/dynasty/go/internal/draft/gateway"
	"github.com/mcdev12/dynasty/go/internal/draft/outbox"
	outboxdb "github.com/mcdev12/dynasty/go/internal/draft/outbox/db"
	"github.com/mcdev12/dynasty/go/internal/draft/pick"
	pickdb "github.com/mcdev12/dynasty/go/internal/draft/pick/db"
	"github.com/mcdev12/dynasty/go/internal/genproto/draft/v1/draftv1connect"
	"github.com/mcdev12/dynasty/go/internal/leagues"
	leaguedb "github.com/mcdev12/dynasty/go/internal/leagues/db"
	"github.com/mcdev12/dynasty/go/internal/users"
	usersdb "github.com/mcdev12/dynasty/go/internal/users/db"
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

	// Setup service clients for state provider
	draftService, draftPickService := setupServiceClients(db)

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
	stateProvider := gateway.NewDraftStateProvider(draftService, draftPickService)

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
		Handler:      gateway.CORSMiddleware(mux),
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

func setupServiceClients(db *sql.DB) (draftv1connect.DraftServiceClient, draftv1connect.DraftPickServiceClient) {
	// Setup queries
	draftQueries := draftdb.New(db)
	pickQueries := pickdb.New(db)
	outboxQueries := outboxdb.New(db)
	leagueQueries := leaguedb.New(db)
	userQueries := usersdb.New(db)

	// Setup repositories
	draftRepo := draftdraft.NewRepository(draftQueries)
	draftPickRepo := pick.NewRepository(pickQueries, db)
	outboxRepo := outbox.NewRepository(outboxQueries)
	leagueRepo := leagues.NewRepository(leagueQueries)
	userRepo := users.NewRepository(userQueries)

	// Setup apps
	draftApp := draftdraft.NewApp(draftRepo)
	pickApp := pick.NewApp(draftPickRepo)
	outboxApp := outbox.NewApp(outboxRepo)
	leagueApp := leagues.NewApp(leagueRepo)
	userApp := users.NewApp(userRepo)

	// Create services (these will act as local clients for the gateway)
	userService := users.NewService(userApp)
	leagueService := leagues.NewService(leagueApp, userService)

	// Create draft service with outbox app and league service
	draftService := draftdraft.NewService(draftApp, outboxApp, leagueService)
	pickService := pick.NewService(pickApp, draftService, outboxApp)

	return draftService, pickService
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
