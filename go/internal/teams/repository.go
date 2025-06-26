package teams

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/mcdev12/dynasty/go/internal/models"
	"github.com/mcdev12/dynasty/go/internal/sqlutil"
	"github.com/mcdev12/dynasty/go/internal/teams/db"
)

// Querier defines what the repository needs from the database layer
type Querier interface {
	CreateTeam(ctx context.Context, arg db.CreateTeamParams) (db.Team, error)
	GetTeam(ctx context.Context, id uuid.UUID) (db.Team, error)
	GetTeamByExternalID(ctx context.Context, arg db.GetTeamByExternalIDParams) (db.Team, error)
	ListTeamsBySport(ctx context.Context, sportID string) ([]db.Team, error)
	ListAllTeams(ctx context.Context) ([]db.Team, error)
	UpdateTeam(ctx context.Context, arg db.UpdateTeamParams) (db.Team, error)
	DeleteTeam(ctx context.Context, id uuid.UUID) error
}

// Repository implements team data access operations
type Repository struct {
	queries Querier
}

// NewRepository creates a new teams repository
func NewRepository(querier Querier) *Repository {
	return &Repository{
		queries: querier,
	}
}

// CreateTeam creates a new team
func (r *Repository) CreateTeam(ctx context.Context, req CreateTeamRequest) (*models.Team, error) {
	params := r.createTeamRequestToParams(req)

	dbTeam, err := r.queries.CreateTeam(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create team: %w", err)
	}

	return r.dbTeamToModel(dbTeam), nil
}

// GetTeam retrieves a team by ID
func (r *Repository) GetTeam(ctx context.Context, id uuid.UUID) (*models.Team, error) {
	dbTeam, err := r.queries.GetTeam(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get team: %w", err)
	}

	return r.dbTeamToModel(dbTeam), nil
}

// GetTeamByExternalID retrieves a team by sport ID and external ID
func (r *Repository) GetTeamByExternalID(ctx context.Context, sportID, externalID string) (*models.Team, error) {
	params := db.GetTeamByExternalIDParams{
		SportID:    sportID,
		ExternalID: externalID,
	}

	dbTeam, err := r.queries.GetTeamByExternalID(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get team by external ID: %w", err)
	}

	return r.dbTeamToModel(dbTeam), nil
}

// ListTeamsBySport retrieves all teams for a specific sport
func (r *Repository) ListTeamsBySport(ctx context.Context, sportID string) ([]models.Team, error) {
	dbTeams, err := r.queries.ListTeamsBySport(ctx, sportID)
	if err != nil {
		return nil, fmt.Errorf("failed to list teams by sport: %w", err)
	}

	teams := make([]models.Team, len(dbTeams))
	for i, dbTeam := range dbTeams {
		teams[i] = *r.dbTeamToModel(dbTeam)
	}

	return teams, nil
}

// ListAllTeams retrieves all teams
func (r *Repository) ListAllTeams(ctx context.Context) ([]models.Team, error) {
	dbTeams, err := r.queries.ListAllTeams(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list all teams: %w", err)
	}

	teams := make([]models.Team, len(dbTeams))
	for i, dbTeam := range dbTeams {
		teams[i] = *r.dbTeamToModel(dbTeam)
	}

	return teams, nil
}

// UpdateTeam updates an existing team
func (r *Repository) UpdateTeam(ctx context.Context, id uuid.UUID, req UpdateTeamRequest) (*models.Team, error) {
	params := r.updateTeamRequestToParams(id, req)

	dbTeam, err := r.queries.UpdateTeam(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to update team: %w", err)
	}

	return r.dbTeamToModel(dbTeam), nil
}

// DeleteTeam deletes a team by ID
func (r *Repository) DeleteTeam(ctx context.Context, id uuid.UUID) error {
	if err := r.queries.DeleteTeam(ctx, id); err != nil {
		return fmt.Errorf("failed to delete team: %w", err)
	}

	return nil
}

// createTeamRequestToParams converts CreateTeamRequest to sqlc params
func (r *Repository) createTeamRequestToParams(req CreateTeamRequest) db.CreateTeamParams {
	return db.CreateTeamParams{
		SportID:         req.SportID,
		ExternalID:      req.ExternalID,
		Name:            req.Name,
		Code:            req.Code,
		City:            req.City,
		Coach:           sqlutil.ToSqlString(req.Coach),
		Owner:           sqlutil.ToSqlString(req.Owner),
		Stadium:         sqlutil.ToSqlString(req.Stadium),
		EstablishedYear: sqlutil.ToSqlInt32(req.EstablishedYear),
	}
}

// updateTeamRequestToParams converts UpdateTeamRequest to sqlc params
func (r *Repository) updateTeamRequestToParams(id uuid.UUID, req UpdateTeamRequest) db.UpdateTeamParams {
	params := db.UpdateTeamParams{
		ID: id,
	}

	if req.Name != nil {
		params.Name = *req.Name
	}

	if req.Code != nil {
		params.Code = *req.Code
	}

	if req.City != nil {
		params.City = *req.City
	}

	if req.Coach != nil {
		params.Coach = sqlutil.ToSqlString(req.Coach)
	}

	if req.Owner != nil {
		params.Owner = sqlutil.ToSqlString(req.Owner)
	}

	if req.Stadium != nil {
		params.Stadium = sqlutil.ToSqlString(req.Stadium)
	}

	if req.EstablishedYear != nil {
		params.EstablishedYear = sqlutil.ToSqlInt32(req.EstablishedYear)
	}

	return params
}

// dbTeamToModel converts a database team to domain model
func (r *Repository) dbTeamToModel(dbTeam db.Team) *models.Team {
	return &models.Team{
		ID:              dbTeam.ID,
		SportID:         dbTeam.SportID,
		ExternalID:      dbTeam.ExternalID,
		Name:            dbTeam.Name,
		Code:            dbTeam.Code,
		City:            dbTeam.City,
		Coach:           sqlutil.FromSqlStringPtr(dbTeam.Coach),
		Owner:           sqlutil.FromSqlStringPtr(dbTeam.Owner),
		Stadium:         sqlutil.FromSqlStringPtr(dbTeam.Stadium),
		EstablishedYear: sqlutil.FromSqlInt32(dbTeam.EstablishedYear),
		CreatedAt:       dbTeam.CreatedAt,
	}
}
