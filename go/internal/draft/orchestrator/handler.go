package orchestrator

import (
	"connectrpc.com/connect"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	draftv1 "github.com/mcdev12/dynasty/go/internal/genproto/draft/v1"
	"time"

	"github.com/google/uuid"
	"github.com/mcdev12/dynasty/go/internal/draft/events"
	"github.com/rs/zerolog/log"
)

// DomainEvent represents a domain event from JetStream
type DomainEvent struct {
	EventID   string          `json:"eventId"`
	EventType string          `json:"eventType"`
	DraftID   string          `json:"draftId"`
	Timestamp time.Time       `json:"timestamp"`
	Payload   json.RawMessage `json:"payload"`
}

// HandleDomainEvent handles incoming domain events and routes them to appropriate handlers
func (o *Orchestrator) HandleDomainEvent(ctx context.Context, eventType string, draftID uuid.UUID, payload []byte) error {
	log.Info().
		Str("event_type", eventType).
		Str("draft_id", draftID.String()).
		Msg("handling domain event")

	switch eventType {
	case "DraftStarted":
		var draftStartedPayload events.DraftStartedPayload
		if err := json.Unmarshal(payload, &draftStartedPayload); err != nil {
			return fmt.Errorf("failed to unmarshal DraftStarted payload: %w", err)
		}
		return o.handleDraftStartedEvent(ctx, draftID, draftStartedPayload)

	case "DraftPaused":
		var draftPausedPayload events.DraftPausedPayload
		if err := json.Unmarshal(payload, &draftPausedPayload); err != nil {
			return fmt.Errorf("failed to unmarshal DraftPaused payload: %w", err)
		}
		return o.handleDraftPausedEvent(ctx, draftID, draftPausedPayload)

	case "DraftResumed":
		var draftResumedPayload events.DraftResumedPayload
		if err := json.Unmarshal(payload, &draftResumedPayload); err != nil {
			return fmt.Errorf("failed to unmarshal DraftResumed payload: %w", err)
		}
		return o.handleDraftResumedEvent(ctx, draftID, draftResumedPayload)

	case "PickMade":
		var pickMadePayload events.PickMadePayload
		if err := json.Unmarshal(payload, &pickMadePayload); err != nil {
			return fmt.Errorf("failed to unmarshal PickMade payload: %w", err)
		}
		return o.handlePickMadeEvent(ctx, draftID, pickMadePayload)

	case "DraftCompleted":
		// For DraftCompleted, clean up tracking maps and log completion
		log.Info().
			Str("draft_id", draftID.String()).
			Msg("draft completed - cleaning up tracking maps")

		// Clean up tracking maps to prevent memory leaks
		o.lastScheduledMu.Lock()
		delete(o.lastScheduled, draftID)
		o.lastScheduledMu.Unlock()

		// Cancel any active timer for this draft
		o.cancelTimer(draftID)

		return nil

	default:
		log.Warn().
			Str("event_type", eventType).
			Str("draft_id", draftID.String()).
			Msg("unknown event type - ignoring")
		return nil
	}
}

// handlePickMadeEvent handles a PickMade domain event by scheduling the next timeout
func (o *Orchestrator) handlePickMadeEvent(ctx context.Context, draftID uuid.UUID, pickPayload events.PickMadePayload) error {
	log.Info().
		Str("draft_id", draftID.String()).
		Str("pick_id", pickPayload.PickID).
		Int("overall_pick", pickPayload.OverallPick).
		Msg("handling PickMade event")

	// Schedule next pick timeout using current time as base
	return o.scheduleNextPick(ctx, draftID, o.clock.Now())
}

// handleDraftStartedEvent handles a DraftStarted domain event by setting up the first pick timer
func (o *Orchestrator) handleDraftStartedEvent(ctx context.Context, draftID uuid.UUID, payload events.DraftStartedPayload) error {
	log.Info().
		Str("draft_id", draftID.String()).
		Str("draft_type", payload.DraftType).
		Int("total_picks", payload.TotalPicks).
		Msg("handling DraftStarted event")

	// Schedule first pick timeout using draft start time as base
	return o.scheduleNextPick(ctx, draftID, payload.StartedAt)
}

// handleDraftPausedEvent handles a DraftPaused domain event by cancelling timers
func (o *Orchestrator) handleDraftPausedEvent(ctx context.Context, draftID uuid.UUID, payload events.DraftPausedPayload) error {
	log.Info().
		Str("draft_id", draftID.String()).
		Str("reason", payload.Reason).
		Msg("handling DraftPaused event")

	// Cancel the active timer to stop timeout processing
	o.cancelTimer(draftID)
	return nil
}

// handleDraftResumedEvent handles a DraftResumed domain event by restarting the pick timer
func (o *Orchestrator) handleDraftResumedEvent(ctx context.Context, draftID uuid.UUID, payload events.DraftResumedPayload) error {
	log.Info().
		Str("draft_id", draftID.String()).
		Msg("handling DraftResumed event")

	// Restart the pick timer using resume time as base
	// TODO: should resume from remaining time instead of full timeout
	return o.scheduleNextPick(ctx, draftID, payload.ResumedAt)
}

func (o *Orchestrator) handleTimeout(ctx context.Context, draftID uuid.UUID) error {
	log.Info().Str("draft_id", draftID.String()).Msg("auto-pick timeout firing")

	// 1) Attempt to claim the next slot
	req, err := o.strat.SelectClaim(ctx, draftID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) || err.Error() == "no available slots to claim" {
			// ← no slots left ⇒ finalize the draft now
			return o.finalizeIfComplete(ctx, draftID)
		}
		log.Warn().Err(err).Msg("auto-pick strategy failed")
		return nil
	}

	// 2) We got a slot—record the pick via gRPC (this will emit PickMade event)
	protoReq := &draftv1.MakePickRequest{
		PickId:      req.PickID.String(),
		DraftId:     req.DraftID.String(),
		TeamId:      req.TeamID.String(),
		PlayerId:    req.PlayerID.String(),
		OverallPick: int32(req.OverallPick),
	}
	_, err = o.draftPickService.MakePick(ctx, connect.NewRequest(protoReq))
	if err != nil {
		return fmt.Errorf("auto-pick MakePick failed: %w", err)
	}

	// 3) After every successful pick, check if that was the last one
	return o.finalizeIfComplete(ctx, draftID)
}

func (o *Orchestrator) finalizeIfComplete(ctx context.Context, draftID uuid.UUID) error {
	remResp, err := o.draftPickService.CountRemainingPicks(ctx, connect.NewRequest(&draftv1.CountRemainingPicksRequest{
		DraftId: draftID.String(),
	}))
	if err != nil {
		return err
	}
	rem := int(remResp.Msg.RemainingPicks)
	if rem > 0 {
		return nil
	}

	// Mark draft completed via draft service (this will emit DraftCompleted event)
	completeReq := &draftv1.CompleteDraftRequest{
		DraftId: draftID.String(),
	}
	_, err = o.draftService.CompleteDraft(ctx, connect.NewRequest(completeReq))
	if err != nil {
		return err
	}

	_, err = o.draftService.ClearNextDeadline(ctx, connect.NewRequest(&draftv1.ClearNextDeadlineRequest{
		DraftId: draftID.String(),
	}))
	if err != nil {
		return err
	}
	return nil
}

func (o *Orchestrator) getPickTime(ctx context.Context, draftID uuid.UUID) (time.Duration, error) {
	getReq := &draftv1.GetDraftRequest{
		DraftId: draftID.String(),
	}
	draftResp, err := o.draftService.GetDraft(ctx, connect.NewRequest(getReq))
	if err != nil {
		return 0, err
	}
	draft := draftResp.Msg.Draft

	secs := draft.Settings.TimePerPickSec
	return time.Duration(secs) * time.Second, nil
}
