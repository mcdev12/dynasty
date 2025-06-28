package roster

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/mcdev12/dynasty/go/internal/models"
	"github.com/mcdev12/dynasty/go/internal/roster/db"
	"github.com/sqlc-dev/pqtype"
)

type Querier interface {
	CreateRoster(ctx context.Context, arg db.CreateRosterParams) (db.Roster, error)
	DeletePlayerFromRoster(ctx context.Context, arg db.DeletePlayerFromRosterParams) error
	DeleteRosterEntry(ctx context.Context, id uuid.UUID) error
	DeleteTeamRoster(ctx context.Context, fantasyTeamID uuid.UUID) error
	GetBenchRosterPlayers(ctx context.Context, fantasyTeamID uuid.UUID) ([]db.Roster, error)
	GetPlayerOnRoster(ctx context.Context, arg db.GetPlayerOnRosterParams) (db.Roster, error)
	GetRoster(ctx context.Context, id uuid.UUID) (db.Roster, error)
	GetRosterPlayersByAcquisitionType(ctx context.Context, arg db.GetRosterPlayersByAcquisitionTypeParams) ([]db.Roster, error)
	GetRosterPlayersByFantasyTeam(ctx context.Context, fantasyTeamID uuid.UUID) ([]db.Roster, error)
	GetRosterPlayersByFantasyTeamAndPosition(ctx context.Context, arg db.GetRosterPlayersByFantasyTeamAndPositionParams) ([]db.Roster, error)
	GetStartingRosterPlayers(ctx context.Context, fantasyTeamID uuid.UUID) ([]db.Roster, error)
	UpdateRosterPlayerKeeperData(ctx context.Context, arg db.UpdateRosterPlayerKeeperDataParams) (db.Roster, error)
	UpdateRosterPlayerPosition(ctx context.Context, arg db.UpdateRosterPlayerPositionParams) (db.Roster, error)
	UpdateRosterPositionAndKeeperData(ctx context.Context, arg db.UpdateRosterPositionAndKeeperDataParams) (db.Roster, error)
}

type Repository struct {
	queries Querier
}

func NewRepository(querier Querier) *Repository {
	return &Repository{
		queries: querier,
	}
}

type CreateRosterRequest struct {
	FantasyTeamID   uuid.UUID              `json:"fantasy_team_id"`
	PlayerID        uuid.UUID              `json:"player_id"`
	Position        models.RosterPosition  `json:"position"`
	AcquisitionType models.AcquisitionType `json:"acquisition_type"`
	KeeperData      json.RawMessage        `json:"keeper_data"`
}

type UpdateRosterPositionRequest struct {
	Position models.RosterPosition `json:"position"`
}

type UpdateRosterKeeperDataRequest struct {
	KeeperData json.RawMessage `json:"keeper_data"`
}

type UpdateRosterPositionAndKeeperDataRequest struct {
	Position   models.RosterPosition `json:"position"`
	KeeperData json.RawMessage       `json:"keeper_data"`
}

type TransferPlayerRequest struct {
	FantasyTeamID   uuid.UUID              `json:"fantasy_team_id"`
	AcquisitionType models.AcquisitionType `json:"acquisition_type"`
	KeeperData      json.RawMessage        `json:"keeper_data"`
}

func (r *Repository) CreateRoster(ctx context.Context, req CreateRosterRequest) (*models.Roster, error) {
	roster, err := r.queries.CreateRoster(ctx, db.CreateRosterParams{
		FantasyTeamID:   req.FantasyTeamID,
		PlayerID:        req.PlayerID,
		Position:        db.RosterPositionEnum(req.Position),
		AcquisitionType: db.AcquisitionTypeEnum(req.AcquisitionType),
		KeeperData:      pqtype.NullRawMessage{RawMessage: req.KeeperData, Valid: len(req.KeeperData) > 0},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create roster entry: %w", err)
	}

	return r.dbRosterToModel(roster), nil
}

func (r *Repository) GetRoster(ctx context.Context, id uuid.UUID) (*models.Roster, error) {
	roster, err := r.queries.GetRoster(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get roster entry: %w", err)
	}

	return r.dbRosterToModel(roster), nil
}

func (r *Repository) GetRosterPlayersByFantasyTeam(ctx context.Context, fantasyTeamID uuid.UUID) ([]models.Roster, error) {
	rosters, err := r.queries.GetRosterPlayersByFantasyTeam(ctx, fantasyTeamID)
	if err != nil {
		return nil, fmt.Errorf("failed to get roster players by fantasy team: %w", err)
	}

	return r.dbRostersToModels(rosters), nil
}

func (r *Repository) GetRosterPlayersByFantasyTeamAndPosition(ctx context.Context, fantasyTeamID uuid.UUID, position models.RosterPosition) ([]models.Roster, error) {
	rosters, err := r.queries.GetRosterPlayersByFantasyTeamAndPosition(ctx, db.GetRosterPlayersByFantasyTeamAndPositionParams{
		FantasyTeamID: fantasyTeamID,
		Position:      db.RosterPositionEnum(position),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get roster players by team and position: %w", err)
	}

	return r.dbRostersToModels(rosters), nil
}

func (r *Repository) GetPlayerOnRoster(ctx context.Context, fantasyTeamID, playerID uuid.UUID) (*models.Roster, error) {
	roster, err := r.queries.GetPlayerOnRoster(ctx, db.GetPlayerOnRosterParams{
		FantasyTeamID: fantasyTeamID,
		PlayerID:      playerID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get player on roster: %w", err)
	}

	return r.dbRosterToModel(roster), nil
}

func (r *Repository) GetStartingRosterPlayers(ctx context.Context, fantasyTeamID uuid.UUID) ([]models.Roster, error) {
	rosters, err := r.queries.GetStartingRosterPlayers(ctx, fantasyTeamID)
	if err != nil {
		return nil, fmt.Errorf("failed to get starting roster players: %w", err)
	}

	return r.dbRostersToModels(rosters), nil
}

func (r *Repository) GetBenchRosterPlayers(ctx context.Context, fantasyTeamID uuid.UUID) ([]models.Roster, error) {
	rosters, err := r.queries.GetBenchRosterPlayers(ctx, fantasyTeamID)
	if err != nil {
		return nil, fmt.Errorf("failed to get bench roster players: %w", err)
	}

	return r.dbRostersToModels(rosters), nil
}

func (r *Repository) GetRosterPlayersByAcquisitionType(ctx context.Context, fantasyTeamID uuid.UUID, acquisitionType models.AcquisitionType) ([]models.Roster, error) {
	rosters, err := r.queries.GetRosterPlayersByAcquisitionType(ctx, db.GetRosterPlayersByAcquisitionTypeParams{
		FantasyTeamID:   fantasyTeamID,
		AcquisitionType: db.AcquisitionTypeEnum(acquisitionType),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get roster players by acquisition type: %w", err)
	}

	return r.dbRostersToModels(rosters), nil
}

func (r *Repository) UpdateRosterPlayerPosition(ctx context.Context, id uuid.UUID, req UpdateRosterPositionRequest) (*models.Roster, error) {
	roster, err := r.queries.UpdateRosterPlayerPosition(ctx, db.UpdateRosterPlayerPositionParams{
		ID:       id,
		Position: db.RosterPositionEnum(req.Position),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update roster player position: %w", err)
	}

	return r.dbRosterToModel(roster), nil
}

func (r *Repository) UpdateRosterPlayerKeeperData(ctx context.Context, id uuid.UUID, req UpdateRosterKeeperDataRequest) (*models.Roster, error) {
	roster, err := r.queries.UpdateRosterPlayerKeeperData(ctx, db.UpdateRosterPlayerKeeperDataParams{
		ID:         id,
		KeeperData: pqtype.NullRawMessage{RawMessage: req.KeeperData, Valid: len(req.KeeperData) > 0},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update roster player keeper data: %w", err)
	}

	return r.dbRosterToModel(roster), nil
}

func (r *Repository) UpdateRosterPositionAndKeeperData(ctx context.Context, id uuid.UUID, req UpdateRosterPositionAndKeeperDataRequest) (*models.Roster, error) {
	roster, err := r.queries.UpdateRosterPositionAndKeeperData(ctx, db.UpdateRosterPositionAndKeeperDataParams{
		ID:         id,
		Position:   db.RosterPositionEnum(req.Position),
		KeeperData: pqtype.NullRawMessage{RawMessage: req.KeeperData, Valid: len(req.KeeperData) > 0},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update roster position and keeper data: %w", err)
	}

	return r.dbRosterToModel(roster), nil
}

func (r *Repository) DeleteRosterEntry(ctx context.Context, id uuid.UUID) error {
	if err := r.queries.DeleteRosterEntry(ctx, id); err != nil {
		return fmt.Errorf("failed to delete roster entry: %w", err)
	}
	return nil
}

func (r *Repository) DeletePlayerFromRoster(ctx context.Context, fantasyTeamID, playerID uuid.UUID) error {
	if err := r.queries.DeletePlayerFromRoster(ctx, db.DeletePlayerFromRosterParams{
		FantasyTeamID: fantasyTeamID,
		PlayerID:      playerID,
	}); err != nil {
		return fmt.Errorf("failed to delete player from roster: %w", err)
	}
	return nil
}

func (r *Repository) DeleteTeamRoster(ctx context.Context, fantasyTeamID uuid.UUID) error {
	if err := r.queries.DeleteTeamRoster(ctx, fantasyTeamID); err != nil {
		return fmt.Errorf("failed to delete team roster: %w", err)
	}
	return nil
}

func (r *Repository) dbRosterToModel(dbRoster db.Roster) *models.Roster {
	var keeperData json.RawMessage
	if dbRoster.KeeperData.Valid {
		keeperData = dbRoster.KeeperData.RawMessage
	}

	return &models.Roster{
		ID:              dbRoster.ID,
		FantasyTeamID:   dbRoster.FantasyTeamID,
		PlayerID:        dbRoster.PlayerID,
		Position:        models.RosterPosition(dbRoster.Position),
		AcquiredAt:      dbRoster.AcquiredAt,
		AcquisitionType: models.AcquisitionType(dbRoster.AcquisitionType),
		KeeperData:      keeperData,
	}
}

func (r *Repository) dbRostersToModels(dbRosters []db.Roster) []models.Roster {
	rosters := make([]models.Roster, len(dbRosters))
	for i, dbRoster := range dbRosters {
		rosters[i] = *r.dbRosterToModel(dbRoster)
	}
	return rosters
}
