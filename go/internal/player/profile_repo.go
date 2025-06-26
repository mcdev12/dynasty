package player

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/mcdev12/dynasty/go/internal/models"
	"github.com/mcdev12/dynasty/go/internal/player/db"
)

// ProfileRepository defines the interface for sport-specific player profile operations
type ProfileRepository interface {
	// CreateProfile creates a sport-specific profile for a player
	CreateProfile(ctx context.Context, qtx db.Querier, playerID uuid.UUID, profile models.Profile) error
	
	// LoadProfile loads a sport-specific profile for a player
	LoadProfile(ctx context.Context, q db.Querier, playerID uuid.UUID) (models.Profile, error)
	
	// UpdateProfile updates a sport-specific profile for a player
	UpdateProfile(ctx context.Context, qtx db.Querier, playerID uuid.UUID, profile models.Profile) error
	
	// DeleteProfile deletes a sport-specific profile for a player
	DeleteProfile(ctx context.Context, qtx db.Querier, playerID uuid.UUID) error
}

// profileRegistry manages sport-specific profile repositories
type profileRegistry struct {
	mu       sync.RWMutex
	profiles map[string]ProfileRepository
}

// global registry instance
var registry = &profileRegistry{
	profiles: make(map[string]ProfileRepository),
}

// RegisterProfileRepo registers a profile repository for a specific sport
func RegisterProfileRepo(sportID string, repo ProfileRepository) error {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	
	if sportID == "" {
		return fmt.Errorf("sport ID cannot be empty")
	}
	
	if _, exists := registry.profiles[sportID]; exists {
		return fmt.Errorf("profile repository already registered for sport %q", sportID)
	}
	
	registry.profiles[sportID] = repo
	return nil
}

// GetProfileRepo retrieves a profile repository for a specific sport
func GetProfileRepo(sportID string) (ProfileRepository, error) {
	registry.mu.RLock()
	defer registry.mu.RUnlock()
	
	repo, exists := registry.profiles[sportID]
	if !exists {
		return nil, fmt.Errorf("no profile repository registered for sport %q", sportID)
	}
	
	return repo, nil
}

// CreateProfileRequest represents a request to create a player profile
type CreateProfileRequest struct {
	Player  *models.Player
	Profile models.Profile // Sport-specific profile data
}

// LoadProfileIntoPlayer loads the appropriate sport-specific profile into a player
func LoadProfileIntoPlayer(ctx context.Context, q db.Querier, player *models.Player) error {
	repo, err := GetProfileRepo(player.SportID)
	if err != nil {
		// No profile repo for this sport is not an error - some sports may not have profiles
		return nil
	}
	
	profile, err := repo.LoadProfile(ctx, q, player.ID)
	if err != nil {
		if errors.Is(err, ErrNoProfile) {
			// No profile exists for this player - this is not an error
			return nil
		}
		return fmt.Errorf("failed to load profile for sport %s: %w", player.SportID, err)
	}
	
	// Type switch to assign the profile to the appropriate field
	switch p := profile.(type) {
	case *models.NFLPlayerProfile:
		player.NFLPlayerProfile = p
	// Add more sports as they are implemented:
	// case *models.NBAPlayerProfile:
	//     player.NBAPlayerProfile = p
	}
	
	return nil
}