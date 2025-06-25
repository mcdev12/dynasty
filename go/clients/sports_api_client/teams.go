package sports_api_client

import (
	"encoding/json"
	"fmt"
)

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
	return c.GetTeamsByLeagueAndSeason(NFLLeagueID, Season2025)
}

func (c *SportsApiClient) GetTeamsByLeagueAndSeason(leagueID, season string) ([]Team, error) {
	endpoint := fmt.Sprintf("%s?league=%s&season=%s", TeamsEndpoint, leagueID, season)
	body, err := c.Get(endpoint)
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
