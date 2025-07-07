package orchestrator

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jonboulle/clockwork"
	"github.com/rs/zerolog/log"
)

// scheduleNextPick is a helper method that handles the common pattern of scheduling a pick timeout.
// It fetches the timeout duration, calculates the next deadline, and sets up a timer.
// Includes single-layer idempotency guard to prevent duplicate scheduling operations.
func (o *Orchestrator) scheduleNextPick(ctx context.Context, draftID uuid.UUID, baseTime time.Time) error {
	// Base-time idempotency guard - prevent duplicate timers with same baseTime
	o.lastScheduledMu.Lock()
	if lastBase, exists := o.lastScheduled[draftID]; exists && lastBase.Equal(baseTime) {
		o.lastScheduledMu.Unlock()
		log.Debug().
			Str("draft_id", draftID.String()).
			Time("base_time", baseTime).
			Msg("skipping duplicate schedule - already scheduled for this exact baseTime")
		return nil
	}
	o.lastScheduled[draftID] = baseTime
	o.lastScheduledMu.Unlock()

	// Get pick timeout duration from draft settings
	timeOut, err := o.getPickTime(ctx, draftID)
	if err != nil {
		return fmt.Errorf("failed to get pick time: %w", err)
	}

	// Calculate next deadline
	next := baseTime.Add(timeOut)

	// Create one-shot timer that will enqueue the draft when it fires
	duration := next.Sub(o.clock.Now())
	if duration > 0 {
		timer := o.clock.NewTimer(duration)
		
		// Atomically replace any existing timer for this draft
		o.replaceTimer(draftID, timer)
		
		// Start goroutine to wait for timer and enqueue work
		go func(id uuid.UUID, t clockwork.Timer) {
			select {
			case <-t.Chan():
				// Timer fired normally - remove from active timers and enqueue
				o.removeTimer(id)
				
				// Clean up lastScheduled entry after timer fires to prevent unbounded growth
				o.lastScheduledMu.Lock()
				delete(o.lastScheduled, id)
				o.lastScheduledMu.Unlock()
				
				select {
				case o.workCh <- id:
					log.Debug().Str("draft_id", id.String()).Msg("timer fired - enqueued for processing")
				default:
					log.Warn().Str("draft_id", id.String()).Msg("timer fired but work channel full")
				}
			case <-ctx.Done():
				// Context cancelled - stop timer and clean up
				stopAndDrainTimer(t)
				o.removeTimer(id)
				
				// Clean up lastScheduled entry when cancelled
				o.lastScheduledMu.Lock()
				delete(o.lastScheduled, id)
				o.lastScheduledMu.Unlock()
				
				log.Debug().Str("draft_id", id.String()).Msg("timer cancelled due to context cancellation")
			}
		}(draftID, timer)

		log.Debug().
			Str("draft_id", draftID.String()).
			Time("deadline", next).
			Dur("duration", duration).
			Msg("scheduled one-shot timer")
	}

	return nil
}

// replaceTimer atomically replaces a timer for a draft, properly cancelling any existing timer.
// This prevents race conditions where a new timer could slip in between Stop() and delete().
func (o *Orchestrator) replaceTimer(draftID uuid.UUID, newTimer clockwork.Timer) {
	o.activeTimersMu.Lock()
	defer o.activeTimersMu.Unlock()

	// Cancel any existing timer first
	if existingTimer, exists := o.activeTimers[draftID]; exists {
		stopAndDrainTimer(existingTimer)
		log.Debug().Str("draft_id", draftID.String()).Msg("replaced existing timer")
	}

	// Store the new timer
	o.activeTimers[draftID] = newTimer
}

// stopAndDrainTimer safely stops a timer and drains its channel to prevent goroutine leaks.
// This follows the pattern recommended in the time.Timer.Stop() documentation.
func stopAndDrainTimer(timer clockwork.Timer) {
	if !timer.Stop() {
		// Timer already fired or was stopped, drain the channel to prevent goroutine leaks
		select {
		case <-timer.Chan():
		default:
		}
	}
}

// cancelTimer cancels and removes an active timer for a draft
func (o *Orchestrator) cancelTimer(draftID uuid.UUID) {
	o.activeTimersMu.Lock()
	defer o.activeTimersMu.Unlock()

	if timer, exists := o.activeTimers[draftID]; exists {
		stopAndDrainTimer(timer)
		delete(o.activeTimers, draftID)
		
		// Clean up lastScheduled entry to prevent unbounded growth
		o.lastScheduledMu.Lock()
		delete(o.lastScheduled, draftID)
		o.lastScheduledMu.Unlock()
		
		log.Debug().Str("draft_id", draftID.String()).Msg("cancelled existing timer")
	}
}

// removeTimer removes a timer from the active timers map (called when timer fires)
func (o *Orchestrator) removeTimer(draftID uuid.UUID) {
	o.activeTimersMu.Lock()
	defer o.activeTimersMu.Unlock()
	delete(o.activeTimers, draftID)
}