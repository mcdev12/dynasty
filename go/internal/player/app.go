package player

import (
	"context"
	"fmt"
	"strconv"

	"github.com/google/uuid"
	sportradarclient "github.com/mcdev12/dynasty/go/clients/sport_radar_client"
	"github.com/mcdev12/dynasty/go/internal/models"
)

// PlayerRepository defines what the app layer needs from the repository
type PlayerRepository interface {
	CreatePlayer(ctx context.Context, req CreatePlayerRequest) (*models.Player, error)
	GetPlayer(ctx context.Context, id uuid.UUID) (*models.Player, error)
	GetPlayerByExternalID(ctx context.Context, sportID, externalID string) (*models.Player, error)
	UpdatePlayerAndProfile(ctx context.Context, playerID uuid.UUID, fullName string, teamID *uuid.UUID, profile models.Profile) (*models.Player, error)
	DeletePlayer(ctx context.Context, id uuid.UUID) error
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
	repo         PlayerRepository
	sportRadar   *sportradarclient.SportRadarClient
}

// NewApp creates a new player App
func NewApp(repo PlayerRepository, sportRadar *sportradarclient.SportRadarClient) *App {
	return &App{
		repo:       repo,
		sportRadar: sportRadar,
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

// SyncPlayersFromAPI synchronizes players from SportRadar API for a specific team
func (a *App) SyncPlayersFromAPI(ctx context.Context, teamAlias string) (*SyncResult, error) {
	result := &SyncResult{}

	// Fetch roster from SportRadar
	roster, err := a.sportRadar.GetTeamRosterByAlias(teamAlias)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch roster from SportRadar: %w", err)
	}

	result.TotalProcessed = len(roster.Players)

	for _, srPlayer := range roster.Players {
		isNew, err := a.upsertPlayer(ctx, srPlayer)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("failed to upsert player %s: %w", srPlayer.Name, err))
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

// SyncAllNFLPlayersFromAPI synchronizes all NFL players from SportRadar API
func (a *App) SyncAllNFLPlayersFromAPI(ctx context.Context) (*SyncResult, error) {
	result := &SyncResult{}

	// First, get all NFL teams
	teams, err := a.sportRadar.GetNFLTeams()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch NFL teams: %w", err)
	}

	// Sync players for each team
	for _, team := range teams {
		teamResult, err := a.SyncPlayersFromAPI(ctx, team.Alias)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("failed to sync players for team %s: %w", team.Alias, err))
			continue
		}

		// Aggregate results
		result.TotalProcessed += teamResult.TotalProcessed
		result.Created += teamResult.Created
		result.Updated += teamResult.Updated
		result.Errors = append(result.Errors, teamResult.Errors...)
	}

	return result, nil
}

// upsertPlayer performs an upsert operation for a player (create if not exists, update if exists)
// Returns true if player was created (new), false if updated
func (a *App) upsertPlayer(ctx context.Context, srPlayer sportradarclient.SRPlayer) (bool, error) {
	// Map SportRadar player to our domain model
	player, profile, err := a.mapSportRadarPlayer(srPlayer)
	if err != nil {
		return false, fmt.Errorf("failed to map SportRadar player: %w", err)
	}

	// Check if player already exists
	existingPlayer, err := a.repo.GetPlayerByExternalID(ctx, player.SportID, player.ExternalID)
	if err != nil {
		// Player doesn't exist, create it
		createReq := CreatePlayerRequest{
			SportID:    player.SportID,
			ExternalID: player.ExternalID,
			FullName:   player.FullName,
			TeamID:     player.TeamID,
			Profile:    profile,
		}
		_, err := a.repo.CreatePlayer(ctx, createReq)
		if err != nil {
			return false, fmt.Errorf("failed to create player: %w", err)
		}
		return true, nil // Created new player
	}

	// Player exists, update both player and profile
	_, err = a.repo.UpdatePlayerAndProfile(ctx, existingPlayer.ID, player.FullName, player.TeamID, profile)
	if err != nil {
		return false, fmt.Errorf("failed to update player: %w", err)
	}

	return false, nil // Updated existing player
}

// mapSportRadarPlayer converts SportRadar player to our domain models
func (a *App) mapSportRadarPlayer(srPlayer sportradarclient.SRPlayer) (*models.Player, models.Profile, error) {
	// Create base player model
	player := &models.Player{
		SportID:    "nfl",
		ExternalID: fmt.Sprintf("sr_%s", srPlayer.ID),
		FullName:   srPlayer.Name,
		// TeamID will be nil for now - we'd need to map the team
	}

	// Create NFL profile
	profile := &models.NFLPlayerProfile{
		Position:   srPlayer.Position,
		College:    stringPtr(srPlayer.College),
		Experience: srPlayer.Experience,
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

	return player, profile, nil
}

// Helper functions
func stringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func intPtr(i int) *int {
	return &i
}