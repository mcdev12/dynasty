package draft

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/mcdev12/dynasty/go/internal/draft/repository"
	"github.com/rs/zerolog/log"
)

type AutoPickStrategy interface {
	// SelectClaim atomically claims the next available slot
	// and returns a MakePickRequest ready for use by the orchestrator.
	SelectClaim(ctx context.Context, draftID uuid.UUID) (repository.MakePickRequest, error)
}

type PlayerApp interface {
	ListAllAvailablePlayers()
}

// 2) RandomStrategy uses random choice for the player.
type RandomStrategy struct {
	draftApp DraftApp
	rng      *rand.Rand
}

// NewRandomStrategy seeds the RNG and returns a strategy.
// NewRandomStrategy constructs a RandomStrategy with its own seed.
func NewRandomStrategy(draftApp DraftApp) *RandomStrategy {
	// Create a new Rand with its own Source, seeded once:
	src := rand.NewSource(time.Now().UnixNano())
	return &RandomStrategy{
		draftApp: draftApp,
		rng:      rand.New(src),
	}
}

// SelectClaim implements AutoPickStrategy.SelectClaim
func (s *RandomStrategy) SelectClaim(ctx context.Context, draftID uuid.UUID) (repository.MakePickRequest, error) {
	// 2a) List available players
	players, err := s.draftApp.ListAvailablePlayersForDraft(ctx, draftID)
	if err != nil {
		return repository.MakePickRequest{}, fmt.Errorf("list players: %w", err)
	}
	if len(players) == 0 {
		return repository.MakePickRequest{}, fmt.Errorf("no available players")
	}

	log.Debug().Msg("got players")

	// 2b) Choose one at random
	choice := players[rand.Intn(len(players))]

	log.Info().Str("player_id", choice.ID.String()).Msg("auto-pick picked player")

	// 2c) Atomically claim the next pick slot
	slot, err := s.draftApp.ClaimNextPickSlot(ctx, draftID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return repository.MakePickRequest{}, fmt.Errorf("no available slots to claim")
		}
		return repository.MakePickRequest{}, fmt.Errorf("claim slot: %w", err)
	}

	log.Info().
		Str("draft_id", draftID.String()).
		Str("player_id", choice.ID.String()).
		Msg("auto-pick claimed slot")

	// 2d) Build the MakePickRequest for the orchestrator
	return repository.MakePickRequest{
		PickID:      slot.PickID,
		DraftID:     draftID,
		TeamID:      slot.TeamID,
		PlayerID:    choice.ID,
		OverallPick: slot.OverallPick,
	}, nil
}
