package base

import (
	"context"
	"fmt"
	"sync"

	sportsapi "github.com/mcdev12/dynasty/go/clients/sports_api_client"
	"github.com/mcdev12/dynasty/go/internal/models"
)

// SportPlugin defines the interface each sport plugin must implement.
type SportPlugin interface {
	Init() error
	FetchTeams(ctx context.Context) ([]sportsapi.Team, error)
	MapExternalTeam(apiTeam sportsapi.Team, sportID string) (*models.Team, error)

	//DefaultScoringTemplates() map[string][]ScoringRule
	//ValidateRoster(r *Roster) error
	//
	//MapExternalPlayer(raw json.RawMessage) (*Player, error)
	//MapExternalStats(raw json.RawMessage) (*Stats, error)
	//MapExternalStatus(raw json.RawMessage) (*Status, error)
	//
	//FetchPlayers(ctx context.Context, since time.Time) ([]json.RawMessage, error)
	//FetchStats(ctx context.Context, gameID string) ([]json.RawMessage, error)
	//FetchStatus(ctx context.Context, since time.Time) ([]json.RawMessage, error)
	//
	//ValidateTrade(proposal *TradeProposal) error
	//MapExternalTrade(raw json.RawMessage) (*TradeProposal, error)
	//
	//// Optional hook for post-scoring adjustments
	//PostApplyAdjustments(results []ScoreResult) []ScoreResult
}

var (
	registry   = make(map[string]SportPlugin)
	registryMu sync.RWMutex
)

// RegisterPlugin adds a plugin implementation under a key.
// It should be called in each sport plugin's init() function.
// The plugin will be initialized later when retrieved.
func RegisterPlugin(key string, plugin SportPlugin) error {
	registryMu.Lock()
	defer registryMu.Unlock()
	if key == "" {
		return fmt.Errorf("plugin key cannot be empty")
	}
	if _, exists := registry[key]; exists {
		return fmt.Errorf("plugin already registered for key %q", key)
	}
	registry[key] = plugin
	return nil
}

// GetPlugin retrieves a plugin by key or returns an error if not found.
func GetPlugin(key string) (SportPlugin, error) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	plugin, exists := registry[key]
	if !exists {
		return nil, fmt.Errorf("no sport plugin registered for key %q", key)
	}
	return plugin, nil
}

// InitializePlugin initializes a specific plugin.
func InitializePlugin(key string) error {
	registryMu.Lock()
	defer registryMu.Unlock()
	plugin, exists := registry[key]
	if !exists {
		return fmt.Errorf("no sport plugin registered for key %q", key)
	}
	if err := plugin.Init(); err != nil {
		return fmt.Errorf("failed to init plugin %q: %w", key, err)
	}
	return nil
}
