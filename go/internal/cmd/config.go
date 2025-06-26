package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/mcdev12/dynasty/go/internal/sports/base"
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