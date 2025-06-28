package roster

import (
	"context"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/mcdev12/dynasty/go/internal/models"
)

// RosterRepository defines what the app layer needs from the repository
type RosterRepository interface {
	CreateRosterPlayer(ctx context.Context, req CreateRosterPlayerRequest) (*models.Roster, error)
	GetRoster(ctx context.Context, id uuid.UUID) (*models.Roster, error)
	GetRosterPlayersByFantasyTeam(ctx context.Context, fantasyTeamID uuid.UUID) ([]models.Roster, error)
	GetRosterPlayersByFantasyTeamAndPosition(ctx context.Context, fantasyTeamID uuid.UUID, position models.RosterPosition) ([]models.Roster, error)
	GetPlayerOnRoster(ctx context.Context, fantasyTeamID, playerID uuid.UUID) (*models.Roster, error)
	GetStartingRosterPlayers(ctx context.Context, fantasyTeamID uuid.UUID) ([]models.Roster, error)
	GetBenchRosterPlayers(ctx context.Context, fantasyTeamID uuid.UUID) ([]models.Roster, error)
	GetRosterPlayersByAcquisitionType(ctx context.Context, fantasyTeamID uuid.UUID, acquisitionType models.AcquisitionType) ([]models.Roster, error)
	UpdateRosterPlayerPosition(ctx context.Context, id uuid.UUID, req UpdateRosterPositionRequest) (*models.Roster, error)
	UpdateRosterPlayerKeeperData(ctx context.Context, id uuid.UUID, req UpdateRosterKeeperDataRequest) (*models.Roster, error)
	UpdateRosterPositionAndKeeperData(ctx context.Context, id uuid.UUID, req UpdateRosterPositionAndKeeperDataRequest) (*models.Roster, error)
	DeleteRosterEntry(ctx context.Context, id uuid.UUID) error
	DeletePlayerFromRoster(ctx context.Context, fantasyTeamID, playerID uuid.UUID) error
	DeleteTeamRoster(ctx context.Context, fantasyTeamID uuid.UUID) error
}

// FantasyTeamsRepository defines what the app layer needs from the fantasy teams repository for validation
type FantasyTeamsRepository interface {
	GetFantasyTeam(ctx context.Context, id uuid.UUID) (*models.FantasyTeam, error)
}

// PlayersRepository defines what the app layer needs from the players repository for validation
type PlayersRepository interface {
	GetPlayer(ctx context.Context, id uuid.UUID) (*models.Player, error)
}

// App handles roster business logic
type App struct {
	repo             RosterRepository
	fantasyTeamsRepo FantasyTeamsRepository
	playersRepo      PlayersRepository
}

// NewApp creates a new roster App
func NewApp(repo RosterRepository, fantasyTeamsRepo FantasyTeamsRepository, playersRepo PlayersRepository) *App {
	return &App{
		repo:             repo,
		fantasyTeamsRepo: fantasyTeamsRepo,
		playersRepo:      playersRepo,
	}
}

// CreateRosterPlayer adds a player to a fantasy team's roster with validation
func (a *App) CreateRosterPlayer(ctx context.Context, req CreateRosterPlayerRequest) (*models.Roster, error) {
	if err := a.validateCreateRosterPlayerRequest(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Verify fantasy team exists
	_, err := a.fantasyTeamsRepo.GetFantasyTeam(ctx, req.FantasyTeamID)
	if err != nil {
		return nil, fmt.Errorf("fantasy team not found: %w", err)
	}

	// Verify player exists
	_, err = a.playersRepo.GetPlayer(ctx, req.PlayerID)
	if err != nil {
		return nil, fmt.Errorf("player not found: %w", err)
	}

	// Check if player is already on this team's roster
	existingRoster, err := a.repo.GetPlayerOnRoster(ctx, req.FantasyTeamID, req.PlayerID)
	if err == nil && existingRoster != nil {
		return nil, fmt.Errorf("player is already on this team's roster")
	}

	roster, err := a.repo.CreateRosterPlayer(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create roster entry: %w", err)
	}

	log.Printf("Added player %s to team %s roster as %s", roster.PlayerID, roster.FantasyTeamID, roster.Position)
	return roster, nil
}

// GetRoster retrieves a roster entry by ID
func (a *App) GetRoster(ctx context.Context, id uuid.UUID) (*models.Roster, error) {
	roster, err := a.repo.GetRoster(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get roster entry: %w", err)
	}
	return roster, nil
}

// GetRosterPlayersByFantasyTeam retrieves all players on a team's roster
func (a *App) GetRosterPlayersByFantasyTeam(ctx context.Context, fantasyTeamID uuid.UUID) ([]models.Roster, error) {
	// Verify fantasy team exists
	_, err := a.fantasyTeamsRepo.GetFantasyTeam(ctx, fantasyTeamID)
	if err != nil {
		return nil, fmt.Errorf("fantasy team not found: %w", err)
	}

	rosters, err := a.repo.GetRosterPlayersByFantasyTeam(ctx, fantasyTeamID)
	if err != nil {
		return nil, fmt.Errorf("failed to get roster players by fantasy team: %w", err)
	}
	return rosters, nil
}

// GetRosterPlayersByFantasyTeamAndPosition retrieves players by team and position
func (a *App) GetRosterPlayersByFantasyTeamAndPosition(ctx context.Context, fantasyTeamID uuid.UUID, position models.RosterPosition) ([]models.Roster, error) {
	if err := a.validateRosterPosition(position); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Verify fantasy team exists
	_, err := a.fantasyTeamsRepo.GetFantasyTeam(ctx, fantasyTeamID)
	if err != nil {
		return nil, fmt.Errorf("fantasy team not found: %w", err)
	}

	rosters, err := a.repo.GetRosterPlayersByFantasyTeamAndPosition(ctx, fantasyTeamID, position)
	if err != nil {
		return nil, fmt.Errorf("failed to get roster players by team and position: %w", err)
	}
	return rosters, nil
}

// GetPlayerOnRoster checks if a specific player is on a team's roster
func (a *App) GetPlayerOnRoster(ctx context.Context, fantasyTeamID, playerID uuid.UUID) (*models.Roster, error) {
	// Verify fantasy team exists
	_, err := a.fantasyTeamsRepo.GetFantasyTeam(ctx, fantasyTeamID)
	if err != nil {
		return nil, fmt.Errorf("fantasy team not found: %w", err)
	}

	// Verify player exists
	_, err = a.playersRepo.GetPlayer(ctx, playerID)
	if err != nil {
		return nil, fmt.Errorf("player not found: %w", err)
	}

	roster, err := a.repo.GetPlayerOnRoster(ctx, fantasyTeamID, playerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get player on roster: %w", err)
	}
	return roster, nil
}

// GetStartingRosterPlayers retrieves all starting players for a team
func (a *App) GetStartingRosterPlayers(ctx context.Context, fantasyTeamID uuid.UUID) ([]models.Roster, error) {
	// Verify fantasy team exists
	_, err := a.fantasyTeamsRepo.GetFantasyTeam(ctx, fantasyTeamID)
	if err != nil {
		return nil, fmt.Errorf("fantasy team not found: %w", err)
	}

	rosters, err := a.repo.GetStartingRosterPlayers(ctx, fantasyTeamID)
	if err != nil {
		return nil, fmt.Errorf("failed to get starting roster players: %w", err)
	}
	return rosters, nil
}

// GetBenchRosterPlayers retrieves all bench players for a team
func (a *App) GetBenchRosterPlayers(ctx context.Context, fantasyTeamID uuid.UUID) ([]models.Roster, error) {
	// Verify fantasy team exists
	_, err := a.fantasyTeamsRepo.GetFantasyTeam(ctx, fantasyTeamID)
	if err != nil {
		return nil, fmt.Errorf("fantasy team not found: %w", err)
	}

	rosters, err := a.repo.GetBenchRosterPlayers(ctx, fantasyTeamID)
	if err != nil {
		return nil, fmt.Errorf("failed to get bench roster players: %w", err)
	}
	return rosters, nil
}

// GetRosterPlayersByAcquisitionType retrieves players by how they were acquired
func (a *App) GetRosterPlayersByAcquisitionType(ctx context.Context, fantasyTeamID uuid.UUID, acquisitionType models.AcquisitionType) ([]models.Roster, error) {
	if err := a.validateAcquisitionType(acquisitionType); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Verify fantasy team exists
	_, err := a.fantasyTeamsRepo.GetFantasyTeam(ctx, fantasyTeamID)
	if err != nil {
		return nil, fmt.Errorf("fantasy team not found: %w", err)
	}

	rosters, err := a.repo.GetRosterPlayersByAcquisitionType(ctx, fantasyTeamID, acquisitionType)
	if err != nil {
		return nil, fmt.Errorf("failed to get roster players by acquisition type: %w", err)
	}
	return rosters, nil
}

// UpdateRosterPlayerPosition updates a player's position on the roster
func (a *App) UpdateRosterPlayerPosition(ctx context.Context, id uuid.UUID, req UpdateRosterPositionRequest) (*models.Roster, error) {
	if err := a.validateUpdateRosterPositionRequest(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Verify roster entry exists
	_, err := a.repo.GetRoster(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("roster entry not found: %w", err)
	}

	roster, err := a.repo.UpdateRosterPlayerPosition(ctx, id, req)
	if err != nil {
		return nil, fmt.Errorf("failed to update roster player position: %w", err)
	}

	log.Printf("Updated roster player position: %s moved to %s", roster.PlayerID, roster.Position)
	return roster, nil
}

// UpdateRosterPlayerKeeperData updates a player's keeper data
func (a *App) UpdateRosterPlayerKeeperData(ctx context.Context, id uuid.UUID, req UpdateRosterKeeperDataRequest) (*models.Roster, error) {
	// Verify roster entry exists
	_, err := a.repo.GetRoster(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("roster entry not found: %w", err)
	}

	roster, err := a.repo.UpdateRosterPlayerKeeperData(ctx, id, req)
	if err != nil {
		return nil, fmt.Errorf("failed to update roster player keeper data: %w", err)
	}

	log.Printf("Updated roster player keeper data: %s", roster.PlayerID)
	return roster, nil
}

// UpdateRosterPositionAndKeeperData updates both position and keeper data
func (a *App) UpdateRosterPositionAndKeeperData(ctx context.Context, id uuid.UUID, req UpdateRosterPositionAndKeeperDataRequest) (*models.Roster, error) {
	if err := a.validateUpdateRosterPositionAndKeeperDataRequest(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Verify roster entry exists
	_, err := a.repo.GetRoster(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("roster entry not found: %w", err)
	}

	roster, err := a.repo.UpdateRosterPositionAndKeeperData(ctx, id, req)
	if err != nil {
		return nil, fmt.Errorf("failed to update roster position and keeper data: %w", err)
	}

	log.Printf("Updated roster player position and keeper data: %s moved to %s", roster.PlayerID, roster.Position)
	return roster, nil
}

// DeleteRosterEntry removes a specific roster entry
func (a *App) DeleteRosterEntry(ctx context.Context, id uuid.UUID) error {
	// Verify roster entry exists
	roster, err := a.repo.GetRoster(ctx, id)
	if err != nil {
		return fmt.Errorf("roster entry not found: %w", err)
	}

	if err := a.repo.DeleteRosterEntry(ctx, id); err != nil {
		return fmt.Errorf("failed to delete roster entry: %w", err)
	}

	log.Printf("Deleted roster entry: player %s from team %s", roster.PlayerID, roster.FantasyTeamID)
	return nil
}

// DeletePlayerFromRoster removes a player from a team's roster
func (a *App) DeletePlayerFromRoster(ctx context.Context, fantasyTeamID, playerID uuid.UUID) error {
	// Verify fantasy team exists
	_, err := a.fantasyTeamsRepo.GetFantasyTeam(ctx, fantasyTeamID)
	if err != nil {
		return fmt.Errorf("fantasy team not found: %w", err)
	}

	// Verify player exists on roster
	_, err = a.repo.GetPlayerOnRoster(ctx, fantasyTeamID, playerID)
	if err != nil {
		return fmt.Errorf("player not found on roster: %w", err)
	}

	if err := a.repo.DeletePlayerFromRoster(ctx, fantasyTeamID, playerID); err != nil {
		return fmt.Errorf("failed to delete player from roster: %w", err)
	}

	log.Printf("Deleted player %s from team %s roster", playerID, fantasyTeamID)
	return nil
}

// DeleteTeamRoster clears an entire team's roster
func (a *App) DeleteTeamRoster(ctx context.Context, fantasyTeamID uuid.UUID) error {
	// Verify fantasy team exists
	team, err := a.fantasyTeamsRepo.GetFantasyTeam(ctx, fantasyTeamID)
	if err != nil {
		return fmt.Errorf("fantasy team not found: %w", err)
	}

	if err := a.repo.DeleteTeamRoster(ctx, fantasyTeamID); err != nil {
		return fmt.Errorf("failed to delete team roster: %w", err)
	}

	log.Printf("Deleted entire roster for team: %s", team.Name)
	return nil
}

// Validation methods

func (a *App) validateCreateRosterPlayerRequest(req CreateRosterPlayerRequest) error {
	if req.FantasyTeamID == uuid.Nil {
		return fmt.Errorf("fantasy_team_id is required")
	}
	if req.PlayerID == uuid.Nil {
		return fmt.Errorf("player_id is required")
	}
	if err := a.validateRosterPosition(req.Position); err != nil {
		return err
	}
	if err := a.validateAcquisitionType(req.AcquisitionType); err != nil {
		return err
	}
	return nil
}

func (a *App) validateUpdateRosterPositionRequest(req UpdateRosterPositionRequest) error {
	return a.validateRosterPosition(req.Position)
}

func (a *App) validateUpdateRosterPositionAndKeeperDataRequest(req UpdateRosterPositionAndKeeperDataRequest) error {
	return a.validateRosterPosition(req.Position)
}

func (a *App) validateTransferPlayerRequest(req TransferPlayerRequest) error {
	if req.FantasyTeamID == uuid.Nil {
		return fmt.Errorf("fantasy_team_id is required")
	}
	if err := a.validateAcquisitionType(req.AcquisitionType); err != nil {
		return err
	}
	return nil
}

func (a *App) validateRosterPosition(position models.RosterPosition) error {
	switch position {
	case models.RosterPositionStarter, models.RosterPositionBench, models.RosterPositionIR, models.RosterPositionTaxi:
		return nil
	default:
		return fmt.Errorf("invalid roster position: %s", position)
	}
}

func (a *App) validateAcquisitionType(acquisitionType models.AcquisitionType) error {
	switch acquisitionType {
	case models.AcquisitionTypeDraft, models.AcquisitionTypeWaiver, models.AcquisitionTypeTrade,
		models.AcquisitionTypeFreeAgent, models.AcquisitionTypeKeeper:
		return nil
	default:
		return fmt.Errorf("invalid acquisition type: %s", acquisitionType)
	}
}
