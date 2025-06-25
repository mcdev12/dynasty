package teams

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/mcdev12/dynasty/go/internal/teams/db"
)

// Querier defines what the repository needs from the database layer
type Querier interface {
	CreateTeam(ctx context.Context, arg db.CreateTeamParams) (db.Team, error)
	GetTeam(ctx context.Context, id pgtype.UUID) (db.Team, error)
	GetTeamByExternalID(ctx context.Context, arg db.GetTeamByExternalIDParams) (db.Team, error)
	ListTeamsBySport(ctx context.Context, sportID string) ([]db.Team, error)
	ListAllTeams(ctx context.Context) ([]db.Team, error)
	UpdateTeam(ctx context.Context, arg db.UpdateTeamParams) (db.Team, error)
	DeleteTeam(ctx context.Context, id pgtype.UUID) error
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
func (r *Repository) CreateTeam(ctx context.Context, req CreateTeamRequest) (*Team, error) {
	params := r.createTeamRequestToParams(req)

	dbTeam, err := r.queries.CreateTeam(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create team: %w", err)
	}

	return r.dbTeamToModel(dbTeam), nil
}

// GetTeam retrieves a team by ID
func (r *Repository) GetTeam(ctx context.Context, id uuid.UUID) (*Team, error) {
	pgUUID := pgtype.UUID{
		Bytes: id,
		Valid: true,
	}

	dbTeam, err := r.queries.GetTeam(ctx, pgUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get team: %w", err)
	}

	return r.dbTeamToModel(dbTeam), nil
}

// GetTeamByExternalID retrieves a team by sport ID and external ID
func (r *Repository) GetTeamByExternalID(ctx context.Context, sportID, externalID string) (*Team, error) {
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
func (r *Repository) ListTeamsBySport(ctx context.Context, sportID string) ([]Team, error) {
	dbTeams, err := r.queries.ListTeamsBySport(ctx, sportID)
	if err != nil {
		return nil, fmt.Errorf("failed to list teams by sport: %w", err)
	}

	teams := make([]Team, len(dbTeams))
	for i, dbTeam := range dbTeams {
		teams[i] = *r.dbTeamToModel(dbTeam)
	}

	return teams, nil
}

// ListAllTeams retrieves all teams
func (r *Repository) ListAllTeams(ctx context.Context) ([]Team, error) {
	dbTeams, err := r.queries.ListAllTeams(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list all teams: %w", err)
	}

	teams := make([]Team, len(dbTeams))
	for i, dbTeam := range dbTeams {
		teams[i] = *r.dbTeamToModel(dbTeam)
	}

	return teams, nil
}

// UpdateTeam updates an existing team
func (r *Repository) UpdateTeam(ctx context.Context, id uuid.UUID, req UpdateTeamRequest) (*Team, error) {
	params := r.updateTeamRequestToParams(id, req)

	dbTeam, err := r.queries.UpdateTeam(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to update team: %w", err)
	}

	return r.dbTeamToModel(dbTeam), nil
}

// DeleteTeam deletes a team by ID
func (r *Repository) DeleteTeam(ctx context.Context, id uuid.UUID) error {
	pgUUID := pgtype.UUID{
		Bytes: id,
		Valid: true,
	}

	if err := r.queries.DeleteTeam(ctx, pgUUID); err != nil {
		return fmt.Errorf("failed to delete team: %w", err)
	}

	return nil
}

// createTeamRequestToParams converts CreateTeamRequest to sqlc params
func (r *Repository) createTeamRequestToParams(req CreateTeamRequest) db.CreateTeamParams {
	params := db.CreateTeamParams{
		SportID:    req.SportID,
		ExternalID: req.ExternalID,
		Name:       req.Name,
		Code:       req.Code,
		City:       req.City,
	}

	if req.Coach != nil {
		params.Coach = pgtype.Text{String: *req.Coach, Valid: true}
	}

	if req.Owner != nil {
		params.Owner = pgtype.Text{String: *req.Owner, Valid: true}
	}

	if req.Stadium != nil {
		params.Stadium = pgtype.Text{String: *req.Stadium, Valid: true}
	}

	if req.EstablishedYear != nil {
		params.EstablishedYear = pgtype.Int4{Int32: int32(*req.EstablishedYear), Valid: true}
	}

	return params
}

// updateTeamRequestToParams converts UpdateTeamRequest to sqlc params
func (r *Repository) updateTeamRequestToParams(id uuid.UUID, req UpdateTeamRequest) db.UpdateTeamParams {
	params := db.UpdateTeamParams{
		ID: pgtype.UUID{Bytes: id, Valid: true},
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
		params.Coach = pgtype.Text{String: *req.Coach, Valid: true}
	}

	if req.Owner != nil {
		params.Owner = pgtype.Text{String: *req.Owner, Valid: true}
	}

	if req.Stadium != nil {
		params.Stadium = pgtype.Text{String: *req.Stadium, Valid: true}
	}

	if req.EstablishedYear != nil {
		params.EstablishedYear = pgtype.Int4{Int32: int32(*req.EstablishedYear), Valid: true}
	}

	return params
}

// dbTeamToModel converts a database team to domain model
func (r *Repository) dbTeamToModel(dbTeam db.Team) *Team {
	team := &Team{
		ID:         uuid.UUID(dbTeam.ID.Bytes),
		SportID:    dbTeam.SportID,
		ExternalID: dbTeam.ExternalID,
		Name:       dbTeam.Name,
		Code:       dbTeam.Code,
		City:       dbTeam.City,
		CreatedAt:  dbTeam.CreatedAt.Time,
	}

	if dbTeam.Coach.Valid {
		team.Coach = &dbTeam.Coach.String
	}

	if dbTeam.Owner.Valid {
		team.Owner = &dbTeam.Owner.String
	}

	if dbTeam.Stadium.Valid {
		team.Stadium = &dbTeam.Stadium.String
	}

	if dbTeam.EstablishedYear.Valid {
		year := int(dbTeam.EstablishedYear.Int32)
		team.EstablishedYear = &year
	}

	return team
}
