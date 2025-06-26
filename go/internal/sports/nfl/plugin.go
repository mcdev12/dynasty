package nfl

import (
	"context"
	"fmt"
	sportradarclient "github.com/mcdev12/dynasty/go/clients/sport_radar_client"
	"os"

	sportsapiclient "github.com/mcdev12/dynasty/go/clients/sports_api_client"
	"github.com/mcdev12/dynasty/go/internal/models"
	"github.com/mcdev12/dynasty/go/internal/sports/base"
)

// NFLPlugin implements the SportPlugin interface for the NFL.
type NFLPlugin struct {
	sportsApi  *sportsapiclient.SportsApiClient
	sportRadar *sportradarclient.SportRadarClient
	config     Config
}

// Config holds NFL-specific configuration.
type Config struct {
	APIBaseURL string `yaml:"api_base_url"`
	APIKey     string `yaml:"api_key"`
	PageSize   int    `yaml:"page_size"` // number of players per fetch
}

// init registers the NFL plugin with the base registry (without initialization).
func init() {
	plugin := &NFLPlugin{}
	if err := base.RegisterPlugin("nfl", plugin); err != nil {
		panic(fmt.Sprintf("Failed to register NFL plugin: %v", err))
	}
}

// Init initializes the plugin, loading config and creating the API client.
func (p *NFLPlugin) Init() error {
	// Get API key from environment
	sportsApiKey := os.Getenv("SPORTS_API_KEY")
	if sportsApiKey == "" {
		return fmt.Errorf("SPORTS_API_KEY environment variable is required for NFL plugin")
	}

	p.config = Config{
		APIKey: sportsApiKey,
	}
	p.sportsApi = sportsapiclient.NewSportsApiClient(p.config.APIKey)
	return nil
}

// FetchTeams retrieves NFL teams from the external API
func (p *NFLPlugin) FetchTeams(ctx context.Context) ([]sportsapiclient.Team, error) {
	// Use the sports API client to get NFL teams
	teams, err := p.sportsApi.GetNFLTeams()
	if err != nil {
		return nil, fmt.Errorf("nfl: failed to fetch teams from API: %w", err)
	}

	return teams, nil
}

// MapExternalTeam maps a sports API team to our internal teams domain model
func (p *NFLPlugin) MapExternalTeam(apiTeam sportsapiclient.Team, sportID string) (*models.Team, error) {
	team := &models.Team{
		SportID:    sportID,
		ExternalID: fmt.Sprintf("sportsapi_%d", apiTeam.ID), // Use API team ID as unique identifier
		Name:       apiTeam.Name,
		Code:       apiTeam.Code,
		City:       apiTeam.City,
	}

	// Handle optional fields
	if apiTeam.Coach != "" {
		team.Coach = &apiTeam.Coach
	}
	if apiTeam.Owner != "" {
		team.Owner = &apiTeam.Owner
	}
	if apiTeam.Stadium != "" {
		team.Stadium = &apiTeam.Stadium
	}
	if apiTeam.Established != 0 {
		team.EstablishedYear = &apiTeam.Established
	}

	return team, nil
}

// FetchPlayers retrieves NFL players updated since the given time.
//func (p *NFLPlugin) FetchPlayers(ctx context.Context, since time.Time) ([]json.RawMessage, error) {
//	// sportsapi.FetchPlayers returns raw JSON objects for players
//	payloads, err := p.api.FetchPlayers(ctx, "nfl", since)
//	if err != nil {
//		return nil, fmt.Errorf("nfl: FetchPlayers error: %w", err)
//	}
//	// Convert payload structs to RawMessage
//	raws := make([]json.RawMessage, len(payloads))
//	for i, payload := range payloads {
//		raw, err := json.Marshal(payload)
//		if err != nil {
//			return nil, fmt.Errorf("nfl: marshal player payload: %w", err)
//		}
//		raws[i] = raw
//	}
//	return raws, nil
//}

// MapExternalPlayer maps a raw JSON payload to a core Player model.
//func (p *NFLPlugin) MapExternalPlayer(raw json.RawMessage) (*models.Player, error) {
//	var tmp struct {
//		ID         string   `json:"id"`
//		ExternalID string   `json:"external_id"`
//		Name       string   `json:"name"`
//		Positions  []string `json:"positions"`
//		TeamID     string   `json:"team_id"`
//	}
//	if err := json.Unmarshal(raw, &tmp); err != nil {
//		return nil, fmt.Errorf("nfl: MapExternalPlayer unmarshal error: %w", err)
//	}
//	player := &models.Player{
//		ID:         tmp.ID,
//		SportID:    "nfl",
//		ExternalID: tmp.ExternalID,
//		FullName:   tmp.Name,
//		Positions:  tmp.Positions,
//		TeamID:     tmp.TeamID,
//	}
//	return player, nil
//}

// The rest of SportPlugin methods can have stub implementations or be left unimplemented.

// DefaultScoringTemplates returns default NFL scoring rules.
//func (p *NFLPlugin) DefaultScoringTemplates() map[string][]models.ScoringRule {
//	return nil
//}

// ValidateRoster validates an NFL roster.
//func (p *NFLPlugin) ValidateRoster(r *models.Roster) error { return nil }
//
//func (p *NFLPlugin) MapExternalStats(raw json.RawMessage) (*models.Stats, error) {
//	return nil, nil
//}
//
//func (p *NFLPlugin) MapExternalStatus(raw json.RawMessage) (*models.Status, error) {
//	return nil, nil
//}
//
//func (p *NFLPlugin) FetchStats(ctx context.Context, gameID string) ([]json.RawMessage, error) {
//	return nil, nil
//}
//
//func (p *NFLPlugin) FetchStatus(ctx context.Context, since time.Time) ([]json.RawMessage, error) {
//	return nil, nil
//}
//
//func (p *NFLPlugin) ValidateTrade(proposal *models.TradeProposal) error { return nil }
//
//func (p *NFLPlugin) MapExternalTrade(raw json.RawMessage) (*models.TradeProposal, error) {
//	return nil, nil
//}
//
//func (p *NFLPlugin) PostApplyAdjustments(results []models.ScoreResult) []models.ScoreResult {
//	return results
//}
