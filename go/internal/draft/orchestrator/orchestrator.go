package orchestrator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jonboulle/clockwork"
	"github.com/mcdev12/dynasty/go/internal/genproto/draft/v1/draftv1connect"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

/*
EVENT FLOW:
1. StartDraft gRPC → DraftService updates status → emits DraftStarted → outbox
2. Outbox Relay → publishes DraftStarted to message bus
3. Orchestrator → subscribes to DraftStarted → creates one-shot timer
4. Timer expires → self-enqueues to workCh → Worker makes auto-pick via gRPC → PickMade event → repeat

TIMER FLOW:
- scheduleNextPick() → timer.NewTimer(duration) → goroutine waits → timer fires → workCh <- draftID
- No polling, no database queries for scheduling, only events
*/

const (
	// Worker pool configuration
	defaultNumWorkers       = 10
	workerChannelBufferSize = 20 // 2x numWorkers
	
	// JetStream consumer configuration
	consumerName          = "draft-orchestrator"
	consumerMaxDeliver    = 5
	consumerAckWait       = 30 * time.Second
	consumerMaxAckPending = 100
	
	// Event processing
	eventChannelBufferSize = 100
	
	// NATS connection configuration
	natsMaxReconnects  = -1 // Infinite
	natsReconnectWait  = 2 * time.Second
)

// Clock is the interface we use for time operations.
// In production, use clockwork.NewRealClock(). In tests, a FakeClock.
type Clock interface {
	Now() time.Time
	NewTimer(d time.Duration) clockwork.Timer
}

type Orchestrator struct {
	draftService     draftv1connect.DraftServiceClient
	draftPickService draftv1connect.DraftPickServiceClient
	clock            Clock
	strat            AutoPickStrategy
	instanceID       string // unique ID for this scheduler instance

	// Worker pool configuration
	numWorkers int
	workCh     chan uuid.UUID

	// Track last scheduled baseTime to prevent duplicate timers with same baseTime
	lastScheduled   map[uuid.UUID]time.Time
	lastScheduledMu sync.Mutex

	// Track active timers for cancellation support
	activeTimers   map[uuid.UUID]clockwork.Timer
	activeTimersMu sync.Mutex

	// JetStream connection and consumer
	nc       *nats.Conn
	js       jetstream.JetStream
	consumer jetstream.Consumer
}

// NewOrchestrator creates a new draft orchestrator with JetStream consumer
func NewOrchestrator(draftService draftv1connect.DraftServiceClient, draftPickService draftv1connect.DraftPickServiceClient, strat AutoPickStrategy, natsURL string) (*Orchestrator, error) {
	numWorkers := defaultNumWorkers

	// Connect to NATS with JetStream
	nc, js, err := setupNATSConnection(natsURL)
	if err != nil {
		return nil, err
	}

	orch := &Orchestrator{
		draftService:     draftService,
		draftPickService: draftPickService,
		strat:            strat,
		clock:            clockwork.NewRealClock(),
		instanceID:       uuid.New().String()[:8], // short ID for logging

		numWorkers:    numWorkers,
		workCh:        make(chan uuid.UUID, workerChannelBufferSize),
		lastScheduled: make(map[uuid.UUID]time.Time),
		activeTimers:  make(map[uuid.UUID]clockwork.Timer),

		nc: nc,
		js: js,
	}

	// Set up JetStream consumer
	if err := orch.ensureConsumer(context.Background()); err != nil {
		nc.Close()
		return nil, fmt.Errorf("ensure JetStream consumer: %w", err)
	}

	return orch, nil
}