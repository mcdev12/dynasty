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
		Position:     sql.NullString{String: nflProfile.Position, Valid: nflProfile.Position != ""},
		Status:       sql.NullString{String: nflProfile.Status, Valid: nflProfile.Status != ""},
		College:      sql.NullString{String: nflProfile.College, Valid: nflProfile.College != ""},
		JerseyNumber: sqlutil.ToSqlInt16(nflProfile.JerseyNumber),
		Experience:   sqlutil.ToSqlInt16(nflProfile.Experience),
		BirthDate:    sqlutil.ToSqlTime(nflProfile.BirthDate),
		HeightCm:     sql.NullInt32{Int32: int32(nflProfile.HeightCm), Valid: true},
		WeightKg:     sql.NullInt32{Int32: int32(nflProfile.WeightKg), Valid: true},
		HeightDesc:   sql.NullString{String: nflProfile.HeightDesc, Valid: nflProfile.HeightDesc != ""},
		WeightDesc:   sql.NullString{String: nflProfile.WeightDesc, Valid: nflProfile.WeightDesc != ""},
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
		Position:     sqlutil.FromSqlString(dbProfile.Position, ""),
		Status:       sqlutil.FromSqlString(dbProfile.Status, ""),
		College:      sqlutil.FromSqlString(dbProfile.College, ""),
		JerseyNumber: sqlutil.FromSqlInt16(dbProfile.JerseyNumber),
		Experience:   sqlutil.FromSqlInt16(dbProfile.Experience),
		BirthDate:    sqlutil.FromSqlTime(dbProfile.BirthDate),
		HeightCm:     int(dbProfile.HeightCm.Int32),
		WeightKg:     int(dbProfile.WeightKg.Int32),
		HeightDesc:   sqlutil.FromSqlString(dbProfile.HeightDesc, ""),
		WeightDesc:   sqlutil.FromSqlString(dbProfile.WeightDesc, ""),
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
		Position:     sql.NullString{String: nflProfile.Position, Valid: nflProfile.Position != ""},
		Status:       sql.NullString{String: nflProfile.Status, Valid: nflProfile.Status != ""},
		College:      sql.NullString{String: nflProfile.College, Valid: nflProfile.College != ""},
		JerseyNumber: sqlutil.ToSqlInt16(nflProfile.JerseyNumber),
		Experience:   sqlutil.ToSqlInt16(nflProfile.Experience),
		BirthDate:    sqlutil.ToSqlTime(nflProfile.BirthDate),
		HeightCm:     sql.NullInt32{Int32: int32(nflProfile.HeightCm), Valid: true},
		WeightKg:     sql.NullInt32{Int32: int32(nflProfile.WeightKg), Valid: true},
		HeightDesc:   sql.NullString{String: nflProfile.HeightDesc, Valid: nflProfile.HeightDesc != ""},
		WeightDesc:   sql.NullString{String: nflProfile.WeightDesc, Valid: nflProfile.WeightDesc != ""},
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