package base

import (
	"fmt"
	"sync"
)

// SportPlugin defines the interface each sport plugin must implement.
type SportPlugin interface {
	Init() error

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
func RegisterPlugin(key string, plugin SportPlugin) error {
	registryMu.Lock()
	defer registryMu.Unlock()
	if key == "" {
		return fmt.Errorf("plugin key cannot be empty")
	}
	if _, exists := registry[key]; exists {
		return fmt.Errorf("plugin already registered for key %q", key)
	}
	if err := plugin.Init(); err != nil {
		return fmt.Errorf("failed to init plugin %q: %w", key, err)
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
