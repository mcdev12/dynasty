package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/mcdev12/dynasty/go/internal/draft/db"
	"github.com/mcdev12/dynasty/go/internal/models"
)

type DraftPickRepository struct {
	queries *db.Queries
	db      *sql.DB
}

func NewDraftPickRepository(queries *db.Queries, db *sql.DB) *DraftPickRepository {
	return &DraftPickRepository{
		queries: queries,
		db:      db,
	}
}

// TODO fix in migration don't accept null player uuids
type CreateDraftPickRequest struct {
	ID            uuid.UUID  `json:"id"`
	DraftID       uuid.UUID  `json:"draft_id"`
	Round         int        `json:"round"`
	Pick          int        `json:"pick"`
	OverallPick   int        `json:"overall_pick"`
	TeamID        uuid.UUID  `json:"team_id"`
	PlayerID      *uuid.UUID `json:"player_id,omitempty"`
	AuctionAmount *float64   `json:"auction_amount,omitempty"`
	KeeperPick    bool       `json:"keeper_pick"`
}

type UpdateDraftPickPlayerRequest struct {
	PlayerID      uuid.UUID `json:"player_id"`
	AuctionAmount *float64  `json:"auction_amount,omitempty"`
	KeeperPick    bool      `json:"keeper_pick"`
}

type MakePickRequest struct {
	PickID      uuid.UUID `json:"pick_id"`
	PlayerID    uuid.UUID `json:"player_id"`
	DraftID     uuid.UUID `json:"draft_id"`
	TeamID      uuid.UUID `json:"team_id"`
	OverallPick int       `json:"overall_pick"`
}

func (dp *DraftPickRepository) CreateDraftPick(ctx context.Context, req CreateDraftPickRequest) (*models.DraftPick, error) {
	var playerID uuid.NullUUID
	if req.PlayerID != nil {
		playerID = uuid.NullUUID{UUID: *req.PlayerID, Valid: true}
	}

	var auctionAmount sql.NullString
	if req.AuctionAmount != nil {
		auctionAmount = sql.NullString{String: fmt.Sprintf("%.2f", *req.AuctionAmount), Valid: true}
	}

	var pickedAt sql.NullTime
	if req.PlayerID != nil {
		// If player is set, set picked_at to now (handled by SQL query)
		pickedAt = sql.NullTime{Valid: false} // Let SQL handle NOW()
	}

	pick, err := dp.queries.CreateDraftPick(ctx, db.CreateDraftPickParams{
		ID:            req.ID,
		DraftID:       req.DraftID,
		Round:         int32(req.Round),
		Pick:          int32(req.Pick),
		OverallPick:   int32(req.OverallPick),
		TeamID:        req.TeamID,
		PlayerID:      playerID,
		PickedAt:      pickedAt,
		AuctionAmount: auctionAmount,
		KeeperPick:    sql.NullBool{Bool: req.KeeperPick, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create draft pick: %w", err)
	}

	return dp.dbDraftPickToModel(pick), nil
}

func (dp *DraftPickRepository) CreateDraftPicksBatch(ctx context.Context, picks []models.DraftPick) error {
	if len(picks) == 0 {
		return nil
	}

	// Prepare arrays for batch insert
	ids := make([]uuid.UUID, len(picks))
	draftIDs := make([]uuid.UUID, len(picks))
	rounds := make([]int32, len(picks))
	pickNumbers := make([]int32, len(picks))
	overallPicks := make([]int32, len(picks))
	teamIDs := make([]uuid.UUID, len(picks))

	for i, pick := range picks {
		ids[i] = pick.ID
		draftIDs[i] = pick.DraftID
		rounds[i] = int32(pick.Round)
		pickNumbers[i] = int32(pick.Pick)
		overallPicks[i] = int32(pick.OverallPick)
		teamIDs[i] = pick.TeamID
	}

	err := dp.queries.CreateDraftPickBatch(ctx, db.CreateDraftPickBatchParams{
		Column1: ids,          // $1::uuid[]
		Column2: draftIDs,     // $2::uuid[]
		Column3: rounds,       // $3::integer[]
		Column4: pickNumbers,  // $4::integer[]
		Column5: overallPicks, // $5::integer[]
		Column6: teamIDs,      // $6::uuid[]
	})
	if err != nil {
		return fmt.Errorf("failed to create draft picks batch: %w", err)
	}

	return nil
}

func (dp *DraftPickRepository) GetDraftPick(ctx context.Context, id uuid.UUID) (*models.DraftPick, error) {
	pick, err := dp.queries.GetDraftPick(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get draft pick: %w", err)
	}

	return dp.dbDraftPickToModel(pick), nil
}

func (dp *DraftPickRepository) GetDraftPicksByDraft(ctx context.Context, draftID uuid.UUID) ([]models.DraftPick, error) {
	picks, err := dp.queries.GetDraftPicksByDraft(ctx, draftID)
	if err != nil {
		return nil, fmt.Errorf("failed to get draft picks by draft: %w", err)
	}

	return dp.dbDraftPicksToModels(picks), nil
}

func (dp *DraftPickRepository) GetDraftPicksByRound(ctx context.Context, draftID uuid.UUID, round int) ([]models.DraftPick, error) {
	picks, err := dp.queries.GetDraftPicksByRound(ctx, db.GetDraftPicksByRoundParams{
		DraftID: draftID,
		Round:   int32(round),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get draft picks by round: %w", err)
	}

	return dp.dbDraftPicksToModels(picks), nil
}

func (dp *DraftPickRepository) GetNextPickForDraft(ctx context.Context, draftID uuid.UUID) (*models.DraftPick, error) {
	pick, err := dp.queries.GetNextPickForDraft(ctx, draftID)
	if err != nil {
		return nil, fmt.Errorf("failed to get next pick for draft: %w", err)
	}

	return dp.dbDraftPickToModel(pick), nil
}

func (dp *DraftPickRepository) UpdateDraftPickPlayer(ctx context.Context, id uuid.UUID, req UpdateDraftPickPlayerRequest) (*models.DraftPick, error) {
	var auctionAmount sql.NullString
	if req.AuctionAmount != nil {
		auctionAmount = sql.NullString{String: fmt.Sprintf("%.2f", *req.AuctionAmount), Valid: true}
	}

	pick, err := dp.queries.UpdateDraftPickPlayer(ctx, db.UpdateDraftPickPlayerParams{
		ID:            id,
		PlayerID:      uuid.NullUUID{UUID: req.PlayerID, Valid: true},
		AuctionAmount: auctionAmount,
		KeeperPick:    sql.NullBool{Bool: req.KeeperPick, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update draft pick player: %w", err)
	}

	return dp.dbDraftPickToModel(pick), nil
}

func (r *DraftPickRepository) DeleteDraftPicksByDraft(ctx context.Context, draftID uuid.UUID) error {
	if err := r.queries.DeleteDraftPicksByDraft(ctx, draftID); err != nil {
		return fmt.Errorf("failed to delete draft picks by draft: %w", err)
	}
	return nil
}

// MakePick creates a txn and does a dual write to the draft pick table and the outbox.
// The outbox is then responsible for emitting events to our worker.
func (dp *DraftPickRepository) MakePick(ctx context.Context, pickRequest MakePickRequest) error {

	txn, err := dp.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if err != nil {
			_ = txn.Rollback()
		}
	}()

	tctx := db.New(txn)

	// Update draft pick
	rows, err := tctx.MakePick(ctx, db.MakePickParams{
		ID:       pickRequest.PickID,
		PlayerID: uuid.NullUUID{UUID: pickRequest.PlayerID, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("update pick: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("error already picked")
	}

	// 2) INSERT draft_outbox
	payload, err := json.Marshal(struct {
		PickID, PlayerID, TeamID uuid.UUID
		Overall                  int32
	}{pickRequest.PickID, pickRequest.PlayerID, pickRequest.TeamID, int32(pickRequest.OverallPick)})
	if err != nil {
		return fmt.Errorf("marshal pick: %w", err)
	}

	err = tctx.InsertOutboxPickMade(ctx, db.InsertOutboxPickMadeParams{
		ID:      uuid.New(),
		DraftID: pickRequest.DraftID,
		Payload: payload,
	})
	if err != nil {
		return fmt.Errorf("insert outbox pick made: %w", err)
	}

	// Commit the transaction
	if err := txn.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (dp *DraftPickRepository) CountRemainingPicks(ctx context.Context, draftID uuid.UUID) (int, error) {
	remaining, err := dp.queries.CountRemainingPicks(ctx, draftID)
	if err != nil {
		return 0, fmt.Errorf("failed to count remaining picks: %w", err)
	}
	return int(remaining), nil
}

type Slot struct {
	PickID      uuid.UUID `json:"pick_id"`
	TeamID      uuid.UUID `json:"team_id"`
	OverallPick int       `json:"overall_pick"`
}

func (dp *DraftPickRepository) ClaimNextPickSlot(ctx context.Context, draftID uuid.UUID) (*Slot, error) {
	slot, err := dp.queries.ClaimNextPickSlot(ctx, draftID)
	if err != nil {
		return nil, fmt.Errorf("failed to claim next pick slot: %w", err)
	}
	return &Slot{
		PickID:      slot.ID,
		TeamID:      slot.TeamID,
		OverallPick: int(slot.OverallPick),
	}, nil
}

// InsertOutboxPickStarted inserts a PickStarted event into the outbox
func (dp *DraftPickRepository) InsertOutboxPickStarted(ctx context.Context, draftID uuid.UUID, payload []byte) error {
	return dp.queries.InsertOutboxPickStarted(ctx, db.InsertOutboxPickStartedParams{
		ID:      uuid.New(),
		DraftID: draftID,
		Payload: payload,
	})
}

// InsertOutboxDraftStarted inserts a DraftStarted event into the outbox
func (dp *DraftPickRepository) InsertOutboxDraftStarted(ctx context.Context, draftID uuid.UUID, payload []byte) error {
	return dp.queries.InsertOutboxDraftStarted(ctx, db.InsertOutboxDraftStartedParams{
		ID:      uuid.New(),
		DraftID: draftID,
		Payload: payload,
	})
}

// InsertOutboxDraftCompleted inserts a DraftCompleted event into the outbox
func (dp *DraftPickRepository) InsertOutboxDraftCompleted(ctx context.Context, draftID uuid.UUID, payload []byte) error {
	return dp.queries.InsertOutboxDraftCompleted(ctx, db.InsertOutboxDraftCompletedParams{
		ID:      uuid.New(),
		DraftID: draftID,
		Payload: payload,
	})
}

// InsertOutboxDraftPaused inserts a DraftPaused event into the outbox
func (dp *DraftPickRepository) InsertOutboxDraftPaused(ctx context.Context, draftID uuid.UUID, payload []byte) error {
	return dp.queries.InsertOutboxDraftPaused(ctx, db.InsertOutboxDraftPausedParams{
		ID:      uuid.New(),
		DraftID: draftID,
		Payload: payload,
	})
}

// InsertOutboxDraftResumed inserts a DraftResumed event into the outbox
func (dp *DraftPickRepository) InsertOutboxDraftResumed(ctx context.Context, draftID uuid.UUID, payload []byte) error {
	return dp.queries.InsertOutboxDraftResumed(ctx, db.InsertOutboxDraftResumedParams{
		ID:      uuid.New(),
		DraftID: draftID,
		Payload: payload,
	})
}

func (r *DraftPickRepository) dbDraftPickToModel(dbPick db.DraftPick) *models.DraftPick {
	var playerID *uuid.UUID
	if dbPick.PlayerID.Valid {
		playerID = &dbPick.PlayerID.UUID
	}

	var pickedAt *time.Time
	if dbPick.PickedAt.Valid {
		pickedAt = &dbPick.PickedAt.Time
	}

	var auctionAmount *float64
	if dbPick.AuctionAmount.Valid {
		// Parse the string back to float64
		if amount, err := strconv.ParseFloat(dbPick.AuctionAmount.String, 64); err == nil {
			auctionAmount = &amount
		}
	}

	keeperPick := false
	if dbPick.KeeperPick.Valid {
		keeperPick = dbPick.KeeperPick.Bool
	}

	return &models.DraftPick{
		ID:            dbPick.ID,
		DraftID:       dbPick.DraftID,
		Round:         int(dbPick.Round),
		Pick:          int(dbPick.Pick),
		OverallPick:   int(dbPick.OverallPick),
		TeamID:        dbPick.TeamID,
		PlayerID:      playerID,
		PickedAt:      pickedAt,
		AuctionAmount: auctionAmount,
		KeeperPick:    keeperPick,
	}
}

func (r *DraftPickRepository) dbDraftPicksToModels(dbPicks []db.DraftPick) []models.DraftPick {
	picks := make([]models.DraftPick, len(dbPicks))
	for i, dbPick := range dbPicks {
		picks[i] = *r.dbDraftPickToModel(dbPick)
	}
	return picks
}
