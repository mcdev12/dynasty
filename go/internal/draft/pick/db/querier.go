// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.29.0

package db

import (
	"context"

	"github.com/google/uuid"
)

type Querier interface {
	ClaimNextPickSlot(ctx context.Context, draftID uuid.UUID) (ClaimNextPickSlotRow, error)
	CountRemainingPicks(ctx context.Context, draftID uuid.UUID) (int64, error)
	CreateDraftPick(ctx context.Context, arg CreateDraftPickParams) (DraftPick, error)
	CreateDraftPickBatch(ctx context.Context, arg CreateDraftPickBatchParams) error
	DeleteDraftPicksByDraft(ctx context.Context, draftID uuid.UUID) error
	GetDraftPick(ctx context.Context, id uuid.UUID) (DraftPick, error)
	GetDraftPicksByDraft(ctx context.Context, draftID uuid.UUID) ([]DraftPick, error)
	GetDraftPicksByRound(ctx context.Context, arg GetDraftPicksByRoundParams) ([]DraftPick, error)
	GetNextPickForDraft(ctx context.Context, draftID uuid.UUID) (DraftPick, error)
	// List all players not yet picked in draft $1, ordered by name.
	ListAvailablePlayersForDraft(ctx context.Context, draftID uuid.UUID) ([]ListAvailablePlayersForDraftRow, error)
	MakePick(ctx context.Context, arg MakePickParams) (int64, error)
	UpdateDraftPickPlayer(ctx context.Context, arg UpdateDraftPickPlayerParams) (DraftPick, error)
}

var _ Querier = (*Queries)(nil)
