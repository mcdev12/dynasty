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

//TODO figure out how to mock querier interface as it uses profile and player methods.

// Repository handles all player-related database operations
type Repository struct {
	db      *sql.DB
	queries *db.Queries
}

// NewRepository creates a new player repository
func NewRepository(queries *db.Queries, database *sql.DB) *Repository {
	return &Repository{
		queries: queries,
		db:      database,
	}
}

// CreatePlayerRequest contains all data needed to create a player
type CreatePlayerRequest struct {
	SportID    string
	ExternalID string
	FullName   string
	TeamID     *uuid.UUID
	Profile    models.Profile // Sport-specific profile (e.g., *models.NFLPlayerProfile)
}

// CreatePlayer creates a player and their sport-specific profile in a transaction
func (r *Repository) CreatePlayer(ctx context.Context, req CreatePlayerRequest) (*models.Player, error) {
	// Start a transaction
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback() // Ignore error since Commit might have succeeded
	}()

	// Use transaction queries
	qtx := r.queries.WithTx(tx)

	// Create the base player
	params := db.CreatePlayerParams{
		SportID:    req.SportID,
		ExternalID: req.ExternalID,
		FullName:   req.FullName,
		TeamID:     sqlutil.ToNullUUID(req.TeamID),
	}

	dbPlayer, err := qtx.CreatePlayer(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create player: %w", err)
	}

	// Convert to domain model
	player := dbPlayerToDomain(dbPlayer)

	// Create sport-specific profile if provided
	if req.Profile != nil {
		profileRepo, err := GetProfileRepo(req.SportID)
		if err != nil {
			return nil, fmt.Errorf("failed to get profile repository: %w", err)
		}

		// Pass the transaction queries directly
		err = profileRepo.CreateProfile(ctx, qtx, player.ID, req.Profile)
		if err != nil {
			return nil, fmt.Errorf("failed to create player profile: %w", err)
		}

		// Attach the profile to the player model
		switch p := req.Profile.(type) {
		case *models.NFLPlayerProfile:
			player.NFLPlayerProfile = p
		}
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return player, nil
}

// GetPlayer retrieves a player by ID with their sport-specific profile
func (r *Repository) GetPlayer(ctx context.Context, id uuid.UUID) (*models.Player, error) {
	dbPlayer, err := r.queries.GetPlayer(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("player not found")
		}
		return nil, fmt.Errorf("failed to get player: %w", err)
	}

	player := dbPlayerToDomain(dbPlayer)

	// Load sport-specific profile
	if err := LoadProfileIntoPlayer(ctx, r.queries, player); err != nil {
		return nil, err
	}

	return player, nil
}

// GetPlayerByExternalID retrieves a player by sport ID and external ID with their profile
func (r *Repository) GetPlayerByExternalID(ctx context.Context, sportID, externalID string) (*models.Player, error) {
	params := db.GetPlayerByExternalIDParams{
		SportID:    sportID,
		ExternalID: externalID,
	}

	dbPlayer, err := r.queries.GetPlayerByExternalID(ctx, params)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("player not found")
		}
		return nil, fmt.Errorf("failed to get player by external ID: %w", err)
	}

	player := dbPlayerToDomain(dbPlayer)

	// Load sport-specific profile
	if err := LoadProfileIntoPlayer(ctx, r.queries, player); err != nil {
		return nil, err
	}

	return player, nil
}

// DeletePlayer deletes a player and their sport-specific profile in a transaction
func (r *Repository) DeletePlayer(ctx context.Context, id uuid.UUID) error {
	// Start a transaction
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback() // Ignore error since Commit might have succeeded
	}()

	// Use transaction queries
	qtx := r.queries.WithTx(tx)

	// Get the player within the transaction to know their sport
	dbPlayer, err := qtx.GetPlayer(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("player not found")
		}
		return fmt.Errorf("failed to get player: %w", err)
	}

	player := dbPlayerToDomain(dbPlayer)

	// Delete sport-specific profile first (due to foreign key constraint)
	profileRepo, err := GetProfileRepo(player.SportID)
	if err == nil {
		// If there's a profile repo, try to delete the profile
		if err := profileRepo.DeleteProfile(ctx, qtx, id); err != nil {
			return fmt.Errorf("failed to delete player profile: %w", err)
		}
	}

	// Delete the base player
	if err := qtx.DeletePlayer(ctx, id); err != nil {
		return fmt.Errorf("failed to delete player: %w", err)
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// UpdatePlayer updates a player's base information
func (r *Repository) UpdatePlayer(ctx context.Context, playerID uuid.UUID, fullName string, teamID *uuid.UUID) (*models.Player, error) {
	params := db.UpdatePlayerParams{
		ID:       playerID,
		FullName: fullName,
		TeamID:   sqlutil.ToNullUUID(teamID),
	}

	dbPlayer, err := r.queries.UpdatePlayer(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to update player: %w", err)
	}

	player := dbPlayerToDomain(dbPlayer)

	// Load sport-specific profile
	if err := LoadProfileIntoPlayer(ctx, r.queries, player); err != nil {
		return nil, err
	}

	return player, nil
}

// UpdatePlayerProfile updates a player's sport-specific profile
func (r *Repository) UpdatePlayerProfile(ctx context.Context, playerID uuid.UUID, profile models.Profile) error {
	// Get the player to determine the sport
	player, err := r.GetPlayer(ctx, playerID)
	if err != nil {
		return fmt.Errorf("failed to get player: %w", err)
	}

	// Get the profile repository for this sport
	profileRepo, err := GetProfileRepo(player.SportID)
	if err != nil {
		return fmt.Errorf("failed to get profile repository: %w", err)
	}

	// Update the profile using the profile repository
	err = profileRepo.UpdateProfile(ctx, r.queries, playerID, profile)
	if err != nil {
		return fmt.Errorf("failed to update profile: %w", err)
	}

	return nil
}

// UpdatePlayerAndProfile updates both player base info and sport-specific profile
func (r *Repository) UpdatePlayerAndProfile(ctx context.Context, playerID uuid.UUID, fullName string, teamID *uuid.UUID, profile models.Profile) (*models.Player, error) {
	// Start a transaction
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback() // Ignore error since Commit might have succeeded
	}()

	// Use transaction queries
	qtx := r.queries.WithTx(tx)

	// Update base player info
	params := db.UpdatePlayerParams{
		ID:       playerID,
		FullName: fullName,
		TeamID:   sqlutil.ToNullUUID(teamID),
	}

	dbPlayer, err := qtx.UpdatePlayer(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to update player: %w", err)
	}

	player := dbPlayerToDomain(dbPlayer)

	// Update sport-specific profile if provided
	if profile != nil {
		profileRepo, err := GetProfileRepo(player.SportID)
		if err != nil {
			return nil, fmt.Errorf("failed to get profile repository: %w", err)
		}

		err = profileRepo.UpdateProfile(ctx, qtx, playerID, profile)
		if err != nil {
			return nil, fmt.Errorf("failed to update player profile: %w", err)
		}

		// Attach the profile to the player model
		switch p := profile.(type) {
		case *models.NFLPlayerProfile:
			player.NFLPlayerProfile = p
		}
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return player, nil
}

// Helper function to convert database player to domain model
func dbPlayerToDomain(dbPlayer db.Player) *models.Player {
	player := &models.Player{
		ID:         dbPlayer.ID,
		SportID:    dbPlayer.SportID,
		ExternalID: dbPlayer.ExternalID,
		FullName:   dbPlayer.FullName,
		CreatedAt:  dbPlayer.CreatedAt,
		TeamID:     sqlutil.FromNullUUID(dbPlayer.TeamID),
	}

	return player
}
