package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"

	"github.com/mcdev12/dynasty/go/internal/dbconfig"
)

func setupDatabase() (*sql.DB, error) {
	cfg := dbconfig.NewConfigFromEnv()
	dsn := cfg.DSN()

	database, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	if err := database.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Printf("Connected to database: %s@%s:%d/%s",
		cfg.User, cfg.Host, cfg.Port, cfg.Database)
	return database, nil
}
