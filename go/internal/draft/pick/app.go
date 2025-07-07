package pick

import (
	"context"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/mcdev12/dynasty/go/internal/models"
)

// PickRepository defines what the pick app layer needs from the pick repository
type PickRepository interface {
	CreateDraftPicksBatch(ctx context.Context, picks []models.DraftPick) error
	GetDraftPick(ctx context.Context, id uuid.UUID) (*models.DraftPick, error)
	GetDraftPicksByDraft(ctx context.Context, draftID uuid.UUID) ([]models.DraftPick, error)
	GetDraftPicksByRound(ctx context.Context, draftID uuid.UUID, round int) ([]models.DraftPick, error)
	GetNextPickForDraft(ctx context.Context, draftID uuid.UUID) (*models.DraftPick, error)
	UpdateDraftPickPlayer(ctx context.Context, id uuid.UUID, req UpdateDraftPickPlayerRequest) (*models.DraftPick, error)
	DeleteDraftPicksByDraft(ctx context.Context, draftID uuid.UUID) (int, error)
	MakePick(ctx context.Context, pickRequest MakePickRequest) error
	CountRemainingPicks(ctx context.Context, draftID uuid.UUID) (int, error)
	ClaimNextPickSlot(ctx context.Context, draftID uuid.UUID) (*Slot, error)
	ListAvailablePlayersForDraft(ctx context.Context, draftID uuid.UUID) ([]AvailablePlayer, error)
}

// App handles pick business logic
type App struct {
	repo PickRepository
}

// NewApp creates a new pick App
func NewApp(repo PickRepository) *App {
	return &App{
		repo: repo,
	}
}

// PrepopulateDraftPicks creates all draft pick slots for a draft based on rounds and team count
func (a *App) PrepopulateDraftPicks(ctx context.Context, draftID uuid.UUID, draftType models.DraftType, settings models.DraftSettings) error {
	// Check if picks already exist
	existingPicks, err := a.repo.GetDraftPicksByDraft(ctx, draftID)
	if err != nil {
		return fmt.Errorf("failed to check existing picks: %w", err)
	}
	if len(existingPicks) > 0 {
		return fmt.Errorf("draft picks already exist for this draft (%d picks found)", len(existingPicks))
	}

	// Calculate number of teams from draft order
	numTeams := len(settings.DraftOrder)
	if numTeams == 0 {
		return fmt.Errorf("draft order is empty, cannot prepopulate picks")
	}

	// Generate all draft picks based on draft type
	var picks []models.DraftPick
	switch draftType {
	case models.DraftTypeSnake, models.DraftTypeRookie:
		picks = a.generateSnakeDraftPicks(draftID, settings.Rounds, settings.DraftOrder, settings.ThirdRoundReversal)
	case models.DraftTypeAuction:
		picks = a.generateAuctionDraftPicks(draftID, settings.Rounds, settings.DraftOrder)
	default:
		return fmt.Errorf("unsupported draft type for prepopulation: %s", draftType)
	}

	// Create all picks in batch
	if err := a.repo.CreateDraftPicksBatch(ctx, picks); err != nil {
		return fmt.Errorf("failed to create draft picks: %w", err)
	}

	log.Printf("Prepopulated %d draft picks for %s draft (ID: %s)", len(picks), draftType, draftID)
	return nil
}

// MakePick makes a draft pick
func (a *App) MakePick(ctx context.Context, req MakePickRequest) error {
	if err := a.validateMakePickRequest(req); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	err := a.repo.MakePick(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to make pick: %w", err)
	}
	return nil
}

// GetDraftPick retrieves a draft pick by ID
func (a *App) GetDraftPick(ctx context.Context, id uuid.UUID) (*models.DraftPick, error) {
	pick, err := a.repo.GetDraftPick(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get draft pick: %w", err)
	}
	return pick, nil
}

// GetDraftPicksByDraft retrieves all draft picks for a draft
func (a *App) GetDraftPicksByDraft(ctx context.Context, draftID uuid.UUID) ([]models.DraftPick, error) {
	picks, err := a.repo.GetDraftPicksByDraft(ctx, draftID)
	if err != nil {
		return nil, fmt.Errorf("failed to get draft picks by draft: %w", err)
	}
	return picks, nil
}

// GetDraftPicksByRound retrieves draft picks for a specific round
func (a *App) GetDraftPicksByRound(ctx context.Context, draftID uuid.UUID, round int) ([]models.DraftPick, error) {
	if round <= 0 {
		return nil, fmt.Errorf("round must be greater than 0")
	}

	picks, err := a.repo.GetDraftPicksByRound(ctx, draftID, round)
	if err != nil {
		return nil, fmt.Errorf("failed to get draft picks by round: %w", err)
	}
	return picks, nil
}

// GetNextPickForDraft returns the next pick that needs to be made for a draft
func (a *App) GetNextPickForDraft(ctx context.Context, draftID uuid.UUID) (*models.DraftPick, error) {
	pick, err := a.repo.GetNextPickForDraft(ctx, draftID)
	if err != nil {
		return nil, fmt.Errorf("failed to get next pick for draft: %w", err)
	}
	return pick, nil
}

// CountRemainingPicks returns the number of unpicked slots in a draft
func (a *App) CountRemainingPicks(ctx context.Context, draftID uuid.UUID) (int, error) {
	count, err := a.repo.CountRemainingPicks(ctx, draftID)
	if err != nil {
		return 0, fmt.Errorf("failed to count remaining picks: %w", err)
	}
	return count, nil
}

// ClaimNextPickSlot atomically claims the next available pick slot for auto-pick
func (a *App) ClaimNextPickSlot(ctx context.Context, draftID uuid.UUID) (*Slot, error) {
	slot, err := a.repo.ClaimNextPickSlot(ctx, draftID)
	if err != nil {
		return nil, fmt.Errorf("failed to claim next pick slot: %w", err)
	}
	return slot, nil
}

// UpdateDraftPickPlayer updates a draft pick with player information
func (a *App) UpdateDraftPickPlayer(ctx context.Context, id uuid.UUID, req UpdateDraftPickPlayerRequest) (*models.DraftPick, error) {
	if err := a.validateUpdateDraftPickPlayerRequest(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Verify pick exists
	_, err := a.repo.GetDraftPick(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("draft pick not found: %w", err)
	}

	pick, err := a.repo.UpdateDraftPickPlayer(ctx, id, req)
	if err != nil {
		return nil, fmt.Errorf("failed to update draft pick player: %w", err)
	}

	log.Printf("Updated draft pick: %s with player %s", id, req.PlayerID)
	return pick, nil
}

// DeleteDraftPicksByDraft deletes all draft picks for a draft
func (a *App) DeleteDraftPicksByDraft(ctx context.Context, draftID uuid.UUID) (int, error) {
	count, err := a.repo.DeleteDraftPicksByDraft(ctx, draftID)
	if err != nil {
		return 0, fmt.Errorf("failed to delete draft picks: %w", err)
	}

	log.Printf("Deleted %d draft picks for draft: %s", count, draftID)
	return count, nil
}

// ListAvailablePlayersForDraft returns all players not yet picked in the specified draft
func (a *App) ListAvailablePlayersForDraft(ctx context.Context, draftID uuid.UUID) ([]AvailablePlayer, error) {
	players, err := a.repo.ListAvailablePlayersForDraft(ctx, draftID)
	if err != nil {
		return nil, fmt.Errorf("failed to list available players for draft: %w", err)
	}

	return players, nil
}


// generateSnakeDraftPicks generates picks for snake and rookie drafts with optional reversal
func (a *App) generateSnakeDraftPicks(draftID uuid.UUID, rounds int, draftOrder []uuid.UUID, thirdRoundReversal bool) []models.DraftPick {
	numTeams := len(draftOrder)
	totalPicks := rounds * numTeams
	picks := make([]models.DraftPick, 0, totalPicks)

	overallPick := 1

	for round := 1; round <= rounds; round++ {
		// Determine if this round should be reversed
		isReversed := (round%2 == 0) // Even rounds are always reversed in snake
		if thirdRoundReversal && round >= 3 {
			// Apply third round reversal logic - reverse every round from round 3 onwards
			isReversed = true
		}

		var roundOrder []uuid.UUID
		if isReversed {
			// Reverse the draft order for this round
			roundOrder = make([]uuid.UUID, numTeams)
			for i, teamID := range draftOrder {
				roundOrder[numTeams-1-i] = teamID
			}
		} else {
			roundOrder = draftOrder
		}

		// Create picks for this round
		for pick, teamID := range roundOrder {
			picks = append(picks, models.DraftPick{
				ID:          uuid.New(),
				DraftID:     draftID,
				Round:       round,
				Pick:        pick + 1, // 1-indexed pick number within round
				OverallPick: overallPick,
				TeamID:      teamID,
				PlayerID:    nil,   // Will be set when pick is made
				PickedAt:    nil,   // Will be set when pick is made
				KeeperPick:  false, // Default to false, can be updated later
			})
			overallPick++
		}
	}

	return picks
}

// generateAuctionDraftPicks generates picks for auction drafts (linear order, no reversal)
func (a *App) generateAuctionDraftPicks(draftID uuid.UUID, rounds int, draftOrder []uuid.UUID) []models.DraftPick {
	numTeams := len(draftOrder)
	totalPicks := rounds * numTeams
	picks := make([]models.DraftPick, 0, totalPicks)

	overallPick := 1

	for round := 1; round <= rounds; round++ {
		// Auction drafts maintain the same order every round (no snake reversal)
		for pick, teamID := range draftOrder {
			picks = append(picks, models.DraftPick{
				ID:            uuid.New(),
				DraftID:       draftID,
				Round:         round,
				Pick:          pick + 1, // 1-indexed pick number within round
				OverallPick:   overallPick,
				TeamID:        teamID,
				PlayerID:      nil,   // Will be set when pick is made
				PickedAt:      nil,   // Will be set when pick is made
				AuctionAmount: nil,   // Will be set when pick is made
				KeeperPick:    false, // Default to false
			})
			overallPick++
		}
	}

	return picks
}

// Validation methods

func (a *App) validateMakePickRequest(req MakePickRequest) error {
	if req.PickID == uuid.Nil {
		return fmt.Errorf("pick_id is required")
	}
	if req.PlayerID == uuid.Nil {
		return fmt.Errorf("player_id is required")
	}
	if req.DraftID == uuid.Nil {
		return fmt.Errorf("draft_id is required")
	}
	if req.TeamID == uuid.Nil {
		return fmt.Errorf("team_id is required")
	}
	if req.OverallPick <= 0 {
		return fmt.Errorf("overall_pick must be greater than 0")
	}
	return nil
}

func (a *App) validateUpdateDraftPickPlayerRequest(req UpdateDraftPickPlayerRequest) error {
	if req.PlayerID == uuid.Nil {
		return fmt.Errorf("player_id is required")
	}
	return nil
}