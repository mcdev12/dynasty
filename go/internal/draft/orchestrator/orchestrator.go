package orchestrator

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/jonboulle/clockwork" // optional; import only if you want fake clocks
	"github.com/mcdev12/dynasty/go/internal/draft/events"
	draftv1 "github.com/mcdev12/dynasty/go/internal/genproto/draft/v1"
	"github.com/mcdev12/dynasty/go/internal/genproto/draft/v1/draftv1connect"
	"github.com/rs/zerolog/log"
	"google.golang.org/protobuf/types/known/timestamppb"
)

/*
ORCHESTRATOR REFACTORING STATUS - EVENT-DRIVEN ARCHITECTURE:

✅ COMPLETED EVENT-DRIVEN REFACTORING:
- Orchestrator is now event-driven consumer, not injected into services
- Added HandleDomainEvent() method to route incoming events
- Event handlers: handleDraftStartedEvent, handleDraftPausedEvent, handleDraftResumedEvent, handlePickMadeEvent
- Removed direct service method calls (StartDraft, PauseDraft, ResumeDraft)
- Services now emit domain events via OutboxApp
- Orchestrator consumes events and calls gRPC clients
- Circular dependency eliminated!

NEW ARCHITECTURE:
- DraftService → Pure CRUD + emits domain events to outbox
- DraftPickService → Handle picks + emits PickMade events
- Outbox Relay → DB outbox table → Message Bus (NATS/Kafka)
- Orchestrator → Subscribes to events → Calls gRPC clients → Manages timers
- Clean one-way dependency flow: DB → Bus → Orchestrator → gRPC → DB

EVENT FLOW:
1. StartDraft gRPC → DraftService updates status → emits DraftStarted → outbox
2. Outbox Relay → publishes DraftStarted to message bus
3. Orchestrator → subscribes to DraftStarted → schedules first timeout → emits PickStarted
4. Timer expires → Orchestrator makes auto-pick via gRPC → PickMade event → repeat
*/

// Clock is the interface we use for time operations.
// In production, use clockwork.NewRealClock(). In tests, a FakeClock.
type Clock interface {
	Now() time.Time
	NewTimer(d time.Duration) clockwork.Timer
}

// NextDeadline represents the next deadline for a draft
type NextDeadline struct {
	DraftID  uuid.UUID  `json:"draft_id"`
	Deadline *time.Time `json:"deadline"`
}

// OutboxApp defines what the orchestrator needs from the outbox app
type OutboxApp interface {
	InsertOutboxDraftStarted(ctx context.Context, draftID uuid.UUID, payload []byte) error
	InsertOutboxDraftCompleted(ctx context.Context, draftID uuid.UUID, payload []byte) error
	InsertOutboxDraftPaused(ctx context.Context, draftID uuid.UUID, payload []byte) error
	InsertOutboxDraftResumed(ctx context.Context, draftID uuid.UUID, payload []byte) error
	InsertOutboxPickStarted(ctx context.Context, draftID uuid.UUID, payload []byte) error
	InsertPickMadeEvent(ctx context.Context, draftID uuid.UUID, payload []byte) error
}

type Orchestrator struct {
	draftService     draftv1connect.DraftServiceClient
	draftPickService draftv1connect.DraftPickServiceClient
	outboxApp        OutboxApp
	batchSize        int32 // how many due picks to claim at once
	clock            Clock
	strat            AutoPickStrategy // << add this
	wakeCh           chan struct{}
	instanceID       string // unique ID for this scheduler instance

	// Worker pool configuration
	numWorkers int
	workCh     chan uuid.UUID

	// Track in-flight work to prevent duplicate processing
	inFlight   map[uuid.UUID]bool
	inFlightMu sync.Mutex
}

// NewOrchestrator creates a new draft orchestrator with worker pool
func NewOrchestrator(draftService draftv1connect.DraftServiceClient, draftPickService draftv1connect.DraftPickServiceClient, outboxApp OutboxApp, strat AutoPickStrategy, batchSize int32) *Orchestrator {
	numWorkers := 10 // Start with small pool
	return &Orchestrator{
		draftService:     draftService,
		draftPickService: draftPickService,
		outboxApp:        outboxApp,
		batchSize:        batchSize,
		strat:            strat,
		clock:            clockwork.NewRealClock(),
		wakeCh:           make(chan struct{}, 1),
		instanceID:       uuid.New().String()[:8], // short ID for logging

		numWorkers: numWorkers,
		workCh:     make(chan uuid.UUID, numWorkers*2), // Buffer to prevent blocking
		inFlight:   make(map[uuid.UUID]bool),
	}
}

// scheduleNextPick is a helper method that handles the common pattern of scheduling a pick timeout.
// It fetches the timeout duration, calculates the next deadline, updates it in the database,
// emits a PickStarted event, and wakes the scheduler.
func (o *Orchestrator) scheduleNextPick(ctx context.Context, draftID uuid.UUID, baseTime time.Time) error {
	// Get pick timeout duration from draft settings
	timeOut, err := o.getPickTime(ctx, draftID)
	if err != nil {
		return fmt.Errorf("failed to get pick time: %w", err)
	}

	// Calculate next deadline
	next := baseTime.Add(timeOut)
	
	// Update deadline in database
	if err := o.updateNextDeadline(ctx, draftID, &next); err != nil {
		return fmt.Errorf("failed to update next deadline: %w", err)
	}

	// Emit PickStarted event
	if err := o.emitPickStartedEvent(ctx, draftID, baseTime, next); err != nil {
		log.Error().Err(err).Str("draft_id", draftID.String()).Msg("failed to emit PickStarted event")
		// Don't fail the operation, just log the error
	}

	// Wake the scheduler in case this new deadline is sooner
	select {
	case o.wakeCh <- struct{}{}:
	default:
	}
	
	return nil
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

// updateNextDeadline updates the next deadline via draft service
func (o *Orchestrator) updateNextDeadline(ctx context.Context, draftID uuid.UUID, deadline *time.Time) error {
	updateReq := &draftv1.UpdateNextDeadlineRequest{
		DraftId: draftID.String(),
	}
	if deadline != nil {
		updateReq.Deadline = timestamppb.New(*deadline)
	}
	_, err := o.draftService.UpdateNextDeadline(ctx, connect.NewRequest(updateReq))
	return err
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

// handleDraftPausedEvent handles a DraftPaused domain event by clearing timers
func (o *Orchestrator) handleDraftPausedEvent(ctx context.Context, draftID uuid.UUID, payload events.DraftPausedPayload) error {
	log.Info().
		Str("draft_id", draftID.String()).
		Str("reason", payload.Reason).
		Msg("handling DraftPaused event")

	// Clear the next deadline to stop timeouts
	clearReq := &draftv1.ClearNextDeadlineRequest{
		DraftId: draftID.String(),
	}
	_, err := o.draftService.ClearNextDeadline(ctx, connect.NewRequest(clearReq))
	if err != nil {
		log.Error().Err(err).Str("draft_id", draftID.String()).Msg("failed to clear deadline for paused draft")
		// Don't fail the operation, just log the error
	}

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
		// For DraftCompleted, we just log it - no action needed from orchestrator
		log.Info().
			Str("draft_id", draftID.String()).
			Msg("draft completed - no orchestrator action needed")
		return nil

	default:
		log.Warn().
			Str("event_type", eventType).
			Str("draft_id", draftID.String()).
			Msg("unknown event type - ignoring")
		return nil
	}
}

// RunScheduler loops forever, sleeping until the next deadline and firing timeouts.
func (o *Orchestrator) RunScheduler(ctx context.Context) error {
	log.Info().Str("instance", o.instanceID).Int("workers", o.numWorkers).Msg("scheduler started")

	// Start worker pool
	var wg sync.WaitGroup
	workerCtx, cancelWorkers := context.WithCancel(ctx)
	defer cancelWorkers()

	for i := 0; i < o.numWorkers; i++ {
		wg.Add(1)
		go o.worker(workerCtx, &wg, i)
	}

	// Ensure workers are cleaned up
	defer func() {
		log.Info().Str("instance", o.instanceID).Msg("shutting down workers")
		cancelWorkers()
		close(o.workCh)
		wg.Wait()
		log.Info().Str("instance", o.instanceID).Msg("all workers shut down")
	}()

	timer := o.clock.NewTimer(0)
	defer timer.Stop()

	const idlePollDuration = 5 * time.Second
	retryCount := 0
	const maxRetries = 3

	for {
		select {
		case <-o.wakeCh:
			log.Debug().Str("instance", o.instanceID).Msg("drained wake channel")
		default:
		}

		ndResp, err := o.draftService.FetchNextDeadline(ctx, connect.NewRequest(&draftv1.FetchNextDeadlineRequest{}))
		if err != nil {
			// Handle transient errors with retry
			retryCount++
			if retryCount <= maxRetries {
				log.Error().
					Err(err).
					Int("retry", retryCount).
					Str("instance", o.instanceID).
					Msg("error fetching next deadline, retrying")
				timer.Reset(time.Second * time.Duration(retryCount))
				select {
				case <-timer.Chan():
					continue
				case <-ctx.Done():
					return nil
				}
			}
			log.Error().Err(err).Str("instance", o.instanceID).Msg("error fetching next deadline after retries")
			return err
		}
		retryCount = 0 // Reset on success

		// Process the response and handle no deadline case
		if ndResp.Msg.NextDeadline == nil {
			// No deadline found - idle with timer reuse
			log.Info().Str("instance", o.instanceID).Msg("no in-progress drafts; polling again in 5s")
			timer.Reset(idlePollDuration)
			select {
			case <-timer.Chan():
				continue
			case <-ctx.Done():
				log.Info().Str("instance", o.instanceID).Msg("shutdown during idle (no drafts)")
				return nil
			case <-o.wakeCh:
				log.Debug().Str("instance", o.instanceID).Msg("woken up from idle")
				continue
			}
		}

		// Convert protobuf response to local type
		nd := &NextDeadline{
			DraftID: uuid.MustParse(ndResp.Msg.NextDeadline.DraftId),
		}
		if ndResp.Msg.NextDeadline.Deadline != nil {
			t := ndResp.Msg.NextDeadline.Deadline.AsTime()
			nd.Deadline = &t
		}

		if nd.Deadline == nil {
			// Draft exists but no deadline - idle with timer reuse
			log.Info().
				Str("draft_id", nd.DraftID.String()).
				Str("instance", o.instanceID).
				Msg("draft exists but no deadline set; polling again in 5s")
			timer.Reset(idlePollDuration)
			select {
			case <-timer.Chan():
				continue
			case <-ctx.Done():
				log.Info().Str("instance", o.instanceID).Msg("shutdown during idle (paused/completed)")
				return nil
			case <-o.wakeCh:
				log.Debug().Str("instance", o.instanceID).Msg("woken up from idle")
				continue
			}
		}

		wait := nd.Deadline.Sub(o.clock.Now())
		if wait > 0 {
			timer.Reset(wait)
			select {
			case <-timer.Chan():
				log.Info().Str("instance", o.instanceID).Msg("timer fired — fetching due drafts")
			case <-ctx.Done():
				log.Info().Str("instance", o.instanceID).Msg("shutdown during wait")
				return nil
			case <-o.wakeCh:
				log.Debug().Str("instance", o.instanceID).Msg("woken up early — new sooner deadline")
				continue
			}
		}

		dueResp, err := o.draftService.FetchDraftsDueForPick(ctx, connect.NewRequest(&draftv1.FetchDraftsDueForPickRequest{
			Limit: o.batchSize,
		}))
		if err != nil {
			log.Error().Err(err).Str("instance", o.instanceID).Msg("error fetching due drafts")
			// Don't exit on error - retry next iteration
			timer.Reset(time.Second)
			select {
			case <-timer.Chan():
				continue
			case <-ctx.Done():
				return nil
			}
		}

		// Convert string IDs back to UUIDs
		dueUUIDs := make([]uuid.UUID, len(dueResp.Msg.DraftIds))
		for i, idStr := range dueResp.Msg.DraftIds {
			dueUUIDs[i] = uuid.MustParse(idStr)
		}

		if len(dueUUIDs) > 0 {
			log.Info().
				Int("count_due", len(dueUUIDs)).
				Int32("batch_size", o.batchSize).
				Str("instance", o.instanceID).
				Msg("processing due drafts")

			// Send drafts to worker pool for parallel processing with deduplication
			for _, draftID := range dueUUIDs {
				o.inFlightMu.Lock()
				if o.inFlight[draftID] {
					// Skip if already being processed
					log.Debug().Str("draft_id", draftID.String()).Str("instance", o.instanceID).Msg("skipping draft already in flight")
					o.inFlightMu.Unlock()
					continue
				}
				o.inFlight[draftID] = true
				o.inFlightMu.Unlock()

				select {
				case <-ctx.Done():
					// Clean up in-flight tracking on shutdown
					o.inFlightMu.Lock()
					delete(o.inFlight, draftID)
					o.inFlightMu.Unlock()
					log.Info().Str("instance", o.instanceID).Msg("shutdown while queueing timeouts")
					return nil
				case o.workCh <- draftID:
					log.Debug().Str("draft_id", draftID.String()).Str("instance", o.instanceID).Msg("queued timeout for worker")
				}
			}
		}
	}
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

	// TODO: Add ClearNextDeadline to DraftService protobuf and implementation
	// For now, we'll skip this since it's not exposed in the service layer
	log.Warn().Str("draft_id", draftID.String()).Msg("ClearNextDeadline not yet implemented in service layer")
	return nil // Skip for now
}

// worker processes draft timeouts from the work channel
func (o *Orchestrator) worker(ctx context.Context, wg *sync.WaitGroup, workerID int) {
	defer wg.Done()

	log.Info().
		Str("instance", o.instanceID).
		Int("worker_id", workerID).
		Msg("worker started")

	for {
		select {
		case <-ctx.Done():
			log.Info().
				Str("instance", o.instanceID).
				Int("worker_id", workerID).
				Msg("worker shutting down")
			return
		case draftID, ok := <-o.workCh:
			if !ok {
				log.Info().
					Str("instance", o.instanceID).
					Int("worker_id", workerID).
					Msg("work channel closed, worker shutting down")
				return
			}

			log.Info().
				Str("draft_id", draftID.String()).
				Str("instance", o.instanceID).
				Int("worker_id", workerID).
				Msg("worker handling timeout")

			if err := o.handleTimeout(ctx, draftID); err != nil {
				log.Error().
					Err(err).
					Str("draft_id", draftID.String()).
					Str("instance", o.instanceID).
					Int("worker_id", workerID).
					Msg("worker timeout handling failed")
			}

			// Clean up in-flight tracking regardless of success/failure
			o.inFlightMu.Lock()
			delete(o.inFlight, draftID)
			o.inFlightMu.Unlock()
		}
	}
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

// emitPickStartedEvent emits a PickStarted event to the outbox when a pick timer begins
func (o *Orchestrator) emitPickStartedEvent(ctx context.Context, draftID uuid.UUID, startedAt, timeoutAt time.Time) error {
	// Get the next pick information via draft pick service
	nextPickResp, err := o.draftPickService.GetNextPickForDraft(ctx, connect.NewRequest(&draftv1.GetNextPickForDraftRequest{
		DraftId: draftID.String(),
	}))
	if err != nil {
		return fmt.Errorf("failed to get next pick for PickStarted event: %w", err)
	}
	nextPick := nextPickResp.Msg.Pick

	// Get draft settings for time per pick via draft service
	getDraftReq := &draftv1.GetDraftRequest{
		DraftId: draftID.String(),
	}
	draftResp, err := o.draftService.GetDraft(ctx, connect.NewRequest(getDraftReq))
	if err != nil {
		return fmt.Errorf("failed to get draft for PickStarted event: %w", err)
	}
	draft := draftResp.Msg.Draft

	// Create PickStarted payload
	payload := events.PickStartedPayload{
		PickID:         nextPick.Id,
		TeamID:         nextPick.TeamId,
		Round:          int(nextPick.Round),
		Pick:           int(nextPick.Pick),
		OverallPick:    int(nextPick.OverallPick),
		StartedAt:      startedAt,
		TimeoutAt:      timeoutAt,
		TimePerPickSec: int(draft.Settings.TimePerPickSec),
	}

	// Marshal payload to JSON
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal PickStarted payload: %w", err)
	}

	// Insert into outbox
	return o.outboxApp.InsertOutboxPickStarted(ctx, draftID, payloadBytes)
}
