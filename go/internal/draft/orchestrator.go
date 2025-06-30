package draft

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jonboulle/clockwork" // optional; import only if you want fake clocks
	"github.com/mcdev12/dynasty/go/internal/draft/repository"
	"github.com/mcdev12/dynasty/go/internal/models"
	"github.com/rs/zerolog/log"
)

// Clock is the interface we use for time operations.
// In production, use clockwork.NewRealClock(). In tests, a FakeClock.
type Clock interface {
	Now() time.Time
	NewTimer(d time.Duration) clockwork.Timer
}

type Orchestrator struct {
	app        DraftApp // your business logic
	batchSize  int32    // how many due picks to claim at once
	clock      Clock
	strat      AutoPickStrategy // << add this
	wakeCh     chan struct{}
	instanceID string // unique ID for this scheduler instance

	// Worker pool configuration
	numWorkers int
	workCh     chan uuid.UUID
	
	// Track in-flight work to prevent duplicate processing
	inFlight map[uuid.UUID]bool
	inFlightMu sync.Mutex
}

// NewOrchestrator creates a new draft orchestrator with worker pool
func NewOrchestrator(app DraftApp, strat AutoPickStrategy, batchSize int32) *Orchestrator {
	numWorkers := 10 // Start with small pool
	return &Orchestrator{
		app:        app,
		batchSize:  batchSize,
		strat:      strat,
		clock:      clockwork.NewRealClock(),
		wakeCh:     make(chan struct{}, 1),
		instanceID: uuid.New().String()[:8], // short ID for logging

		numWorkers: numWorkers,
		workCh:     make(chan uuid.UUID, numWorkers*2), // Buffer to prevent blocking
		inFlight:   make(map[uuid.UUID]bool),
	}
}

// MakePick handles the RPC for a user‐made pick, then schedules the next timeout.
func (o *Orchestrator) MakePick(ctx context.Context, req repository.MakePickRequest) error {
	timeOut, err := o.getPickTime(ctx, req.DraftID)
	if err != nil {
		return err
	}

	if err := o.app.MakePick(ctx, req); err != nil {
		return err
	}
	// 2) Schedule next timeout
	next := o.clock.Now().Add(timeOut)
	if err := o.app.UpdateNextDeadline(ctx, req.DraftID, &next); err != nil {
		return err
	}

	// signal the scheduler in case this new deadline is sooner
	select {
	case o.wakeCh <- struct{}{}:
	default:
	}
	return nil
}

// StartDraft starts the draft and sets a new deadline.
func (o *Orchestrator) StartDraft(ctx context.Context, draftID uuid.UUID) error {
	_, err := o.app.UpdateDraftStatus(ctx, draftID, repository.UpdateDraftStatusRequest{Status: models.DraftStatusInProgress})
	if err != nil {
		return err
	}

	timeOut, err := o.getPickTime(ctx, draftID)
	if err != nil {
		return err
	}
	next := o.clock.Now().Add(timeOut)

	if err := o.app.UpdateNextDeadline(ctx, draftID, &next); err != nil {
		return err
	}

	// wake the scheduler
	select {
	case o.wakeCh <- struct{}{}:
	default:
	}
	return nil
}

// PauseDraft pauses a draft and clears its deadline.
func (o *Orchestrator) PauseDraft(ctx context.Context, draftID uuid.UUID) error {
	_, err := o.app.UpdateDraftStatus(ctx, draftID, repository.UpdateDraftStatusRequest{Status: models.DraftStatusPaused})
	if err != nil {
		return err
	}
	return o.app.ClearNextDeadline(ctx, draftID)
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
		// TODO do this in a loop Drain wake channel to prevent tight loops
		select {
		case <-o.wakeCh:
			log.Debug().Str("instance", o.instanceID).Msg("drained wake channel")
		default:
		}

		nd, err := o.app.FetchNextDeadline(ctx)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				// No drafts in progress - idle with timer reuse
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

		due, err := o.app.FetchDraftsDueForPick(ctx, o.batchSize)
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

		if len(due) > 0 {
			log.Info().
				Int("count_due", len(due)).
				Int32("batch_size", o.batchSize).
				Str("instance", o.instanceID).
				Msg("processing due drafts")

			// Send drafts to worker pool for parallel processing with deduplication
			for _, draftID := range due {
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

	// 2) We got a slot—record the pick and schedule the next deadline
	if err := o.MakePick(ctx, req); err != nil {
		return fmt.Errorf("auto-pick MakePick failed: %w", err)
	}

	// 3) After every successful pick, check if that was the last one
	return o.finalizeIfComplete(ctx, draftID)
}

func (o *Orchestrator) finalizeIfComplete(ctx context.Context, draftID uuid.UUID) error {
	rem, _ := o.app.CountRemainingPicks(ctx, draftID)
	if rem > 0 {
		return nil
	}
	// Mark draft completed and clear any deadline
	_, err := o.app.UpdateDraftStatus(ctx, draftID, repository.UpdateDraftStatusRequest{
		Status: models.DraftStatusCompleted,
	})
	if err != nil {
		return err
	}
	return o.app.ClearNextDeadline(ctx, draftID)

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
	draft, err := o.app.GetDraft(ctx, draftID)
	if err != nil {
		return 0, err
	}

	secs := draft.Settings.TimePerPickSec
	return time.Duration(secs) * time.Second, nil
}
