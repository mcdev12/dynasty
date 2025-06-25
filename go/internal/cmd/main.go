package main

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/mcdev12/dynasty/go/clients/sports_api_client"
	"github.com/mcdev12/dynasty/go/internal/sports/base"
	_ "github.com/mcdev12/dynasty/go/internal/sports/nfl"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Sports struct {
		EnabledPlugins []string `yaml:"enabled_plugins"`
		Plugins        map[string]map[string]interface{} `yaml:"plugins"`
	} `yaml:"sports"`
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

func main() {
	fmt.Println("Hello, world from Dynasty!")

	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Could not load .env file: %v", err)
	}

	config, err := loadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	for _, pluginName := range config.Sports.EnabledPlugins {
		plugin, err := base.GetPlugin(pluginName)
		if err != nil {
			log.Fatalf("Failed to get plugin %s: %v", pluginName, err)
		}
		fmt.Printf("Successfully loaded plugin: %s\n", pluginName)
		_ = plugin
	}

	apiKey := os.Getenv("SPORTS_API_KEY")
	if apiKey == "" {
		log.Fatal("SPORTS_API_KEY environment variable is required")
	}

	client := sports_api_client.NewSportsApiClient(apiKey)
	teams, err := client.GetNFLTeams()
	if err != nil {
		log.Fatalf("Failed to get NFL teams: %v", err)
	}

	fmt.Printf("Found %d NFL teams:\n", len(teams))
	fmt.Println(teams)
}
