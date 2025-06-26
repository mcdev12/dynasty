package main

import (
	"log"

	"github.com/joho/godotenv"
	_ "github.com/mcdev12/dynasty/go/internal/sports/nfl"
)

func main() {
	log.Println("Starting Dynasty application...")

	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Could not load .env file: %v", err)
	}

	// Load application config
	config, err := loadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Setup sports plugins
	plugins, err := setupSportsPlugins(config)
	if err != nil {
		log.Fatalf("Failed to setup sports plugins: %v", err)
	}

	// Setup database connection
	database, err := setupDatabase()
	if err != nil {
		log.Fatalf("Failed to setup database: %v", err)
	}
	defer database.Close()

	// Setup services
	services := setupServices(database, plugins)

	// Setup and start server
	server := setupServer(services)
	
	log.Printf("Server starting on %s", server.Addr)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}