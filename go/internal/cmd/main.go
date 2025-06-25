package main

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/mcdev12/dynasty/go/clients/sports_api_client"
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
	fmt.Println(teams)
}
