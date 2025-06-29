package main

import (
	"context"
	"database/sql"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/mcdev12/dynasty/go/internal/dbconfig"
	"github.com/mcdev12/dynasty/go/internal/draft/outbox"
)

func main() {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Could not load .env file: %v", err)
	}

	// Setup logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	// Setup database connection
	cfg := dbconfig.NewConfigFromEnv()
	dsn := cfg.DSN()

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal("failed to open database:", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal("failed to ping database:", err)
	}

	logger.Info("connected to database",
		slog.String("host", cfg.Host),
		slog.Int("port", cfg.Port),
		slog.String("database", cfg.Database))

	// Configure JetStream
	jsConfig := outbox.DefaultJetStreamConfig()
	if url := os.Getenv("NATS_URL"); url != "" {
		jsConfig.URL = url
	}

	// Create JetStream publisher
	publisher, err := outbox.NewJetStreamPublisher(jsConfig, logger)
	if err != nil {
		log.Fatal("failed to create JetStream publisher:", err)
	}
	defer func() {
		if err := publisher.Close(); err != nil {
			logger.Error("failed to close publisher", slog.String("error", err.Error()))
		}
	}()

	// Configure realtime worker
	rtConfig := outbox.DefaultRealtimeConfig()
	rtConfig.DatabaseURL = dsn // Use the same DSN for LISTEN connection
	logger.Debug("using database URL for LISTEN connection", slog.String("database", cfg.Database))
	if interval := os.Getenv("FALLBACK_INTERVAL"); interval != "" {
		if d, err := time.ParseDuration(interval); err == nil {
			rtConfig.FallbackInterval = d
		}
	}

	// Create realtime worker
	worker, err := outbox.NewRealtimeWorker(db, publisher, rtConfig, logger)
	if err != nil {
		log.Fatal("failed to create realtime worker:", err)
	}

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start worker
	if err := worker.Start(ctx); err != nil {
		log.Fatal("failed to start realtime worker:", err)
	}

	// Setup health check endpoint
	if os.Getenv("ENABLE_HEALTH_CHECK") == "true" {
		// Get NATS connection from publisher (you'd need to expose this)
		healthChecker := outbox.NewRealtimeHealthChecker(worker, db, nil, 5*time.Minute)

		http.Handle("/health", healthChecker)
		http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
			exporter := outbox.NewPrometheusExporter(healthChecker)
			w.Header().Set("Content-Type", "text/plain")
			if _, err := w.Write([]byte(exporter.Export(r.Context()))); err != nil {
				logger.Error("failed to write metrics", slog.String("error", err.Error()))
			}
		})

		go func() {
			port := os.Getenv("HEALTH_PORT")
			if port == "" {
				port = "8080"
			}
			logger.Info("starting health check server", slog.String("port", port))
			if err := http.ListenAndServe(":"+port, nil); err != nil {
				logger.Error("health check server failed", slog.String("error", err.Error()))
			}
		}()
	}

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	logger.Info("realtime outbox worker started, press Ctrl+C to stop")

	// Wait for shutdown signal
	<-sigChan
	logger.Info("shutdown signal received, starting graceful shutdown")

	// Cancel context to signal goroutines to stop
	cancel()

	// Create a timeout for graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Stop worker gracefully
	shutdownDone := make(chan struct{})
	go func() {
		if err := worker.Stop(); err != nil {
			logger.Error("failed to stop realtime worker", slog.String("error", err.Error()))
		}
		close(shutdownDone)
	}()

	select {
	case <-shutdownDone:
		logger.Info("realtime outbox worker stopped gracefully")
	case <-shutdownCtx.Done():
		logger.Error("shutdown timeout exceeded, forcing exit")
	}
}
