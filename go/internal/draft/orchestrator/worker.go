package orchestrator

import (
	"context"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/jonboulle/clockwork"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/rs/zerolog/log"
)

// RunScheduler runs the event-driven orchestrator as a JetStream consumer.
// Recovery happens automatically through JetStream event replay with DeliverAllPolicy.
func (o *Orchestrator) RunScheduler(ctx context.Context) error {
	log.Info().
		Str("instance", o.instanceID).
		Int("workers", o.numWorkers).
		Msg("event-driven orchestrator started as JetStream consumer")

	// Create event processing channel
	eventCh := make(chan jetstream.Msg, eventChannelBufferSize)

	// Start JetStream consumer
	consumeCtx, err := o.consumer.Consume(func(msg jetstream.Msg) {
		select {
		case eventCh <- msg:
		case <-ctx.Done():
			msg.Nak()
		}
	})
	if err != nil {
		return fmt.Errorf("start JetStream consumer: %w", err)
	}
	defer consumeCtx.Stop()

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

	// Process events and handle shutdown
	for {
		select {
		case <-ctx.Done():
			log.Info().Str("instance", o.instanceID).Msg("orchestrator shutdown requested")
			goto shutdown
		case msg := <-eventCh:
			if err := o.processEvent(ctx, msg); err != nil {
				log.Error().Err(err).Msg("failed to process event")
				msg.Nak()
			} else {
				msg.Ack()
			}
		}
	}

shutdown:

	// Cancel any remaining active timers using bullet-proof cancellation
	o.activeTimersMu.Lock()
	for draftID, timer := range o.activeTimers {
		stopAndDrainTimer(timer)
		log.Debug().Str("draft_id", draftID.String()).Msg("cancelled timer on shutdown")
	}
	o.activeTimers = make(map[uuid.UUID]clockwork.Timer) // Clear the map
	o.activeTimersMu.Unlock()

	return nil
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
		}
	}
}