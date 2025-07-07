package pick

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/mcdev12/dynasty/go/internal/draft/pick/db"
	"github.com/mcdev12/dynasty/go/internal/models"
)

type Repository struct {
	queries *db.Queries
	sqlDB   *sql.DB
}

func NewRepository(queries *db.Queries, sqlDB *sql.DB) *Repository {
	return &Repository{
		queries: queries,
		sqlDB:   sqlDB,
	}
}


func (r *Repository) CreateDraftPick(ctx context.Context, req CreateDraftPickRequest) (*models.DraftPick, error) {
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
		pickedAt = sql.NullTime{Time: time.Now(), Valid: true}
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

func (r *Repository) CreateDraftPicksBatch(ctx context.Context, picks []models.DraftPick) error {
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

	// Execute batch insert
	err := r.queries.CreateDraftPickBatch(ctx, db.CreateDraftPickBatchParams{
		Column1: ids,
		Column2: draftIDs,
		Column3: rounds,
		Column4: pickNumbers,
		Column5: overallPicks,
		Column6: teamIDs,
	})
	if err != nil {
		return fmt.Errorf("failed to batch create draft picks: %w", err)
	}

	return nil
}

func (r *Repository) GetDraftPick(ctx context.Context, id uuid.UUID) (*models.DraftPick, error) {
	pick, err := r.queries.GetDraftPick(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get draft pick: %w", err)
	}

	return r.dbDraftPickToModel(pick), nil
}

func (r *Repository) GetDraftPicksByDraft(ctx context.Context, draftID uuid.UUID) ([]models.DraftPick, error) {
	picks, err := r.queries.GetDraftPicksByDraft(ctx, draftID)
	if err != nil {
		return nil, fmt.Errorf("failed to get draft picks by draft: %w", err)
	}

	result := make([]models.DraftPick, len(picks))
	for i, pick := range picks {
		result[i] = *r.dbDraftPickToModel(pick)
	}

	return result, nil
}

func (r *Repository) GetDraftPicksByRound(ctx context.Context, draftID uuid.UUID, round int) ([]models.DraftPick, error) {
	picks, err := r.queries.GetDraftPicksByRound(ctx, db.GetDraftPicksByRoundParams{
		DraftID: draftID,
		Round:   int32(round),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get draft picks by round: %w", err)
	}

	result := make([]models.DraftPick, len(picks))
	for i, pick := range picks {
		result[i] = *r.dbDraftPickToModel(pick)
	}

	return result, nil
}

func (r *Repository) GetNextPickForDraft(ctx context.Context, draftID uuid.UUID) (*models.DraftPick, error) {
	pick, err := r.queries.GetNextPickForDraft(ctx, draftID)
	if err != nil {
		return nil, fmt.Errorf("failed to get next pick for draft: %w", err)
	}

	return r.dbDraftPickToModel(pick), nil
}

func (r *Repository) UpdateDraftPickPlayer(ctx context.Context, id uuid.UUID, req UpdateDraftPickPlayerRequest) (*models.DraftPick, error) {
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

func (r *Repository) DeleteDraftPicksByDraft(ctx context.Context, draftID uuid.UUID) (int, error) {
	// Use direct SQL execution to get the count of deleted rows
	result, err := r.sqlDB.ExecContext(ctx, "DELETE FROM draft_picks WHERE draft_id = $1", draftID)
	if err != nil {
		return 0, fmt.Errorf("failed to delete draft picks by draft: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected count: %w", err)
	}
	
	return int(rowsAffected), nil
}

func (r *Repository) MakePick(ctx context.Context, req MakePickRequest) error {
	rowsAffected, err := r.queries.MakePick(ctx, db.MakePickParams{
		ID:       req.PickID,
		PlayerID: uuid.NullUUID{UUID: req.PlayerID, Valid: true},
	})
	if err != nil {
		return fmt.Errorf("failed to make pick: %w", err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("pick already made or pick not found")
	}
	return nil
}

func (r *Repository) CountRemainingPicks(ctx context.Context, draftID uuid.UUID) (int, error) {
	count, err := r.queries.CountRemainingPicks(ctx, draftID)
	if err != nil {
		return 0, fmt.Errorf("failed to count remaining picks: %w", err)
	}
	return int(count), nil
}

func (r *Repository) ClaimNextPickSlot(ctx context.Context, draftID uuid.UUID) (*Slot, error) {
	row, err := r.queries.ClaimNextPickSlot(ctx, draftID)
	if err != nil {
		return nil, fmt.Errorf("failed to claim next pick slot: %w", err)
	}

	return &Slot{
		PickID:      row.ID,
		TeamID:      row.TeamID,
		OverallPick: int(row.OverallPick),
	}, nil
}

func (r *Repository) ListAvailablePlayersForDraft(ctx context.Context, draftID uuid.UUID) ([]AvailablePlayer, error) {
	rows, err := r.queries.ListAvailablePlayersForDraft(ctx, draftID)
	if err != nil {
		return nil, fmt.Errorf("failed to list available players for draft: %w", err)
	}

	players := make([]AvailablePlayer, len(rows))
	for i, row := range rows {
		players[i] = AvailablePlayer{
			ID:       row.ID,
			FullName: row.FullName,
			TeamID:   row.TeamID.UUID, // Convert NullUUID to UUID
		}
	}

	return players, nil
}

// Helper function to convert DB draft pick to model
func (r *Repository) dbDraftPickToModel(dbPick db.DraftPick) *models.DraftPick {
	pick := &models.DraftPick{
		ID:          dbPick.ID,
		DraftID:     dbPick.DraftID,
		Round:       int(dbPick.Round),
		Pick:        int(dbPick.Pick),
		OverallPick: int(dbPick.OverallPick),
		TeamID:      dbPick.TeamID,
		KeeperPick:  dbPick.KeeperPick.Bool,
	}

	if dbPick.PlayerID.Valid {
		pick.PlayerID = &dbPick.PlayerID.UUID
	}
	if dbPick.PickedAt.Valid {
		pick.PickedAt = &dbPick.PickedAt.Time
	}
	if dbPick.AuctionAmount.Valid {
		amount, err := strconv.ParseFloat(dbPick.AuctionAmount.String, 64)
		if err == nil {
			pick.AuctionAmount = &amount
		}
	}

	return pick
}