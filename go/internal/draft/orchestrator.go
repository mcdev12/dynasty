package draft

import (
	"context"
	"database/sql"
	"errors"
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
	app       DraftApp // your business logic
	batchSize int32    // how many due picks to claim at once
	clock     Clock

	wakeCh chan struct{}
}

// TODO make this its own? Decouple from this binary?
func NewOrchestrator(app DraftApp, batchSize int32) *Orchestrator {
	return &Orchestrator{
		app:       app,
		batchSize: batchSize,
		clock:     clockwork.NewRealClock(),
		wakeCh:    make(chan struct{}, 1),
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
// TODO probably ahve to kill this its too complex lmao
// TODO figure out completion
func (o *Orchestrator) RunScheduler(ctx context.Context) error {
	log.Info().Msg("scheduler started")
	timer := o.clock.NewTimer(0)
	defer timer.Stop()

	for {
		//log.Debug().Msg("scheduler: fetching next deadline")
		nd, err := o.app.FetchNextDeadline(ctx)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				log.Info().Msg("no in-progress drafts; polling again in 5s")
				select {
				case <-time.After(5 * time.Second):
					continue
				case <-ctx.Done():
					log.Info().Msg("shutdown during idle (no drafts)")
					return nil
				case <-o.wakeCh:
					log.Debug().Msg("woken up")
					continue
				}
			}
			log.Error().Err(err).Msg("error fetching next deadline")
			return err
		}

		if nd.Deadline == nil {
			log.Info().
				Str("draft_id", nd.DraftID.String()).
				Msg("draft exists but no deadline set; polling again in 5s")
			select {
			case <-time.After(5 * time.Second):
				continue
			case <-ctx.Done():
				log.Info().Msg("shutdown during idle (paused/completed)")
				return nil
			case <-o.wakeCh:
				log.Debug().Msg("woken up")
				continue
			}
		}

		wait := nd.Deadline.Sub(o.clock.Now())
		//log.Info().
		//	Str("draft_id", nd.DraftID.String()).
		//	Time("next_deadline", *nd.Deadline).
		//	Dur("will_wait", wait).
		//	Msg("waiting until next deadline")

		if wait > 0 {
			timer.Reset(wait)
			select {
			case <-timer.Chan():
				log.Info().Msg("timer fired — fetching due drafts")
			case <-ctx.Done():
				log.Info().Msg("shutdown during wait")
				return nil
			case <-o.wakeCh:
				log.Debug().Msg("woken up early — new sooner deadline")
				continue
			}
		}

		due, err := o.app.FetchDraftsDueForPick(ctx, o.batchSize)
		if err != nil {
			log.Error().Err(err).Msg("error fetching due drafts")
			return err
		}
		log.Info().
			Int("count_due", len(due)).
			Int32("batch_size", o.batchSize).
			Msg("processing due drafts")

		for _, draftID := range due {
			log.Info().Str("draft_id", draftID.String()).Msg("handling timeout")
			if err := o.handleTimeout(ctx, draftID); err != nil {
				log.Error().
					Err(err).
					Str("draft_id", draftID.String()).
					Msg("timeout handling failed")
			}
		}
	}
}

// TODO implement
func (o *Orchestrator) handleTimeout(ctx context.Context, draftID uuid.UUID) error {
	// 1) Compute the next timeout for this draft
	timeout, err := o.getPickTime(ctx, draftID)
	if err != nil {
		return err
	}

	// 2) Advance the deadline in the DB so we don't spin on the same one
	next := o.clock.Now().Add(timeout)
	if err := o.app.UpdateNextDeadline(ctx, draftID, &next); err != nil {
		return err
	}

	log.Info().
		Str("draft_id", draftID.String()).
		Time("new_deadline", next).
		Msg("advanced deadline on timeout (no-op pick)")

	return nil
}

func (o *Orchestrator) getPickTime(ctx context.Context, draftID uuid.UUID) (time.Duration, error) {
	draft, err := o.app.GetDraft(ctx, draftID)
	if err != nil {
		return 0, err
	}

	secs := draft.Settings.TimePerPickSec
	return time.Duration(secs) * time.Second, nil
}
