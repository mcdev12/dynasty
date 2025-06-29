package outbox

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	draftdb "github.com/mcdev12/dynasty/go/internal/draft/db"
	"github.com/nats-io/nats.go"
)

type HealthStatus struct {
	Healthy           bool
	LastEventTime     time.Time
	EventsProcessed   uint64
	PendingEvents     int
	DatabaseConnected bool
	NATSConnected     bool
	ListenerActive    bool
	Errors            []string
}

type HealthChecker interface {
	Check(ctx context.Context) HealthStatus
}

type RealtimeHealthChecker struct {
	worker    *RealtimeWorker
	db        *sql.DB
	natsConn  *nats.Conn
	queries   *draftdb.Queries
	threshold time.Duration // How long without events before unhealthy
}

func NewRealtimeHealthChecker(worker *RealtimeWorker, db *sql.DB, natsConn *nats.Conn, threshold time.Duration) *RealtimeHealthChecker {
	return &RealtimeHealthChecker{
		worker:    worker,
		db:        db,
		natsConn:  natsConn,
		queries:   draftdb.New(db),
		threshold: threshold,
	}
}

func (h *RealtimeHealthChecker) Check(ctx context.Context) HealthStatus {
	status := HealthStatus{
		Healthy: true,
		Errors:  []string{},
	}

	// Get worker stats
	processed, lastTime := h.worker.Stats()
	status.EventsProcessed = processed
	status.LastEventTime = lastTime

	// Check database connection
	if err := h.db.PingContext(ctx); err != nil {
		status.DatabaseConnected = false
		status.Healthy = false
		status.Errors = append(status.Errors, fmt.Sprintf("database ping failed: %v", err))
	} else {
		status.DatabaseConnected = true
	}

	// Check NATS connection
	if h.natsConn != nil {
		status.NATSConnected = h.natsConn.IsConnected()
		if !status.NATSConnected {
			status.Healthy = false
			status.Errors = append(status.Errors, "NATS disconnected")
		}
	}

	// Check listener (by checking if worker is running)
	h.worker.mu.Lock()
	status.ListenerActive = h.worker.running
	h.worker.mu.Unlock()

	if !status.ListenerActive {
		status.Healthy = false
		status.Errors = append(status.Errors, "listener not active")
	}

	// Check pending events count
	if status.DatabaseConnected {
		pending, err := h.countPendingEvents(ctx)
		if err != nil {
			status.Errors = append(status.Errors, fmt.Sprintf("failed to count pending events: %v", err))
		} else {
			status.PendingEvents = pending
			// Alert if too many pending events
			if pending > 1000 {
				status.Errors = append(status.Errors, fmt.Sprintf("high pending event count: %d", pending))
			}
		}
	}

	// Check if we haven't processed events recently (only if we have pending events)
	if status.PendingEvents > 0 && !status.LastEventTime.IsZero() {
		timeSinceLastEvent := time.Since(status.LastEventTime)
		if timeSinceLastEvent > h.threshold {
			status.Healthy = false
			status.Errors = append(status.Errors, fmt.Sprintf("no events processed for %s", timeSinceLastEvent))
		}
	}

	return status
}

func (h *RealtimeHealthChecker) countPendingEvents(ctx context.Context) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM draft_outbox WHERE sent_at IS NULL`
	err := h.db.QueryRowContext(ctx, query).Scan(&count)
	return count, err
}

// HTTP handler helper
func (h *RealtimeHealthChecker) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	status := h.Check(ctx)

	response := map[string]interface{}{
		"healthy":            status.Healthy,
		"events_processed":   status.EventsProcessed,
		"pending_events":     status.PendingEvents,
		"last_event_time":    status.LastEventTime,
		"database_connected": status.DatabaseConnected,
		"nats_connected":     status.NATSConnected,
		"listener_active":    status.ListenerActive,
		"errors":             status.Errors,
	}

	w.Header().Set("Content-Type", "application/json")

	if !status.Healthy {
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	json.NewEncoder(w).Encode(response)
}

// Metrics exporter for Prometheus
type PrometheusExporter struct {
	checker HealthChecker
}

func NewPrometheusExporter(checker HealthChecker) *PrometheusExporter {
	return &PrometheusExporter{checker: checker}
}

func (e *PrometheusExporter) Export(ctx context.Context) string {
	status := e.checker.Check(ctx)

	healthy := 0
	if status.Healthy {
		healthy = 1
	}

	dbConnected := 0
	if status.DatabaseConnected {
		dbConnected = 1
	}

	natsConnected := 0
	if status.NATSConnected {
		natsConnected = 1
	}

	listenerActive := 0
	if status.ListenerActive {
		listenerActive = 1
	}

	return fmt.Sprintf(`# HELP outbox_healthy Whether the outbox system is healthy
# TYPE outbox_healthy gauge
outbox_healthy %d

# HELP outbox_events_processed_total Total number of events processed
# TYPE outbox_events_processed_total counter
outbox_events_processed_total %d

# HELP outbox_pending_events Current number of pending events
# TYPE outbox_pending_events gauge
outbox_pending_events %d

# HELP outbox_database_connected Whether database is connected
# TYPE outbox_database_connected gauge
outbox_database_connected %d

# HELP outbox_nats_connected Whether NATS is connected
# TYPE outbox_nats_connected gauge
outbox_nats_connected %d

# HELP outbox_listener_active Whether the listener is active
# TYPE outbox_listener_active gauge
outbox_listener_active %d

# HELP outbox_last_event_timestamp Unix timestamp of last processed event
# TYPE outbox_last_event_timestamp gauge
outbox_last_event_timestamp %d
`,
		healthy,
		status.EventsProcessed,
		status.PendingEvents,
		dbConnected,
		natsConnected,
		listenerActive,
		status.LastEventTime.Unix(),
	)
}
