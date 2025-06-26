package player

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/mcdev12/dynasty/go/internal/models"
	"github.com/mcdev12/dynasty/go/internal/player/db"
	"github.com/mcdev12/dynasty/go/internal/sqlutil"
)

// NFLProfileRepository handles NFL-specific player profile operations
type NFLProfileRepository struct{}

// NewNFLProfileRepository creates a new NFL profile repository
func NewNFLProfileRepository() *NFLProfileRepository {
	return &NFLProfileRepository{}
}

// CreateProfile creates an NFL player profile
func (r *NFLProfileRepository) CreateProfile(ctx context.Context, qtx db.Querier, playerID uuid.UUID, profile models.Profile) error {
	nflProfile, ok := profile.(*models.NFLPlayerProfile)
	if !ok {
		return fmt.Errorf("expected *models.NFLPlayerProfile, got %T", profile)
	}
	
	// Convert domain model to database params
	params := db.CreateNFLPlayerProfileParams{
		PlayerID:     playerID,
		HeightCm:     sqlutil.ToSqlInt32(nflProfile.HeightCm),
		WeightKg:     sqlutil.ToSqlInt32(nflProfile.WeightKg),
		GroupRole:    sql.NullString{String: nflProfile.GroupRole, Valid: nflProfile.GroupRole != ""},
		Position:     sql.NullString{String: nflProfile.Position, Valid: nflProfile.Position != ""},
		Age:          sqlutil.ToSqlInt32(nflProfile.Age),
		HeightDesc:   sql.NullString{String: nflProfile.HeightDesc, Valid: nflProfile.HeightDesc != ""},
		WeightDesc:   sql.NullString{String: nflProfile.WeightDesc, Valid: nflProfile.WeightDesc != ""},
		College:      sqlutil.ToSqlString(nflProfile.College),
		JerseyNumber: sqlutil.ToSqlInt16(nflProfile.JerseyNumber),
		SalaryDesc:   sqlutil.ToSqlString(nflProfile.SalaryDesc),
		Experience:   sqlutil.ToSqlInt16(nflProfile.Experience),
	}
	
	_, err := qtx.CreateNFLPlayerProfile(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to create NFL player profile: %w", err)
	}
	
	return nil
}

// LoadProfile loads an NFL player profile
func (r *NFLProfileRepository) LoadProfile(ctx context.Context, q db.Querier, playerID uuid.UUID) (models.Profile, error) {
	dbProfile, err := q.GetNFLPlayerProfile(ctx, playerID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNoProfile // No profile exists for this player
		}
		return nil, fmt.Errorf("failed to get NFL player profile: %w", err)
	}
	
	// Convert database model to domain model
	profile := &models.NFLPlayerProfile{
		PlayerID:     playerID,
		HeightCm:     sqlutil.FromSqlInt32(dbProfile.HeightCm),
		WeightKg:     sqlutil.FromSqlInt32(dbProfile.WeightKg),
		GroupRole:    sqlutil.FromSqlString(dbProfile.GroupRole, ""),
		Position:     sqlutil.FromSqlString(dbProfile.Position, ""),
		Age:          sqlutil.FromSqlInt32(dbProfile.Age),
		HeightDesc:   sqlutil.FromSqlString(dbProfile.HeightDesc, ""),
		WeightDesc:   sqlutil.FromSqlString(dbProfile.WeightDesc, ""),
		College:      sqlutil.FromSqlStringPtr(dbProfile.College),
		JerseyNumber: sqlutil.FromSqlInt16(dbProfile.JerseyNumber),
		SalaryDesc:   sqlutil.FromSqlStringPtr(dbProfile.SalaryDesc),
		Experience:   sqlutil.FromSqlInt16(dbProfile.Experience),
	}
	
	return profile, nil
}

// UpdateProfile updates an NFL player profile
func (r *NFLProfileRepository) UpdateProfile(ctx context.Context, qtx db.Querier, playerID uuid.UUID, profile models.Profile) error {
	nflProfile, ok := profile.(*models.NFLPlayerProfile)
	if !ok {
		return fmt.Errorf("expected *models.NFLPlayerProfile, got %T", profile)
	}

	// Convert domain model to database params
	params := db.UpdateNFLPlayerProfileParams{
		PlayerID:     playerID,
		HeightCm:     sqlutil.ToSqlInt32(nflProfile.HeightCm),
		WeightKg:     sqlutil.ToSqlInt32(nflProfile.WeightKg),
		GroupRole:    sql.NullString{String: nflProfile.GroupRole, Valid: nflProfile.GroupRole != ""},
		Position:     sql.NullString{String: nflProfile.Position, Valid: nflProfile.Position != ""},
		Age:          sqlutil.ToSqlInt32(nflProfile.Age),
		HeightDesc:   sql.NullString{String: nflProfile.HeightDesc, Valid: nflProfile.HeightDesc != ""},
		WeightDesc:   sql.NullString{String: nflProfile.WeightDesc, Valid: nflProfile.WeightDesc != ""},
		College:      sqlutil.ToSqlString(nflProfile.College),
		JerseyNumber: sqlutil.ToSqlInt16(nflProfile.JerseyNumber),
		SalaryDesc:   sqlutil.ToSqlString(nflProfile.SalaryDesc),
		Experience:   sqlutil.ToSqlInt16(nflProfile.Experience),
	}

	_, err := qtx.UpdateNFLPlayerProfile(ctx, params)
	if err != nil {
		return fmt.Errorf("failed to update NFL player profile: %w", err)
	}

	return nil
}

// DeleteProfile deletes an NFL player profile
func (r *NFLProfileRepository) DeleteProfile(ctx context.Context, qtx db.Querier, playerID uuid.UUID) error {
	err := qtx.DeleteNFLPlayerProfile(ctx, playerID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("failed to delete NFL player profile: %w", err)
	}
	
	// Return nil even if no rows affected (idempotent delete)
	return nil
}

// init registers the NFL profile repository on package initialization
func init() {
	if err := RegisterProfileRepo("nfl", NewNFLProfileRepository()); err != nil {
		panic(fmt.Sprintf("Failed to register NFL profile repository: %v", err))
	}
}