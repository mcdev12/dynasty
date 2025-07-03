package leagues

import (
	"context"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/mcdev12/dynasty/go/internal/models"
)

// LeaguesRepository defines what the app layer needs from the repository
type LeaguesRepository interface {
	CreateLeague(ctx context.Context, req CreateLeagueRequest) (*models.League, error)
	GetLeague(ctx context.Context, id uuid.UUID) (*models.League, error)
	GetLeaguesByCommissioner(ctx context.Context, commissionerID uuid.UUID) ([]models.League, error)
	UpdateLeague(ctx context.Context, id uuid.UUID, req UpdateLeagueRequest) (*models.League, error)
	UpdateLeagueStatus(ctx context.Context, id uuid.UUID, status models.LeagueStatus) (*models.League, error)
	UpdateLeagueSettings(ctx context.Context, id uuid.UUID, settings interface{}) (*models.League, error)
	DeleteLeague(ctx context.Context, id uuid.UUID) error
}

// App handles leagues business logic
type App struct {
	repo LeaguesRepository
}

// NewApp creates a new leagues App
func NewApp(repo LeaguesRepository) *App {
	return &App{
		repo: repo,
	}
}

// CreateLeague creates a new league with validation
func (a *App) CreateLeague(ctx context.Context, req CreateLeagueRequest) (*models.League, error) {
	if err := a.validateCreateLeagueRequest(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	league, err := a.repo.CreateLeague(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create league: %w", err)
	}

	log.Printf("Created league: %s (%s) for sport %s", league.Name, league.LeagueType, league.SportID)
	return league, nil
}

// GetLeague retrieves a league by ID
func (a *App) GetLeague(ctx context.Context, id uuid.UUID) (*models.League, error) {
	league, err := a.repo.GetLeague(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get league: %w", err)
	}
	return league, nil
}

// GetLeaguesByCommissioner retrieves leagues by commissioner ID
func (a *App) GetLeaguesByCommissioner(ctx context.Context, commissionerID uuid.UUID) ([]models.League, error) {
	leagues, err := a.repo.GetLeaguesByCommissioner(ctx, commissionerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get leagues by commissioner: %w", err)
	}
	return leagues, nil
}

// UpdateLeague updates an existing league with validation
func (a *App) UpdateLeague(ctx context.Context, id uuid.UUID, req UpdateLeagueRequest) (*models.League, error) {
	if err := a.validateUpdateLeagueRequest(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Verify league exists
	_, err := a.repo.GetLeague(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("league not found: %w", err)
	}

	league, err := a.repo.UpdateLeague(ctx, id, req)
	if err != nil {
		return nil, fmt.Errorf("failed to update league: %w", err)
	}

	log.Printf("Updated league: %s (%s)", league.Name, league.LeagueType)
	return league, nil
}

// UpdateLeagueStatus updates only the status of a league
func (a *App) UpdateLeagueStatus(ctx context.Context, id uuid.UUID, status models.LeagueStatus) (*models.League, error) {
	if err := a.validateLeagueStatus(status); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Verify league exists
	_, err := a.repo.GetLeague(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("league not found: %w", err)
	}

	league, err := a.repo.UpdateLeagueStatus(ctx, id, status)
	if err != nil {
		return nil, fmt.Errorf("failed to update league status: %w", err)
	}

	log.Printf("Updated league status: %s -> %s", league.Name, status)
	return league, nil
}

// UpdateLeagueSettings updates only the settings of a league
func (a *App) UpdateLeagueSettings(ctx context.Context, id uuid.UUID, settings interface{}) (*models.League, error) {
	if settings == nil {
		return nil, fmt.Errorf("league settings cannot be nil")
	}

	// Verify league exists
	_, err := a.repo.GetLeague(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("league not found: %w", err)
	}

	league, err := a.repo.UpdateLeagueSettings(ctx, id, settings)
	if err != nil {
		return nil, fmt.Errorf("failed to update league settings: %w", err)
	}

	log.Printf("Updated league settings: %s", league.Name)
	return league, nil
}

// DeleteLeague deletes a league by ID
func (a *App) DeleteLeague(ctx context.Context, id uuid.UUID) error {
	// Verify league exists
	league, err := a.repo.GetLeague(ctx, id)
	if err != nil {
		return fmt.Errorf("league not found: %w", err)
	}

	if err := a.repo.DeleteLeague(ctx, id); err != nil {
		return fmt.Errorf("failed to delete league: %w", err)
	}

	log.Printf("Deleted league: %s (%s)", league.Name, league.LeagueType)
	return nil
}

// validateCreateLeagueRequest validates create league request
func (a *App) validateCreateLeagueRequest(req CreateLeagueRequest) error {
	if req.Name == "" {
		return fmt.Errorf("name is required")
	}
	if req.SportID == "" {
		return fmt.Errorf("sport_id is required")
	}
	if req.LeagueType == "" {
		return fmt.Errorf("league_type is required")
	}
	if err := a.validateLeagueType(req.LeagueType); err != nil {
		return err
	}
	if req.CommissionerID == uuid.Nil {
		return fmt.Errorf("commissioner_id is required")
	}
	if req.LeagueSettings == nil {
		return fmt.Errorf("league_settings is required")
	}
	if req.Status == "" {
		return fmt.Errorf("status is required")
	}
	if err := a.validateLeagueStatus(req.Status); err != nil {
		return err
	}
	if req.Season == "" {
		return fmt.Errorf("season is required")
	}
	return nil
}

// validateUpdateLeagueRequest validates update league request
func (a *App) validateUpdateLeagueRequest(req UpdateLeagueRequest) error {
	if req.Name == "" {
		return fmt.Errorf("name cannot be empty")
	}
	if req.SportID == "" {
		return fmt.Errorf("sport_id cannot be empty")
	}
	if req.LeagueType == "" {
		return fmt.Errorf("league_type cannot be empty")
	}
	if err := a.validateLeagueType(req.LeagueType); err != nil {
		return err
	}
	if req.CommissionerID == uuid.Nil {
		return fmt.Errorf("commissioner_id cannot be empty")
	}
	if req.LeagueSettings == nil {
		return fmt.Errorf("league_settings cannot be nil")
	}
	if req.Status == "" {
		return fmt.Errorf("status cannot be empty")
	}
	if err := a.validateLeagueStatus(req.Status); err != nil {
		return err
	}
	if req.Season == "" {
		return fmt.Errorf("season cannot be empty")
	}
	return nil
}

// validateLeagueType validates league type
func (a *App) validateLeagueType(leagueType models.LeagueType) error {
	switch leagueType {
	case models.LeagueTypeRedraft, models.LeagueTypeKeeper, models.LeagueTypeDynasty:
		return nil
	default:
		return fmt.Errorf("invalid league type: %s", leagueType)
	}
}

// validateLeagueStatus validates league status
func (a *App) validateLeagueStatus(status models.LeagueStatus) error {
	switch status {
	case models.LeagueStatusPending, models.LeagueStatusCancelled, models.LeagueStatusActive, models.LeagueStatusCompleted:
		return nil
	default:
		return fmt.Errorf("invalid league status: %s", status)
	}
}
