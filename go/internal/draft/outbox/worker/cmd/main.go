package main

import (
	"context"
	"database/sql"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/mcdev12/dynasty/go/internal/dbconfig"
	"github.com/mcdev12/dynasty/go/internal/draft/outbox"
	outboxdb "github.com/mcdev12/dynasty/go/internal/draft/outbox/db"
	"github.com/mcdev12/dynasty/go/internal/draft/outbox/worker"
)

func main() {
	// load .env
	if err := godotenv.Load(); err != nil {
		log.Warn().Err(err).Msg("could not load .env file")
	}

	// configure zerolog console output and level
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
	zerolog.SetGlobalLevel(zerolog.DebugLevel)

	// DB config
	cfg := dbconfig.NewConfigFromEnv()
	dsn := cfg.DSN()
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal().Err(err).Msg("open database")
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		log.Fatal().Err(err).Msg("ping database")
	}
	log.Info().
		Str("host", cfg.Host).
		Int("port", cfg.Port).
		Str("database", cfg.Database).
		Msg("connected to database")

	// JetStream publisher
	jsCfg := worker.DefaultJetStreamConfig()
	if url := os.Getenv("NATS_URL"); url != "" {
		jsCfg.URL = url
	}
	publisher, err := worker.NewJetStreamPublisher(jsCfg)
	if err != nil {
		log.Fatal().Err(err).Msg("create JetStream publisher")
	}
	defer func() {
		if err := publisher.Close(); err != nil {
			log.Error().Err(err).Msg("close publisher")
		}
	}()

	// Listener config
	ltCfg := worker.DefaultListenerConfig()
	ltCfg.DatabaseURL = dsn
	if iv := os.Getenv("FALLBACK_INTERVAL"); iv != "" {
		if d, err := time.ParseDuration(iv); err == nil {
			ltCfg.FallbackInterval = d
		}
	}

	// Create outbox repository and app
	queries := outboxdb.New(db)
	repo := outbox.NewRepository(queries)
	app := outbox.NewApp(repo)

	listener, err := worker.NewListener(app, publisher, ltCfg)
	if err != nil {
		log.Fatal().Err(err).Msg("create outbox listener")
	}

	//GRACEFUL SHUTDOWN

	// signal‐aware context
	ctx, stop := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// run listener
	errCh := make(chan error, 1)
	go func() {
		log.Info().Msg("starting realtime listener")
		errCh <- listener.Start(ctx)
	}()

	// wait for shutdown or error
	select {
	case <-ctx.Done():
		log.Info().Msg("shutdown signal received")
		// allow in-flight work to finish
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		<-shutdownCtx.Done()
		log.Info().Msg("graceful shutdown complete")

	case err := <-errCh:
		log.Error().Err(err).Msg("listener exited unexpectedly")
	}
}
