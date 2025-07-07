package main

import (
	"context"
	"errors"
	"net/http"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	_ "github.com/mcdev12/dynasty/go/internal/sports/nfl"
	"github.com/rs/zerolog/log"
)

func main() {
	log.Info().Msg("Starting Dynasty application")

	// Listen for shutdown signals
	ctx, stop := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM,
	)
	defer stop()

	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Warn().
			Err(err).
			Msg("Could not load .env file; proceeding with existing environment")
	}

	// Load application config
	config, err := loadConfig("config.yaml")
	if err != nil {
		log.Fatal().
			Err(err).
			Msg("Failed to load config")
	}

	// Setup sports plugins
	plugins, err := setupSportsPlugins(config)
	if err != nil {
		log.Fatal().
			Err(err).
			Msg("Failed to setup sports plugins")
	}

	// Setup database connection
	database, err := setupDatabase()
	if err != nil {
		log.Fatal().
			Err(err).
			Msg("Failed to setup database")
	}
	defer database.Close()

	// Setup services
	services := setupServices(database, plugins)

	// NOTE: Draft orchestrator now runs as a separate binary
	// See go/internal/draft/orchestrator/cmd/main.go

	// Setup HTTP/gRPC server
	server := setupServer(services)

	// Start server in goroutine
	go func() {
		log.Info().
			Str("addr", server.Addr).
			Msg("Server starting")
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error().
				Err(err).
				Msg("Server terminated unexpectedly")
		}
	}()

	// Wait for shutdown signal
	<-ctx.Done()
	log.Info().Msg("Shutdown signal received")

	// Graceful shutdown
	if err := server.Shutdown(context.Background()); err != nil {
		log.Error().
			Err(err).
			Msg("Server shutdown failed")
	}
	log.Info().Msg("Server shutdown complete")
}
