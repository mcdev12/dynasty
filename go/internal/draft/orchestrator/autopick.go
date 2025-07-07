package orchestrator

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/mcdev12/dynasty/go/internal/draft/pick"
	draftv1 "github.com/mcdev12/dynasty/go/internal/genproto/draft/v1"
	"github.com/mcdev12/dynasty/go/internal/genproto/draft/v1/draftv1connect"
	"github.com/rs/zerolog/log"
)

type AutoPickStrategy interface {
	// SelectClaim atomically claims the next available slot
	// and returns a MakePickRequest ready for use by the orchestrator.
	SelectClaim(ctx context.Context, draftID uuid.UUID) (pick.MakePickRequest, error)
}

// RandomStrategy uses random choice for the player.
type RandomStrategy struct {
	draftPickService draftv1connect.DraftPickServiceClient
	rng              *rand.Rand
}

// NewRandomStrategy constructs a RandomStrategy with its own seed.
func NewRandomStrategy(draftPickService draftv1connect.DraftPickServiceClient) *RandomStrategy {
	// Create a new Rand with its own Source, seeded once:
	src := rand.NewSource(time.Now().UnixNano())
	return &RandomStrategy{
		draftPickService: draftPickService,
		rng:              rand.New(src),
	}
}

// SelectClaim implements AutoPickStrategy.SelectClaim
func (s *RandomStrategy) SelectClaim(ctx context.Context, draftID uuid.UUID) (pick.MakePickRequest, error) {
	// 2a) List available players via draft pick service
	playersReq := &draftv1.ListAvailablePlayersForDraftRequest{
		DraftId: draftID.String(),
	}
	playersResp, err := s.draftPickService.ListAvailablePlayersForDraft(ctx, connect.NewRequest(playersReq))
	if err != nil {
		return pick.MakePickRequest{}, fmt.Errorf("list players: %w", err)
	}
	if len(playersResp.Msg.Players) == 0 {
		return pick.MakePickRequest{}, fmt.Errorf("no available players")
	}

	// 2b) Choose one at random
	choice := playersResp.Msg.Players[s.rng.Intn(len(playersResp.Msg.Players))]
	// 2c) Atomically claim the next pick slot via draft pick service
	claimReq := &draftv1.ClaimNextPickSlotRequest{
		DraftId: draftID.String(),
	}
	claimResp, err := s.draftPickService.ClaimNextPickSlot(ctx, connect.NewRequest(claimReq))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return pick.MakePickRequest{}, fmt.Errorf("no available slots to claim")
		}
		return pick.MakePickRequest{}, fmt.Errorf("claim slot: %w", err)
	}

	log.Info().
		Str("draft_id", draftID.String()).
		Str("player_id", choice.Id).
		Msg("auto-pick picked player slot")

	// 2d) Build the MakePickRequest for the orchestrator
	pickID, err := uuid.Parse(claimResp.Msg.Slot.PickId)
	if err != nil {
		return pick.MakePickRequest{}, fmt.Errorf("invalid pick ID: %w", err)
	}
	teamID, err := uuid.Parse(claimResp.Msg.Slot.TeamId)
	if err != nil {
		return pick.MakePickRequest{}, fmt.Errorf("invalid team ID: %w", err)
	}
	playerID, err := uuid.Parse(choice.Id)
	if err != nil {
		return pick.MakePickRequest{}, fmt.Errorf("invalid player ID: %w", err)
	}

	return pick.MakePickRequest{
		PickID:      pickID,
		DraftID:     draftID,
		TeamID:      teamID,
		PlayerID:    playerID,
		OverallPick: int(claimResp.Msg.Slot.OverallPick),
	}, nil
}
