package draft

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/mcdev12/dynasty/go/internal/draft/db"
	"github.com/mcdev12/dynasty/go/internal/models"
)

type DraftPickQuerier interface {
	CreateDraftPick(ctx context.Context, arg db.CreateDraftPickParams) (db.DraftPick, error)
	CreateDraftPickBatch(ctx context.Context, arg db.CreateDraftPickBatchParams) error
	GetDraftPick(ctx context.Context, id uuid.UUID) (db.DraftPick, error)
	GetDraftPicksByDraft(ctx context.Context, draftID uuid.UUID) ([]db.DraftPick, error)
	GetDraftPicksByRound(ctx context.Context, arg db.GetDraftPicksByRoundParams) ([]db.DraftPick, error)
	GetNextPickForDraft(ctx context.Context, draftID uuid.UUID) (db.DraftPick, error)
	UpdateDraftPickPlayer(ctx context.Context, arg db.UpdateDraftPickPlayerParams) (db.DraftPick, error)
	DeleteDraftPicksByDraft(ctx context.Context, draftID uuid.UUID) error
}

type DraftPickRepository struct {
	queries DraftPickQuerier
}

func NewDraftPickRepository(querier DraftPickQuerier) *DraftPickRepository {
	return &DraftPickRepository{
		queries: querier,
	}
}

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

func (r *DraftPickRepository) CreateDraftPick(ctx context.Context, req CreateDraftPickRequest) (*models.DraftPick, error) {
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

	pick, err := r.queries.CreateDraftPick(ctx, db.CreateDraftPickParams{
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

	return r.dbDraftPickToModel(pick), nil
}

func (r *DraftPickRepository) CreateDraftPicksBatch(ctx context.Context, picks []models.DraftPick) error {
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

	err := r.queries.CreateDraftPickBatch(ctx, db.CreateDraftPickBatchParams{
		Column1: ids,        // $1::uuid[]
		Column2: draftIDs,   // $2::uuid[]
		Column3: rounds,     // $3::integer[]
		Column4: pickNumbers, // $4::integer[]
		Column5: overallPicks, // $5::integer[]
		Column6: teamIDs,    // $6::uuid[]
	})
	if err != nil {
		return fmt.Errorf("failed to create draft picks batch: %w", err)
	}

	return nil
}

func (r *DraftPickRepository) GetDraftPick(ctx context.Context, id uuid.UUID) (*models.DraftPick, error) {
	pick, err := r.queries.GetDraftPick(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get draft pick: %w", err)
	}

	return r.dbDraftPickToModel(pick), nil
}

func (r *DraftPickRepository) GetDraftPicksByDraft(ctx context.Context, draftID uuid.UUID) ([]models.DraftPick, error) {
	picks, err := r.queries.GetDraftPicksByDraft(ctx, draftID)
	if err != nil {
		return nil, fmt.Errorf("failed to get draft picks by draft: %w", err)
	}

	return r.dbDraftPicksToModels(picks), nil
}

func (r *DraftPickRepository) GetDraftPicksByRound(ctx context.Context, draftID uuid.UUID, round int) ([]models.DraftPick, error) {
	picks, err := r.queries.GetDraftPicksByRound(ctx, db.GetDraftPicksByRoundParams{
		DraftID: draftID,
		Round:   int32(round),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get draft picks by round: %w", err)
	}

	return r.dbDraftPicksToModels(picks), nil
}

func (r *DraftPickRepository) GetNextPickForDraft(ctx context.Context, draftID uuid.UUID) (*models.DraftPick, error) {
	pick, err := r.queries.GetNextPickForDraft(ctx, draftID)
	if err != nil {
		return nil, fmt.Errorf("failed to get next pick for draft: %w", err)
	}

	return r.dbDraftPickToModel(pick), nil
}

func (r *DraftPickRepository) UpdateDraftPickPlayer(ctx context.Context, id uuid.UUID, req UpdateDraftPickPlayerRequest) (*models.DraftPick, error) {
	var auctionAmount sql.NullString
	if req.AuctionAmount != nil {
		auctionAmount = sql.NullString{String: fmt.Sprintf("%.2f", *req.AuctionAmount), Valid: true}
	}

	pick, err := r.queries.UpdateDraftPickPlayer(ctx, db.UpdateDraftPickPlayerParams{
		ID:            id,
		PlayerID:      uuid.NullUUID{UUID: req.PlayerID, Valid: true},
		AuctionAmount: auctionAmount,
		KeeperPick:    sql.NullBool{Bool: req.KeeperPick, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update draft pick player: %w", err)
	}

	return r.dbDraftPickToModel(pick), nil
}

func (r *DraftPickRepository) DeleteDraftPicksByDraft(ctx context.Context, draftID uuid.UUID) error {
	if err := r.queries.DeleteDraftPicksByDraft(ctx, draftID); err != nil {
		return fmt.Errorf("failed to delete draft picks by draft: %w", err)
	}
	return nil
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