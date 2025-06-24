package clients

import (
	"encoding/json"
	"fmt"
)

type SportsApiClient struct {
	*BaseClient
}

func NewSportsApiClient(apiKey string) *SportsApiClient {
	client := &SportsApiClient{
		BaseClient: NewBaseClient("https://v1.american-football.api-sports.io"),
	}

	client.SetHeader("X-RapidAPI-Key", apiKey)
	client.SetHeader("X-RapidAPI-Host", "v1.american-football.api-sports.io")

	return client
}

type Country struct {
	Name string `json:"name"`
	Code string `json:"code"`
	Flag string `json:"flag"`
}

type Team struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	Code        string  `json:"code"`
	City        string  `json:"city"`
	Coach       string  `json:"coach"`
	Owner       string  `json:"owner"`
	Stadium     string  `json:"stadium"`
	Established int     `json:"established"`
	Logo        string  `json:"logo"`
	Country     Country `json:"country"`
}

type TeamsResponse struct {
	Get        string                 `json:"get"`
	Parameters map[string]interface{} `json:"parameters"`
	Errors     interface{}            `json:"errors"`
	Results    int                    `json:"results"`
	Response   []Team                 `json:"response"`
}

func (c *SportsApiClient) GetNFLTeams() ([]Team, error) {
	body, err := c.Get("/teams?league=1&season=2023")
	if err != nil {
		return nil, fmt.Errorf("failed to get teams: %w", err)
	}

	var response TeamsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w, raw response: %s", err, string(body))
	}

	if response.Errors != nil {
		if errMap, ok := response.Errors.(map[string]interface{}); ok && len(errMap) > 0 {
			return nil, fmt.Errorf("API returned errors: %v", response.Errors)
		}
	}

	return response.Response, nil
}
