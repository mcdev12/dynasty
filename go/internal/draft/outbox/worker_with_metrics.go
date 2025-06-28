package outbox

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// WorkerWithMetrics extends Worker with metrics collection
type WorkerWithMetrics struct {
	*Worker
	metrics MetricsCollector
}

func NewWorkerWithMetrics(worker *Worker, metrics MetricsCollector) *WorkerWithMetrics {
	return &WorkerWithMetrics{
		Worker:  worker,
		metrics: metrics,
	}
}

func (w *WorkerWithMetrics) processOutbox(ctx context.Context) {
	start := time.Now()
	
	// Get count of unsent events for lag metric
	// In production, you'd want a separate query for this
	// For now, we'll track it during processing
	
	w.Worker.processOutbox(ctx)
	
	duration := time.Since(start)
	// This would be updated based on actual batch processing
	w.metrics.RecordBatchProcessed(0, duration)
}

func (w *WorkerWithMetrics) publishWithRetry(ctx context.Context, event OutboxEvent) error {
	var lastErr error

	for attempt := 0; attempt <= w.config.MaxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(w.config.RetryDelay * time.Duration(attempt)):
			}
		}

		if err := w.publisher.Publish(ctx, event); err != nil {
			lastErr = err
			w.metrics.RecordPublishAttempt(event.EventType, attempt+1, false)
			w.logger.Warn("failed to publish event, retrying",
				slog.String("event_id", event.ID.String()),
				slog.Int("attempt", attempt+1),
				slog.String("error", err.Error()))
			continue
		}

		w.metrics.RecordPublishAttempt(event.EventType, attempt+1, true)
		return nil
	}

	return fmt.Errorf("failed after %d attempts: %w", w.config.MaxRetries+1, lastErr)
}