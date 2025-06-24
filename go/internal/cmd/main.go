package main

import (
	"fmt"
	"github.com/mcdev12/dynasty/go/clients/sports_api_client"
	"log"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	fmt.Println("Hello, world from Dynasty!")

	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Could not load .env file: %v", err)
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
	for _, team := range teams {
		fmt.Printf("- %s (%s) - %s\n", team.Name, team.Code, team.City)
	}
}
