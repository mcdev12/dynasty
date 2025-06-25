package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"connectrpc.com/grpcreflect"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/mcdev12/dynasty/go/internal/genproto/team/v1/teamv1connect"
	"github.com/mcdev12/dynasty/go/internal/sports/base"
	_ "github.com/mcdev12/dynasty/go/internal/sports/nfl"
	"github.com/mcdev12/dynasty/go/internal/teams"
	"github.com/mcdev12/dynasty/go/internal/teams/db"
	"github.com/rs/cors"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Sports struct {
		EnabledPlugins []string                          `yaml:"enabled_plugins"`
		Plugins        map[string]map[string]interface{} `yaml:"plugins"`
	} `yaml:"sports"`
}

type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
	SSLMode  string
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &config, nil
}

func setupDatabase() (*pgxpool.Pool, error) {
	dbConfig := DatabaseConfig{
		Host:     getEnv("DB_HOST", "localhost"),
		Port:     getEnvAsInt("DB_PORT", 5432),
		User:     getEnv("DB_USER", "postgres"),
		Password: getEnv("DB_PASSWORD", "postgres"),
		Database: getEnv("DB_NAME", "dynasty"),
		SSLMode:  getEnv("DB_SSLMODE", "disable"),
	}

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		dbConfig.User, dbConfig.Password, dbConfig.Host, dbConfig.Port, dbConfig.Database, dbConfig.SSLMode)

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Printf("Connected to database: %s@%s:%d/%s", dbConfig.User, dbConfig.Host, dbConfig.Port, dbConfig.Database)
	return pool, nil
}

func setupSportsPlugins(config *Config) (map[string]base.SportPlugin, error) {
	plugins := make(map[string]base.SportPlugin)
	for _, key := range config.Sports.EnabledPlugins {
		// Initialize the plugin now that environment variables are loaded
		if err := base.InitializePlugin(key); err != nil {
			return nil, fmt.Errorf("failed to initialize plugin %s: %w", key, err)
		}

		plg, err := base.GetPlugin(key)
		if err != nil {
			return nil, fmt.Errorf("failed to get plugin %s: %w", key, err)
		}

		log.Printf("Successfully initialized and loaded plugin: %s", key)
		plugins[key] = plg
	}
	return plugins, nil
}

func main() {
	log.Println("Starting Dynasty application...")

	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Could not load .env file: %v", err)
	}

	// Load application config
	config, err := loadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Setup sports plugins
	var plugins map[string]base.SportPlugin
	if plugins, err = setupSportsPlugins(config); err != nil {
		log.Fatalf("Failed to setup sports plugins: %v", err)
	}

	// Setup database connection
	dbPool, err := setupDatabase()
	if err != nil {
		log.Fatalf("Failed to setup database: %v", err)
	}
	defer dbPool.Close()

	// Wire up dependency injection chain
	// Database layer -> Repository layer -> App layer -> Service layer
	queries := db.New(dbPool)
	teamsRepo := teams.NewRepository(queries)
	teamsApp := teams.NewApp(teamsRepo, plugins)
	teamsService := teams.NewService(teamsApp)

	// Setup HTTP server with gRPC services
	mux := http.NewServeMux()

	// Add CORS middleware
	c := cors.New(cors.Options{
		AllowedMethods: []string{
			http.MethodHead,
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
		},
		AllowedOrigins: []string{"*"},
		AllowedHeaders: []string{"*"},
	})

	// Register team service
	teamServicePath, teamServiceHandler := teamv1connect.NewTeamServiceHandler(teamsService)
	mux.Handle(teamServicePath, teamServiceHandler)

	// Setup reflection for grpcui/grpcurl
	reflector := grpcreflect.NewStaticReflector(
		teamv1connect.TeamServiceName,
	)
	mux.Handle(grpcreflect.NewHandlerV1(reflector))
	mux.Handle(grpcreflect.NewHandlerV1Alpha(reflector))

	// Add health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("OK")); err != nil {
			log.Printf("Failed to write health check response: %v", err)
		}
	})

	// Wrap with CORS
	handler := c.Handler(mux)

	// Setup HTTP/2 server
	server := &http.Server{
		Addr:    fmt.Sprintf(":%s", getEnv("PORT", "8080")),
		Handler: h2c.NewHandler(handler, &http2.Server{}),
	}

	log.Printf("Server starting on %s", server.Addr)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
