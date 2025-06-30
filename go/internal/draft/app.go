package draft

import (
	"context"
	"fmt"
	"github.com/mcdev12/dynasty/go/internal/draft/repository"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/mcdev12/dynasty/go/internal/models"
)

// DraftRepository defines what the app layer needs from the repository
type DraftRepository interface {
	CreateDraft(ctx context.Context, req repository.CreateDraftRequest) (*models.Draft, error)
	GetDraft(ctx context.Context, id uuid.UUID) (*models.Draft, error)
	UpdateDraftStatus(ctx context.Context, id uuid.UUID, req repository.UpdateDraftStatusRequest) (*models.Draft, error)
	UpdateDraft(ctx context.Context, id uuid.UUID, req repository.UpdateDraftRequest) (*models.Draft, error)
	DeleteDraft(ctx context.Context, id uuid.UUID) error
	FetchNextDeadline(ctx context.Context) (*repository.NextDeadline, error)
	FetchDraftsDueForPick(ctx context.Context, limit int32) ([]uuid.UUID, error)
	UpdateNextDeadline(ctx context.Context, draftID uuid.UUID, deadline *time.Time) error
	ClearNextDeadline(ctx context.Context, id uuid.UUID) error
	ListAvailablePlayersForDraft(ctx context.Context, draftID uuid.UUID) ([]repository.AvailablePlayer, error)
}

// DraftPickRepositoryImpl defines what the app layer needs from the draft pick repository
type DraftPickRepositoryImpl interface {
	CreateDraftPicksBatch(ctx context.Context, picks []models.DraftPick) error
	GetDraftPicksByDraft(ctx context.Context, draftID uuid.UUID) ([]models.DraftPick, error)
	MakePick(ctx context.Context, pickRequest repository.MakePickRequest) error
	CountRemainingPicks(ctx context.Context, draftID uuid.UUID) (int, error)
	ClaimNextPickSlot(ctx context.Context, draftID uuid.UUID) (*repository.Slot, error)
	GetNextPickForDraft(ctx context.Context, draftID uuid.UUID) (*models.DraftPick, error)
	InsertOutboxPickStarted(ctx context.Context, draftID uuid.UUID, payload []byte) error
}

// LeaguesRepository defines what the app layer needs from the leagues repository for validation
type LeaguesRepository interface {
	GetLeague(ctx context.Context, id uuid.UUID) (*models.League, error)
}

// App handles draft business logic
type App struct {
	repo        DraftRepository
	pickRepo    DraftPickRepositoryImpl
	leaguesRepo LeaguesRepository
}

// NewApp creates a new draft App
// TODO use zerolog
func NewApp(repo DraftRepository, pickRepo DraftPickRepositoryImpl, leaguesRepo LeaguesRepository) *App {
	return &App{
		repo:        repo,
		pickRepo:    pickRepo,
		leaguesRepo: leaguesRepo,
	}
}

// CreateDraft creates a new draft with validation
// TODO create draft status should always be in not started
// TODO later in Init league we will do lots of things
func (a *App) CreateDraft(ctx context.Context, req repository.CreateDraftRequest) (*models.Draft, error) {
	if err := a.validateCreateDraftRequest(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Verify league exists
	league, err := a.leaguesRepo.GetLeague(ctx, req.LeagueID)
	if err != nil {
		return nil, fmt.Errorf("league not found: %w", err)
	}

	// Validate draft settings based on draft type
	if err := a.validateDraftSettings(req.DraftType, req.Settings); err != nil {
		return nil, fmt.Errorf("invalid draft settings: %w", err)
	}

	draft, err := a.repo.CreateDraft(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create draft: %w", err)
	}

	log.Printf("Created draft: %s draft for league %s", draft.DraftType, league.Name)
	return draft, nil
}

// GetDraft retrieves a draft by ID
func (a *App) GetDraft(ctx context.Context, id uuid.UUID) (*models.Draft, error) {
	draft, err := a.repo.GetDraft(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get draft: %w", err)
	}
	return draft, nil
}

// UpdateDraftStatus updates the status of a draft with validation
func (a *App) UpdateDraftStatus(ctx context.Context, id uuid.UUID, req repository.UpdateDraftStatusRequest) (*models.Draft, error) {
	if err := a.validateDraftStatus(req.Status); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Verify draft exists and get current status
	currentDraft, err := a.repo.GetDraft(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("draft not found: %w", err)
	}

	// Validate status transition
	if err := a.validateStatusTransition(currentDraft.Status, req.Status); err != nil {
		return nil, fmt.Errorf("invalid status transition: %w", err)
	}

	draft, err := a.repo.UpdateDraftStatus(ctx, id, req)
	if err != nil {
		return nil, fmt.Errorf("failed to update draft status: %w", err)
	}

	log.Printf("Updated draft status: %s -> %s", currentDraft.Status, req.Status)
	return draft, nil
}

// UpdateDraft updates a draft's settings and/or scheduled time
func (a *App) UpdateDraft(ctx context.Context, id uuid.UUID, req repository.UpdateDraftRequest) (*models.Draft, error) {
	// Verify draft exists
	currentDraft, err := a.repo.GetDraft(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("draft not found: %w", err)
	}

	// Only allow updates for NOT_STARTED drafts
	if currentDraft.Status != models.DraftStatusNotStarted {
		return nil, fmt.Errorf("can only update drafts with status %s, current status is %s",
			models.DraftStatusNotStarted, currentDraft.Status)
	}

	// Validate new settings if provided
	if req.Settings != nil {
		if err := a.validateDraftSettings(currentDraft.DraftType, *req.Settings); err != nil {
			return nil, fmt.Errorf("invalid draft settings: %w", err)
		}
	}

	// Validate scheduled_at if provided
	if req.ScheduledAt != nil && req.ScheduledAt.Before(time.Now()) {
		return nil, fmt.Errorf("scheduled_at must be in the future")
	}

	draft, err := a.repo.UpdateDraft(ctx, id, req)
	if err != nil {
		return nil, fmt.Errorf("failed to update draft: %w", err)
	}

	log.Printf("Updated draft %s: settings=%v, scheduled_at=%v", id, req.Settings != nil, req.ScheduledAt)
	return draft, nil
}

// DeleteDraft deletes a draft by ID (only allowed for NOT_STARTED drafts)
// TODO when would you ever want to do this lol
func (a *App) DeleteDraft(ctx context.Context, id uuid.UUID) error {
	// Verify draft exists and check status
	draft, err := a.repo.GetDraft(ctx, id)
	if err != nil {
		return fmt.Errorf("draft not found: %w", err)
	}

	// Only allow deletion of drafts that haven't started
	if draft.Status != models.DraftStatusNotStarted {
		return fmt.Errorf("cannot delete draft with status %s, only %s drafts can be deleted",
			draft.Status, models.DraftStatusNotStarted)
	}

	if err := a.repo.DeleteDraft(ctx, id); err != nil {
		return fmt.Errorf("failed to delete draft: %w", err)
	}

	log.Printf("Deleted draft: %s draft (status: %s)", draft.DraftType, draft.Status)
	return nil
}

// PrepopulateDraftPicks creates all draft pick slots for a draft based on rounds and team count
func (a *App) PrepopulateDraftPicks(ctx context.Context, draftID uuid.UUID) error {
	// Get the draft to access settings
	draft, err := a.repo.GetDraft(ctx, draftID)
	if err != nil {
		return fmt.Errorf("failed to get draft: %w", err)
	}

	// Only allow prepopulation for NOT_STARTED drafts
	if draft.Status != models.DraftStatusNotStarted {
		return fmt.Errorf("can only prepopulate picks for drafts with status %s, current status is %s",
			models.DraftStatusNotStarted, draft.Status)
	}

	// Check if picks already exist
	existingPicks, err := a.pickRepo.GetDraftPicksByDraft(ctx, draftID)
	if err != nil {
		return fmt.Errorf("failed to check existing picks: %w", err)
	}
	if len(existingPicks) > 0 {
		return fmt.Errorf("draft picks already exist for this draft (%d picks found)", len(existingPicks))
	}

	// Calculate number of teams from draft order
	numTeams := len(draft.Settings.DraftOrder)
	if numTeams == 0 {
		return fmt.Errorf("draft order is empty, cannot prepopulate picks")
	}

	// Generate all draft picks based on draft type
	var picks []models.DraftPick
	switch draft.DraftType {
	case models.DraftTypeSnake, models.DraftTypeRookie:
		picks = a.generateSnakeDraftPicks(draftID, draft.Settings.Rounds, draft.Settings.DraftOrder, draft.Settings.ThirdRoundReversal)
	case models.DraftTypeAuction:
		picks = a.generateAuctionDraftPicks(draftID, draft.Settings.Rounds, draft.Settings.DraftOrder)
	default:
		return fmt.Errorf("unsupported draft type for prepopulation: %s", draft.DraftType)
	}

	// Create all picks in batch
	if err := a.pickRepo.CreateDraftPicksBatch(ctx, picks); err != nil {
		return fmt.Errorf("failed to create draft picks: %w", err)
	}

	log.Printf("Prepopulated %d draft picks for %s draft (ID: %s)", len(picks), draft.DraftType, draftID)
	return nil
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

func (a *App) MakePick(ctx context.Context, req repository.MakePickRequest) error {
	// Verify draft exists and check status
	_, err := a.repo.GetDraft(ctx, req.DraftID)
	if err != nil {
		return fmt.Errorf("draft not found: %w", err)
	}

	err = a.pickRepo.MakePick(ctx, req)
	if err != nil {
		return err
	}
	return nil
}

// validateCreateDraftRequest validates create draft request
func (a *App) validateCreateDraftRequest(req repository.CreateDraftRequest) error {
	if req.ID == uuid.Nil {
		return fmt.Errorf("id is required")
	}
	if req.LeagueID == uuid.Nil {
		return fmt.Errorf("league_id is required")
	}
	if req.DraftType == "" {
		return fmt.Errorf("draft_type is required")
	}
	if err := a.validateDraftType(req.DraftType); err != nil {
		return err
	}
	if req.Status == "" {
		return fmt.Errorf("status is required")
	}
	if err := a.validateDraftStatus(req.Status); err != nil {
		return err
	}
	return nil
}

// validateDraftType validates draft type
func (a *App) validateDraftType(draftType models.DraftType) error {
	switch draftType {
	case models.DraftTypeSnake, models.DraftTypeAuction, models.DraftTypeRookie:
		return nil
	default:
		return fmt.Errorf("invalid draft type: %s", draftType)
	}
}

// validateDraftStatus validates draft status
func (a *App) validateDraftStatus(status models.DraftStatus) error {
	switch status {
	case models.DraftStatusNotStarted, models.DraftStatusInProgress, models.DraftStatusPaused,
		models.DraftStatusCompleted, models.DraftStatusCancelled:
		return nil
	default:
		return fmt.Errorf("invalid draft status: %s", status)
	}
}

// TODO move to formal FSM
// validateStatusTransition validates if a status transition is allowed
func (a *App) validateStatusTransition(currentStatus, newStatus models.DraftStatus) error {
	// Allow same status (no-op)
	if currentStatus == newStatus {
		return nil
	}

	allowedTransitions := map[models.DraftStatus][]models.DraftStatus{
		models.DraftStatusNotStarted: {models.DraftStatusInProgress, models.DraftStatusCancelled},
		models.DraftStatusInProgress: {models.DraftStatusPaused, models.DraftStatusCompleted, models.DraftStatusCancelled},
		models.DraftStatusPaused:     {models.DraftStatusInProgress, models.DraftStatusCancelled},
		models.DraftStatusCompleted:  {}, // No transitions allowed from completed
		models.DraftStatusCancelled:  {}, // No transitions allowed from cancelled
	}

	allowedNext, exists := allowedTransitions[currentStatus]
	if !exists {
		return fmt.Errorf("unknown current status: %s", currentStatus)
	}

	for _, allowed := range allowedNext {
		if newStatus == allowed {
			return nil
		}
	}

	return fmt.Errorf("transition from %s to %s is not allowed", currentStatus, newStatus)
}

// validateDraftSettings validates draft settings based on draft type
func (a *App) validateDraftSettings(draftType models.DraftType, settings models.DraftSettings) error {
	// Common validations
	if settings.Rounds <= 0 {
		return fmt.Errorf("rounds must be greater than 0")
	}
	if settings.TimePerPickSec < 0 {
		return fmt.Errorf("time_per_pick_sec cannot be negative")
	}

	// Type-specific validations
	switch draftType {
	case models.DraftTypeAuction:
		if settings.BudgetPerTeam == nil || *settings.BudgetPerTeam <= 0 {
			return fmt.Errorf("budget_per_team is required and must be greater than 0 for auction drafts")
		}
		if settings.MinBidIncrement == nil || *settings.MinBidIncrement <= 0 {
			return fmt.Errorf("min_bid_increment is required and must be greater than 0 for auction drafts")
		}
		if settings.TimePerNominationSec == nil || *settings.TimePerNominationSec < 0 {
			return fmt.Errorf("time_per_nomination_sec is required and cannot be negative for auction drafts")
		}

	case models.DraftTypeSnake:
		// Snake drafts require draft order to be set
		if len(settings.DraftOrder) == 0 {
			return fmt.Errorf("draft_order is required for snake drafts")
		}

	case models.DraftTypeRookie:
		// Rookie drafts are similar to snake but typically shorter
		if len(settings.DraftOrder) == 0 {
			return fmt.Errorf("draft_order is required for rookie drafts")
		}
		if settings.Rounds > 5 {
			return fmt.Errorf("rookie drafts typically have 5 or fewer rounds")
		}
	}

	return nil
}

// FetchNextDeadline retrieves the next draft deadline across all active drafts
func (a *App) FetchNextDeadline(ctx context.Context) (*repository.NextDeadline, error) {
	deadline, err := a.repo.FetchNextDeadline(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch next deadline: %w", err)
	}
	return deadline, nil
}

// FetchDraftsDueForPick retrieves drafts that have exceeded their pick deadline
func (a *App) FetchDraftsDueForPick(ctx context.Context, limit int32) ([]uuid.UUID, error) {
	if limit <= 0 {
		return nil, fmt.Errorf("limit must be greater than 0")
	}

	draftIDs, err := a.repo.FetchDraftsDueForPick(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch drafts due for pick: %w", err)
	}

	return draftIDs, nil
}

// UpdateNextDeadline updates the deadline for when the next pick should be made
func (a *App) UpdateNextDeadline(ctx context.Context, draftID uuid.UUID, deadline *time.Time) error {
	// Verify draft exists and is in progress
	draft, err := a.repo.GetDraft(ctx, draftID)
	if err != nil {
		return fmt.Errorf("draft not found: %w", err)
	}

	// Only allow deadline updates for drafts in progress
	if draft.Status != models.DraftStatusInProgress {
		return fmt.Errorf("can only update deadline for drafts with status %s, current status is %s",
			models.DraftStatusInProgress, draft.Status)
	}

	if err := a.repo.UpdateNextDeadline(ctx, draftID, deadline); err != nil {
		return fmt.Errorf("failed to update next deadline: %w", err)
	}

	return nil
}

// ClearNextDeadline removes the deadline for a draft (used when pausing or completing)
func (a *App) ClearNextDeadline(ctx context.Context, draftID uuid.UUID) error {
	// Verify draft exists
	draft, err := a.repo.GetDraft(ctx, draftID)
	if err != nil {
		return fmt.Errorf("draft not found: %w", err)
	}

	// Only clear deadline for paused, completed, or cancelled drafts
	switch draft.Status {
	case models.DraftStatusPaused, models.DraftStatusCompleted, models.DraftStatusCancelled:
		// Valid states for clearing deadline
	default:
		return fmt.Errorf("can only clear deadline for drafts with status PAUSED, COMPLETED, or CANCELLED, current status is %s",
			draft.Status)
	}

	if err := a.repo.ClearNextDeadline(ctx, draftID); err != nil {
		return fmt.Errorf("failed to clear next deadline: %w", err)
	}
	return nil
}

// CountRemainingPicks returns the number of unpicked slots in a draft
func (a *App) CountRemainingPicks(ctx context.Context, draftID uuid.UUID) (int, error) {
	// Verify draft exists
	if _, err := a.repo.GetDraft(ctx, draftID); err != nil {
		return 0, fmt.Errorf("draft not found: %w", err)
	}

	count, err := a.pickRepo.CountRemainingPicks(ctx, draftID)
	if err != nil {
		return 0, fmt.Errorf("failed to count remaining picks: %w", err)
	}
	return count, nil
}

// ClaimNextPickSlot atomically claims the next available pick slot for auto-pick
func (a *App) ClaimNextPickSlot(ctx context.Context, draftID uuid.UUID) (*repository.Slot, error) {
	// Verify draft exists and is in progress
	draft, err := a.repo.GetDraft(ctx, draftID)
	if err != nil {
		return nil, fmt.Errorf("draft not found: %w", err)
	}

	if draft.Status != models.DraftStatusInProgress {
		return nil, fmt.Errorf("can only claim pick slots for drafts with status %s, current status is %s",
			models.DraftStatusInProgress, draft.Status)
	}
	slot, err := a.pickRepo.ClaimNextPickSlot(ctx, draftID)
	if err != nil {
		return nil, fmt.Errorf("failed to claim next pick slot: %w", err)
	}
	return slot, nil
}

// ListAvailablePlayersForDraft returns all players not yet picked in the specified draft
func (a *App) ListAvailablePlayersForDraft(ctx context.Context, draftID uuid.UUID) ([]repository.AvailablePlayer, error) {
	// Verify draft exists
	if _, err := a.repo.GetDraft(ctx, draftID); err != nil {
		return nil, fmt.Errorf("draft not found: %w", err)
	}

	players, err := a.repo.ListAvailablePlayersForDraft(ctx, draftID)
	if err != nil {
		return nil, fmt.Errorf("failed to list available players for draft: %w", err)
	}

	return players, nil
}

// GetNextPickForDraft returns the next pick that needs to be made for a draft
func (a *App) GetNextPickForDraft(ctx context.Context, draftID uuid.UUID) (*models.DraftPick, error) {
	// Verify draft exists
	if _, err := a.repo.GetDraft(ctx, draftID); err != nil {
		return nil, fmt.Errorf("draft not found: %w", err)
	}

	pick, err := a.pickRepo.GetNextPickForDraft(ctx, draftID)
	if err != nil {
		return nil, fmt.Errorf("failed to get next pick for draft: %w", err)
	}

	return pick, nil
}

// InsertOutboxPickStarted inserts a PickStarted event into the outbox
func (a *App) InsertOutboxPickStarted(ctx context.Context, draftID uuid.UUID, payload []byte) error {
	if err := a.pickRepo.InsertOutboxPickStarted(ctx, draftID, payload); err != nil {
		return fmt.Errorf("failed to insert PickStarted outbox event: %w", err)
	}
	return nil
}
