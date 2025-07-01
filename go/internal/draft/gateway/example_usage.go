package gateway

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// ExampleGatewayService demonstrates how to use the connection manager
type ExampleGatewayService struct {
	connectionManager *ConnectionManager
	wsHandler         *WebSocketHandler
}

// NewExampleGatewayService creates a new example gateway service
func NewExampleGatewayService() *ExampleGatewayService {
	config := DefaultConnectionConfig()
	cm := NewConnectionManager(config)
	handler := NewWebSocketHandler(cm)

	return &ExampleGatewayService{
		connectionManager: cm,
		wsHandler:         handler,
	}
}

// Start starts the gateway service
func (s *ExampleGatewayService) Start(ctx context.Context) error {
	// Start the connection manager
	go s.connectionManager.Start(ctx)

	// Set up HTTP server for WebSocket endpoints
	mux := http.NewServeMux()
	s.wsHandler.RegisterRoutes(mux)

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	log.Info().Msg("draft gateway WebSocket server starting on :8080")

	// Start server in goroutine
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error().Err(err).Msg("WebSocket server failed")
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()

	// Graceful shutdown
	log.Info().Msg("shutting down draft gateway")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return server.Shutdown(shutdownCtx)
}

// BroadcastEvent broadcasts an event to all clients connected to a draft
func (s *ExampleGatewayService) BroadcastEvent(draftID uuid.UUID, event *DraftEvent) {
	s.connectionManager.BroadcastToDraft(draftID, event)
}

// GetStats returns connection statistics
func (s *ExampleGatewayService) GetStats() map[string]interface{} {
	return s.connectionManager.GetConnectionStats()
}

// Example usage functions for testing

// SimulatePickStartedEvent creates and broadcasts a PickStarted event
func (s *ExampleGatewayService) SimulatePickStartedEvent(draftID uuid.UUID) {
	event := &DraftEvent{
		ID:        uuid.New().String(),
		DraftID:   draftID.String(),
		Type:      EventTypePickStarted,
		Timestamp: time.Now(),
		Data:      []byte(`{"pick_id":"test-pick","team_id":"test-team","time_per_pick_sec":60}`),
	}

	s.BroadcastEvent(draftID, event)
	log.Info().Str("draft_id", draftID.String()).Msg("simulated PickStarted event")
}

// SimulatePickMadeEvent creates and broadcasts a PickMade event
func (s *ExampleGatewayService) SimulatePickMadeEvent(draftID uuid.UUID) {
	event := &DraftEvent{
		ID:        uuid.New().String(),
		DraftID:   draftID.String(),
		Type:      EventTypePickMade,
		Timestamp: time.Now(),
		Data:      []byte(`{"player_name":"Patrick Mahomes","team_name":"Chiefs Dynasty","overall_pick":5}`),
	}

	s.BroadcastEvent(draftID, event)
	log.Info().Str("draft_id", draftID.String()).Msg("simulated PickMade event")
}
