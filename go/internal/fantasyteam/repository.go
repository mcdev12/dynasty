package fantasyteam

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/mcdev12/dynasty/go/internal/fantasyteam/db"
	"github.com/mcdev12/dynasty/go/internal/models"
)

type Querier interface {
	CreateFantasyTeam(ctx context.Context, arg db.CreateFantasyTeamParams) (db.FantasyTeam, error)
	DeleteFantasyTeam(ctx context.Context, id uuid.UUID) error
	GetFantasyTeam(ctx context.Context, id uuid.UUID) (db.FantasyTeam, error)
	GetFantasyTeamByLeagueAndOwner(ctx context.Context, arg db.GetFantasyTeamByLeagueAndOwnerParams) (db.FantasyTeam, error)
	GetFantasyTeamsByLeague(ctx context.Context, leagueID uuid.UUID) ([]db.FantasyTeam, error)
	GetFantasyTeamsByOwner(ctx context.Context, ownerID uuid.UUID) ([]db.FantasyTeam, error)
	UpdateFantasyTeam(ctx context.Context, arg db.UpdateFantasyTeamParams) (db.FantasyTeam, error)
}

type Repository struct {
	queries Querier
}

func NewRepository(querier Querier) *Repository {
	return &Repository{
		queries: querier,
	}
}

type CreateFantasyTeamRequest struct {
	LeagueID uuid.UUID `json:"league_id"`
	OwnerID  uuid.UUID `json:"owner_id"`
	Name     string    `json:"name"`
	LogoURL  string    `json:"logo_url"`
}

type UpdateFantasyTeamRequest struct {
	Name    string `json:"name"`
	LogoURL string `json:"logo_url"`
}

func (r *Repository) CreateFantasyTeam(ctx context.Context, req CreateFantasyTeamRequest) (*models.FantasyTeam, error) {
	team, err := r.queries.CreateFantasyTeam(ctx, db.CreateFantasyTeamParams{
		LeagueID: req.LeagueID,
		OwnerID:  req.OwnerID,
		Name:     req.Name,
		LogoUrl:  sql.NullString{String: req.LogoURL, Valid: req.LogoURL != ""},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create fantasy team: %w", err)
	}

	return r.dbFantasyTeamToModel(team), nil
}

func (r *Repository) GetFantasyTeam(ctx context.Context, id uuid.UUID) (*models.FantasyTeam, error) {
	team, err := r.queries.GetFantasyTeam(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get fantasy team: %w", err)
	}

	return r.dbFantasyTeamToModel(team), nil
}

func (r *Repository) GetFantasyTeamsByLeague(ctx context.Context, leagueID uuid.UUID) ([]models.FantasyTeam, error) {
	teams, err := r.queries.GetFantasyTeamsByLeague(ctx, leagueID)
	if err != nil {
		return nil, fmt.Errorf("failed to get fantasy teams by league: %w", err)
	}

	result := make([]models.FantasyTeam, len(teams))
	for i, team := range teams {
		result[i] = *r.dbFantasyTeamToModel(team)
	}
	return result, nil
}

func (r *Repository) GetFantasyTeamsByOwner(ctx context.Context, ownerID uuid.UUID) ([]models.FantasyTeam, error) {
	teams, err := r.queries.GetFantasyTeamsByOwner(ctx, ownerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get fantasy teams by owner: %w", err)
	}

	result := make([]models.FantasyTeam, len(teams))
	for i, team := range teams {
		result[i] = *r.dbFantasyTeamToModel(team)
	}
	return result, nil
}

func (r *Repository) GetFantasyTeamByLeagueAndOwner(ctx context.Context, ownerID, leagueID uuid.UUID) (*models.FantasyTeam, error) {
	team, err := r.queries.GetFantasyTeamByLeagueAndOwner(ctx, db.GetFantasyTeamByLeagueAndOwnerParams{
		OwnerID:  ownerID,
		LeagueID: leagueID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get fantasy team by league and owner: %w", err)
	}

	return r.dbFantasyTeamToModel(team), nil
}

func (r *Repository) UpdateFantasyTeam(ctx context.Context, id uuid.UUID, req UpdateFantasyTeamRequest) (*models.FantasyTeam, error) {
	team, err := r.queries.UpdateFantasyTeam(ctx, db.UpdateFantasyTeamParams{
		ID:      id,
		Name:    req.Name,
		LogoUrl: sql.NullString{String: req.LogoURL, Valid: req.LogoURL != ""},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update fantasy team: %w", err)
	}

	return r.dbFantasyTeamToModel(team), nil
}

func (r *Repository) DeleteFantasyTeam(ctx context.Context, id uuid.UUID) error {
	if err := r.queries.DeleteFantasyTeam(ctx, id); err != nil {
		return fmt.Errorf("failed to delete fantasy team: %w", err)
	}
	return nil
}

func (r *Repository) dbFantasyTeamToModel(dbTeam db.FantasyTeam) *models.FantasyTeam {
	return &models.FantasyTeam{
		ID:        dbTeam.ID,
		LeagueID:  dbTeam.LeagueID,
		OwnerID:   dbTeam.OwnerID,
		Name:      dbTeam.Name,
		LogoURL:   dbTeam.LogoUrl.String,
		CreatedAt: dbTeam.CreatedAt,
	}
}
