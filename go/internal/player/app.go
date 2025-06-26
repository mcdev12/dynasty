package player

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	sportradarclient "github.com/mcdev12/dynasty/go/clients/sport_radar_client"
	"github.com/mcdev12/dynasty/go/internal/models"
	"github.com/mcdev12/dynasty/go/internal/sports/base"
)

// PlayerRepository defines what the app layer needs from the repository
type PlayerRepository interface {
	CreatePlayer(ctx context.Context, req CreatePlayerRequest) (*models.Player, error)
	GetPlayer(ctx context.Context, id uuid.UUID) (*models.Player, error)
	GetPlayerByExternalID(ctx context.Context, sportID, externalID string) (*models.Player, error)
	UpdatePlayerAndProfile(ctx context.Context, playerID uuid.UUID, fullName string, teamID *uuid.UUID, profile models.Profile) (*models.Player, error)
	DeletePlayer(ctx context.Context, id uuid.UUID) error
}

type TeamApp interface {
	GetTeamBySportIdAndCode(ctx context.Context, sportID, code string) (*models.Team, error)
}

// SyncResult represents the result of syncing players from external API
type SyncResult struct {
	TotalProcessed int     `json:"total_processed"`
	Created        int     `json:"created"`
	Updated        int     `json:"updated"`
	Errors         []error `json:"errors,omitempty"`
}

// App handles player business logic
type App struct {
	repo    PlayerRepository
	teamApp TeamApp
	plugins map[string]base.SportPlugin
}

// NewApp creates a new player App
func NewApp(repo PlayerRepository, plugins map[string]base.SportPlugin, teamApp TeamApp) *App {
	return &App{
		repo:    repo,
		teamApp: teamApp,
		plugins: plugins,
	}
}

// CreatePlayer creates a new player with validation
func (a *App) CreatePlayer(ctx context.Context, req CreatePlayerRequest) (*models.Player, error) {
	if err := a.validateCreatePlayerRequest(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Check if player with same external ID already exists
	existingPlayer, err := a.repo.GetPlayerByExternalID(ctx, req.SportID, req.ExternalID)
	if err == nil && existingPlayer != nil {
		return nil, fmt.Errorf("player with external ID %s already exists for sport %s", req.ExternalID, req.SportID)
	}

	player, err := a.repo.CreatePlayer(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create player: %w", err)
	}

	return player, nil
}

// GetPlayer retrieves a player by ID
func (a *App) GetPlayer(ctx context.Context, id uuid.UUID) (*models.Player, error) {
	player, err := a.repo.GetPlayer(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get player: %w", err)
	}
	return player, nil
}

// GetPlayerByExternalID retrieves a player by sport ID and external ID
func (a *App) GetPlayerByExternalID(ctx context.Context, sportID, externalID string) (*models.Player, error) {
	player, err := a.repo.GetPlayerByExternalID(ctx, sportID, externalID)
	if err != nil {
		return nil, fmt.Errorf("failed to get player by external ID: %w", err)
	}
	return player, nil
}

// DeletePlayer deletes a player by ID
func (a *App) DeletePlayer(ctx context.Context, id uuid.UUID) error {
	// Verify player exists
	_, err := a.repo.GetPlayer(ctx, id)
	if err != nil {
		return fmt.Errorf("player not found: %w", err)
	}

	if err := a.repo.DeletePlayer(ctx, id); err != nil {
		return fmt.Errorf("failed to delete player: %w", err)
	}

	return nil
}

// validateCreatePlayerRequest validates create player request
func (a *App) validateCreatePlayerRequest(req CreatePlayerRequest) error {
	if req.SportID == "" {
		return fmt.Errorf("sport_id is required")
	}
	if req.ExternalID == "" {
		return fmt.Errorf("external_id is required")
	}
	if req.FullName == "" {
		return fmt.Errorf("full_name is required")
	}
	return nil
}

// SyncPlayersFromAPI synchronizes players from external API for a specific team
func (a *App) SyncPlayersFromAPI(ctx context.Context, teamAlias string, sportId string) (*SyncResult, error) {
	result := &SyncResult{}

	// Fetch team for team id
	team, err := a.teamApp.GetTeamBySportIdAndCode(ctx, sportId, teamAlias)
	if err != nil {
		return nil, fmt.Errorf("failed to get team: %w", err)
	}

	// Get the NFL plugin (assuming NFL for now)
	plugin, exists := a.plugins[sportId]
	if !exists {
		return nil, fmt.Errorf("no plugin found for sport: nfl")
	}

	// Fetch players from the plugin for a specific team
	players, err := plugin.FetchPlayers(ctx, teamAlias)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch players from plugin: %w", err)
	}

	result.TotalProcessed = len(players)

	for _, player := range players {
		isNew, err := a.upsertPlayerFromPlugin(ctx, plugin, player, team.ID)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("failed to upsert player %s: %w", player.Name, err))
			continue
		}

		if isNew {
			result.Created++
		} else {
			result.Updated++
		}
	}

	return result, nil
}

// SyncAllNFLPlayersFromAPI synchronizes all NFL players from external API
// This method should be called with specific team aliases instead
func (a *App) SyncAllNFLPlayersFromAPI(ctx context.Context) (*SyncResult, error) {
	return nil, fmt.Errorf("use SyncPlayersFromAPI(ctx, teamAlias) for specific teams instead")
}

// upsertPlayerFromPlugin performs an upsert operation for a player from plugin data
// Returns true if player was created (new), false if updated
func (a *App) upsertPlayerFromPlugin(ctx context.Context, plugin base.SportPlugin, srPlayer sportradarclient.SRPlayer, teamId uuid.UUID) (bool, error) {
	// Map player data using the plugin (includes attached profile)
	player, err := plugin.MapExternalPlayer(srPlayer)
	if err != nil {
		return false, fmt.Errorf("failed to map external player: %w", err)
	}

	player.TeamID = &teamId

	// Check if player already exists
	existingPlayer, err := a.repo.GetPlayerByExternalID(ctx, player.SportID, player.ExternalID)
	if err != nil {
		// Player doesn't exist, create it
		createReq := CreatePlayerRequest{
			SportID:    player.SportID,
			ExternalID: player.ExternalID,
			FullName:   player.FullName,
			TeamID:     player.TeamID,
			Profile:    player.NFLPlayerProfile,
		}
		_, err := a.repo.CreatePlayer(ctx, createReq)
		if err != nil {
			return false, fmt.Errorf("failed to create player: %w", err)
		}
		return true, nil // Created new player
	}

	// Player exists, update it
	_, err = a.repo.UpdatePlayerAndProfile(ctx, existingPlayer.ID, player.FullName, player.TeamID, player.NFLPlayerProfile)
	if err != nil {
		return false, fmt.Errorf("failed to update player: %w", err)
	}

	return false, nil // Updated existing player
}
