package leagues

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/mcdev12/dynasty/go/internal/leagues/db"
	"github.com/mcdev12/dynasty/go/internal/models"
)

// Querier defines what the repository needs from the database layer
type Querier interface {
	CreateLeague(ctx context.Context, arg db.CreateLeagueParams) (db.League, error)
	DeleteLeague(ctx context.Context, id uuid.UUID) error
	GetLeague(ctx context.Context, id uuid.UUID) (db.League, error)
	GetLeaguesByCommissioner(ctx context.Context, commissionerID uuid.UUID) ([]db.League, error)
	UpdateLeague(ctx context.Context, arg db.UpdateLeagueParams) (db.League, error)
	UpdateLeagueSettings(ctx context.Context, arg db.UpdateLeagueSettingsParams) (db.League, error)
	UpdateLeagueStatus(ctx context.Context, arg db.UpdateLeagueStatusParams) (db.League, error)
}

// Repository implements league data access operations
type Repository struct {
	queries Querier
}

// NewRepository creates a new leagues repository
func NewRepository(querier Querier) *Repository {
	return &Repository{
		queries: querier,
	}
}

// CreateLeague creates a new league
func (r *Repository) CreateLeague(ctx context.Context, req CreateLeagueRequest) (*models.League, error) {
	// Marshal league settings to JSON
	settingsJSON, err := json.Marshal(req.LeagueSettings)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal league settings: %w", err)
	}

	league, err := r.queries.CreateLeague(ctx, db.CreateLeagueParams{
		Name:           req.Name,
		SportID:        req.SportID,
		LeagueType:     db.LeagueType(req.LeagueType),
		CommissionerID: req.CommissionerID,
		LeagueSettings: settingsJSON,
		Status:         db.LeagueStatus(req.Status),
		Season:         req.Season,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create league: %w", err)
	}

	return r.dbLeagueToModel(league), nil
}

// GetLeague retrieves a league by ID
func (r *Repository) GetLeague(ctx context.Context, id uuid.UUID) (*models.League, error) {
	league, err := r.queries.GetLeague(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get league: %w", err)
	}

	return r.dbLeagueToModel(league), nil
}

// GetLeaguesByCommissioner retrieves leagues by commissioner ID
func (r *Repository) GetLeaguesByCommissioner(ctx context.Context, commissionerID uuid.UUID) ([]models.League, error) {
	leagues, err := r.queries.GetLeaguesByCommissioner(ctx, commissionerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get leagues by commissioner: %w", err)
	}

	return r.dbLeaguesToModels(leagues), nil
}

// UpdateLeague updates an existing league
func (r *Repository) UpdateLeague(ctx context.Context, id uuid.UUID, req UpdateLeagueRequest) (*models.League, error) {
	// Marshal league settings to JSON
	settingsJSON, err := json.Marshal(req.LeagueSettings)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal league settings: %w", err)
	}

	league, err := r.queries.UpdateLeague(ctx, db.UpdateLeagueParams{
		ID:             id,
		Name:           req.Name,
		SportID:        req.SportID,
		LeagueType:     db.LeagueType(req.LeagueType),
		CommissionerID: req.CommissionerID,
		LeagueSettings: settingsJSON,
		Status:         db.LeagueStatus(req.Status),
		Season:         req.Season,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update league: %w", err)
	}

	return r.dbLeagueToModel(league), nil
}

// UpdateLeagueStatus updates only the status of a league
func (r *Repository) UpdateLeagueStatus(ctx context.Context, id uuid.UUID, status models.LeagueStatus) (*models.League, error) {
	league, err := r.queries.UpdateLeagueStatus(ctx, db.UpdateLeagueStatusParams{
		ID:     id,
		Status: db.LeagueStatus(status),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update league status: %w", err)
	}

	return r.dbLeagueToModel(league), nil
}

// UpdateLeagueSettings updates only the settings of a league
func (r *Repository) UpdateLeagueSettings(ctx context.Context, id uuid.UUID, settings interface{}) (*models.League, error) {
	// Marshal league settings to JSON
	settingsJSON, err := json.Marshal(settings)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal league settings: %w", err)
	}

	league, err := r.queries.UpdateLeagueSettings(ctx, db.UpdateLeagueSettingsParams{
		ID:             id,
		LeagueSettings: settingsJSON,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update league settings: %w", err)
	}

	return r.dbLeagueToModel(league), nil
}

// DeleteLeague deletes a league by ID
func (r *Repository) DeleteLeague(ctx context.Context, id uuid.UUID) error {
	if err := r.queries.DeleteLeague(ctx, id); err != nil {
		return fmt.Errorf("failed to delete league: %w", err)
	}
	return nil
}

// dbLeagueToModel converts a database league to domain model
func (r *Repository) dbLeagueToModel(dbLeague db.League) *models.League {
	// Unmarshal league settings from JSON
	var settings interface{}
	if len(dbLeague.LeagueSettings) > 0 {
		if err := json.Unmarshal(dbLeague.LeagueSettings, &settings); err != nil {
			// If unmarshal fails, store as raw JSON string
			settings = string(dbLeague.LeagueSettings)
		}
	}

	return &models.League{
		ID:             dbLeague.ID,
		Name:           dbLeague.Name,
		SportID:        dbLeague.SportID,
		LeagueType:     models.LeagueType(dbLeague.LeagueType),
		CommissionerID: dbLeague.CommissionerID,
		LeagueSettings: settings,
		Status:         models.LeagueStatus(dbLeague.Status),
		Season:         dbLeague.Season,
		CreatedAt:      dbLeague.CreatedAt,
		UpdatedAt:      dbLeague.UpdatedAt,
	}
}

// dbLeaguesToModels converts multiple database leagues to domain models
func (r *Repository) dbLeaguesToModels(dbLeagues []db.League) []models.League {
	leagues := make([]models.League, len(dbLeagues))
	for i, dbLeague := range dbLeagues {
		leagues[i] = *r.dbLeagueToModel(dbLeague)
	}
	return leagues
}
