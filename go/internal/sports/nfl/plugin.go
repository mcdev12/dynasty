package nfl

import (
	"fmt"
	"github.com/mcdev12/dynasty/go/internal/sports/base"

	sportsapi "github.com/mcdev12/dynasty/go/clients/sports_api_client"
	//"github.com/mcdev12/dynasty/internal/models"
	//"github.com/mcdev12/dynasty/internal/sports/base"
	//"github.com/mcdev12/dynasty/internal/sportsapi"
	//_ "github.com/mcdev12/dynasty/internal/sports/nfl" // register plugin
)

// NFLPlugin implements the SportPlugin interface for the NFL.
type NFLPlugin struct {
	api    *sportsapi.SportsApiClient
	config Config
}

// Config holds NFL-specific configuration.
type Config struct {
	APIBaseURL string `yaml:"api_base_url"`
	APIKey     string `yaml:"api_key"`
	PageSize   int    `yaml:"page_size"` // number of players per fetch
}

// init registers the NFL plugin with the base registry.
func init() {
	plugin := &NFLPlugin{}
	if err := base.RegisterPlugin("nfl", plugin); err != nil {
		panic(fmt.Sprintf("Failed to register NFL plugin: %v", err))
	}
}

// Init initializes the plugin, loading config and creating the API client.
func (p *NFLPlugin) Init() error {
	// Load config from YAML (skipped, use defaults or env)
	// For brevity, use hard-coded defaults or environment vars.
	p.config = Config{
		APIKey: "YOUR_API_KEY",
	}
	p.api = sportsapi.NewSportsApiClient(p.config.APIKey)
	return nil
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
