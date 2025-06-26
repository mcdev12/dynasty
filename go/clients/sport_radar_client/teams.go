package sport_radar_client

import (
	"encoding/json"
	"fmt"
)

// SportRadar API response structures
type SRTeam struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Alias  string `json:"alias"`
	Market string `json:"market"`
	SrID   string `json:"sr_id"`
}

type SRLeague struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Alias string `json:"alias"`
}

type SRTeamsResponse struct {
	League SRLeague `json:"league"`
	Teams  []SRTeam `json:"teams"`
}

// SRPlayer represents a player in the SportRadar API response
type SRPlayer struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	FirstName   string  `json:"first_name"`
	LastName    string  `json:"last_name"`
	Position    string  `json:"position"`
	Jersey      string  `json:"jersey"`
	Height      int     `json:"height"`
	Weight      float64 `json:"weight"`
	Experience  int     `json:"experience"`
	College     string  `json:"college"`
	BirthDate   string  `json:"birth_date"`
	BirthPlace  string  `json:"birth_place"`
	Status      string  `json:"status"`
	SrID        string  `json:"sr_id"`
}

type SRRosterResponse struct {
	ID      string     `json:"id"`
	Name    string     `json:"name"`
	Market  string     `json:"market"`
	Alias   string     `json:"alias"`
	SrID    string     `json:"sr_id"`
	Players []SRPlayer `json:"players"`
}

// GetNFLTeams retrieves all NFL teams from SportRadar API
func (c *SportRadarClient) GetNFLTeams() ([]SRTeam, error) {
	// Build endpoint: v7/{language_code}/league/teams.json
	endpoint := fmt.Sprintf("v7/%s/league/teams.json", languageCodeEnglish)

	body, err := c.Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to get NFL teams: %w", err)
	}

	var response SRTeamsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w, raw response: %s", err, string(body))
	}

	return response.Teams, nil
}

// GetTeamRoster retrieves the full roster for a specific NFL team
func (c *SportRadarClient) GetTeamRoster(teamID string) (*SRRosterResponse, error) {
	// Build endpoint: v7/{language_code}/teams/{team_id}/full_roster.json
	endpoint := fmt.Sprintf("v7/%s/teams/%s/full_roster.json", languageCodeEnglish, teamID)

	body, err := c.Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to get team roster: %w", err)
	}

	var response SRRosterResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal roster response: %w, raw response: %s", err, string(body))
	}

	return &response, nil
}

// GetTeamRosterByAlias retrieves the full roster for a team by its alias (e.g., "SF", "KC")
func (c *SportRadarClient) GetTeamRosterByAlias(alias string) (*SRRosterResponse, error) {
	// First get all teams to find the team ID
	teams, err := c.GetNFLTeams()
	if err != nil {
		return nil, fmt.Errorf("failed to get teams: %w", err)
	}

	var teamID string
	for _, team := range teams {
		if team.Alias == alias {
			teamID = team.ID
			break
		}
	}

	if teamID == "" {
		return nil, fmt.Errorf("team with alias '%s' not found", alias)
	}

	return c.GetTeamRoster(teamID)
}
