package fantasyteam

import (
	"context"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/mcdev12/dynasty/go/internal/models"
)

// FantasyTeamRepository defines what the app layer needs from the repository
type FantasyTeamRepository interface {
	CreateFantasyTeam(ctx context.Context, req CreateFantasyTeamRequest) (*models.FantasyTeam, error)
	GetFantasyTeam(ctx context.Context, id uuid.UUID) (*models.FantasyTeam, error)
	GetFantasyTeamsByLeague(ctx context.Context, leagueID uuid.UUID) ([]models.FantasyTeam, error)
	GetFantasyTeamsByOwner(ctx context.Context, ownerID uuid.UUID) ([]models.FantasyTeam, error)
	GetFantasyTeamByLeagueAndOwner(ctx context.Context, ownerID, leagueID uuid.UUID) (*models.FantasyTeam, error)
	UpdateFantasyTeam(ctx context.Context, id uuid.UUID, req UpdateFantasyTeamRequest) (*models.FantasyTeam, error)
	DeleteFantasyTeam(ctx context.Context, id uuid.UUID) error
}

// App handles fantasy teams business logic
type App struct {
	repo FantasyTeamRepository
}

// NewApp creates a new fantasy teams App
func NewApp(repo FantasyTeamRepository) *App {
	return &App{
		repo: repo,
	}
}

// CreateFantasyTeam creates a new fantasy team with validation
func (a *App) CreateFantasyTeam(ctx context.Context, req CreateFantasyTeamRequest) (*models.FantasyTeam, error) {
	if err := a.validateCreateFantasyTeamRequest(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Check if owner already has a team in this league
	existingTeam, err := a.repo.GetFantasyTeamByLeagueAndOwner(ctx, req.OwnerID, req.LeagueID)
	if err == nil && existingTeam != nil {
		return nil, fmt.Errorf("owner already has a team in this league")
	}

	team, err := a.repo.CreateFantasyTeam(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create fantasy team: %w", err)
	}

	log.Printf("Created fantasy team: %s for owner %s in league %s", team.Name, team.OwnerID, team.LeagueID)
	return team, nil
}

// GetFantasyTeam retrieves a fantasy team by ID
func (a *App) GetFantasyTeam(ctx context.Context, id uuid.UUID) (*models.FantasyTeam, error) {
	team, err := a.repo.GetFantasyTeam(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get fantasy team: %w", err)
	}
	return team, nil
}

// GetFantasyTeamsByLeague retrieves fantasy teams by league ID
func (a *App) GetFantasyTeamsByLeague(ctx context.Context, leagueID uuid.UUID) ([]models.FantasyTeam, error) {
	teams, err := a.repo.GetFantasyTeamsByLeague(ctx, leagueID)
	if err != nil {
		return nil, fmt.Errorf("failed to get fantasy teams by league: %w", err)
	}
	return teams, nil
}

// GetFantasyTeamsByOwner retrieves fantasy teams by owner ID
func (a *App) GetFantasyTeamsByOwner(ctx context.Context, ownerID uuid.UUID) ([]models.FantasyTeam, error) {
	teams, err := a.repo.GetFantasyTeamsByOwner(ctx, ownerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get fantasy teams by owner: %w", err)
	}
	return teams, nil
}

// GetFantasyTeamByLeagueAndOwner retrieves a fantasy team by league and owner
func (a *App) GetFantasyTeamByLeagueAndOwner(ctx context.Context, ownerID, leagueID uuid.UUID) (*models.FantasyTeam, error) {
	team, err := a.repo.GetFantasyTeamByLeagueAndOwner(ctx, ownerID, leagueID)
	if err != nil {
		return nil, fmt.Errorf("failed to get fantasy team by league and owner: %w", err)
	}
	return team, nil
}

// UpdateFantasyTeam updates an existing fantasy team with validation
func (a *App) UpdateFantasyTeam(ctx context.Context, id uuid.UUID, req UpdateFantasyTeamRequest) (*models.FantasyTeam, error) {
	if err := a.validateUpdateFantasyTeamRequest(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Verify team exists
	_, err := a.repo.GetFantasyTeam(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("fantasy team not found: %w", err)
	}

	team, err := a.repo.UpdateFantasyTeam(ctx, id, req)
	if err != nil {
		return nil, fmt.Errorf("failed to update fantasy team: %w", err)
	}

	log.Printf("Updated fantasy team: %s", team.Name)
	return team, nil
}

// DeleteFantasyTeam deletes a fantasy team by ID
func (a *App) DeleteFantasyTeam(ctx context.Context, id uuid.UUID) error {
	// Verify team exists
	team, err := a.repo.GetFantasyTeam(ctx, id)
	if err != nil {
		return fmt.Errorf("fantasy team not found: %w", err)
	}

	if err := a.repo.DeleteFantasyTeam(ctx, id); err != nil {
		return fmt.Errorf("failed to delete fantasy team: %w", err)
	}

	log.Printf("Deleted fantasy team: %s", team.Name)
	return nil
}

// validateCreateFantasyTeamRequest validates create fantasy team request
func (a *App) validateCreateFantasyTeamRequest(req CreateFantasyTeamRequest) error {
	if req.Name == "" {
		return fmt.Errorf("name is required")
	}
	if req.LeagueID == uuid.Nil {
		return fmt.Errorf("league_id is required")
	}
	if req.OwnerID == uuid.Nil {
		return fmt.Errorf("owner_id is required")
	}
	return nil
}

// validateUpdateFantasyTeamRequest validates update fantasy team request
func (a *App) validateUpdateFantasyTeamRequest(req UpdateFantasyTeamRequest) error {
	if req.Name == "" {
		return fmt.Errorf("name cannot be empty")
	}
	return nil
}
