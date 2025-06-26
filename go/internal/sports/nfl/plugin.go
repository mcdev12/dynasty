package nfl

import (
	"context"
	"fmt"
	"os"
	"strconv"

	sportradarclient "github.com/mcdev12/dynasty/go/clients/sport_radar_client"
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
	SportsAPI struct {
		APIKey  string `yaml:"api_key"`
		BaseURL string `yaml:"base_url"`
	} `yaml:"sports_api"`
	SportRadar struct {
		APIKey  string `yaml:"api_key"`
		BaseURL string `yaml:"base_url"`
	} `yaml:"sport_radar"`
	PageSize int `yaml:"page_size"` // number of players per fetch
}

// init registers the NFL plugin with the base registry (without initialization).
func init() {
	plugin := &NFLPlugin{}
	if err := base.RegisterPlugin("nfl", plugin); err != nil {
		panic(fmt.Sprintf("Failed to register NFL plugin: %v", err))
	}
}

// Init initializes the plugin, loading config and creating the API clients.
func (p *NFLPlugin) Init() error {
	// Get API keys from environment
	sportsApiKey := os.Getenv("SPORTS_API_KEY")
	if sportsApiKey == "" {
		return fmt.Errorf("SPORTS_API_KEY environment variable is required for NFL plugin")
	}

	sportRadarApiKey := os.Getenv("SPORT_RADAR_API_KEY")
	if sportRadarApiKey == "" {
		return fmt.Errorf("SPORT_RADAR_API_KEY environment variable is required for NFL plugin")
	}

	// Initialize config with API keys
	p.config = Config{
		SportsAPI: struct {
			APIKey  string `yaml:"api_key"`
			BaseURL string `yaml:"base_url"`
		}{
			APIKey:  sportsApiKey,
			BaseURL: "https://api.sportsdata.io", // default base URL
		},
		SportRadar: struct {
			APIKey  string `yaml:"api_key"`
			BaseURL string `yaml:"base_url"`
		}{
			APIKey:  sportRadarApiKey,
			BaseURL: "https://api.sportradar.us", // default base URL
		},
		PageSize: 100, // default page size
	}

	// Initialize API clients with their respective keys
	p.sportsApi = sportsapiclient.NewSportsApiClient(p.config.SportsAPI.APIKey)
	p.sportRadar = sportradarclient.NewSportRadarClient(p.config.SportRadar.APIKey)

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

// FetchPlayers retrieves NFL players for a specific team by alias.
func (p *NFLPlugin) FetchPlayers(ctx context.Context, teamAlias string) ([]sportradarclient.SRPlayer, error) {
	// Fetch roster from SportRadar using team alias
	roster, err := p.sportRadar.GetTeamRosterByAlias(teamAlias)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch roster from SportRadar: %w", err)
	}

	return roster.Players, nil
}

// MapExternalPlayer maps a SportRadar player to core Player model with attached NFL profile.
func (p *NFLPlugin) MapExternalPlayer(srPlayer sportradarclient.SRPlayer) (*models.Player, error) {
	// Create NFL profile with all available fields
	profile := &models.NFLPlayerProfile{
		Position:     srPlayer.Position,
		College:      stringPtr(srPlayer.College),
		Experience:   srPlayer.Experience,
		HeightDesc:   fmt.Sprintf("%d\"", srPlayer.Height), // Convert inches to description
		WeightDesc:   fmt.Sprintf("%.0f lbs", srPlayer.Weight),
		GroupRole:    "", // TODO: Map from position to group role (Offense/Defense/Special Teams)
		JerseyNumber: 0,  // Will be set below if available
	}

	// Convert height from inches to cm
	if srPlayer.Height > 0 {
		heightCm := int(float64(srPlayer.Height) * 2.54)
		profile.HeightCm = &heightCm
	}

	// Convert weight from pounds to kg
	if srPlayer.Weight > 0 {
		weightKg := int(srPlayer.Weight * 0.453592)
		profile.WeightKg = &weightKg
	}

	// Parse jersey number
	if srPlayer.Jersey != "" {
		if jerseyNum, err := strconv.Atoi(srPlayer.Jersey); err == nil {
			profile.JerseyNumber = jerseyNum
		}
	}

	// Create base player model with attached profile
	player := &models.Player{
		SportID:          "nfl",
		ExternalID:       fmt.Sprintf("sr_%s", srPlayer.ID),
		FullName:         srPlayer.Name,
		NFLPlayerProfile: profile,
		// TODO Team id is mapped in App. Think about how to handle free agents. Should team be optional?
	}

	return player, nil
}

// Helper functions
func stringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

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
