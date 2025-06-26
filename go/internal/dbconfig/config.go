package dbconfig

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds Postgres connection settings.
type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
	SSLMode  string
}

// NewConfigFromEnv reads DB_* environment variables (with defaults).
func NewConfigFromEnv() Config {
	port, err := strconv.Atoi(getEnv("DB_PORT", "5432"))
	if err != nil {
		port = 5432
	}

	return Config{
		Host:     getEnv("DB_HOST", "localhost"),
		Port:     port,
		User:     getEnv("DB_USER", "postgres"),
		Password: getEnv("DB_PASSWORD", "postgres"),
		Database: getEnv("DB_NAME", "dynasty"),
		SSLMode:  getEnv("DB_SSLMODE", "disable"),
	}
}

// DSN returns the Postgres connection URL.
func (c Config) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.User, c.Password, c.Host, c.Port, c.Database, c.SSLMode,
	)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
