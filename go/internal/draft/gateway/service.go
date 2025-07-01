package gateway

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"net/http"

	"github.com/rs/zerolog/log"
)

// Service is the main draft gateway service that handles WebSocket connections and event broadcasting
type Service struct {
	connectionManager *ConnectionManager
	wsHandler         *WebSocketHandler
	eventConsumer     *EventConsumer
	stateHandler      *StateHandler
	stateProvider     StateProvider
}

// Config holds configuration for the draft gateway service
type Config struct {
	ConnectionConfig ConnectionConfig
	JetStreamConfig  JetStreamConsumerConfig
}

// DefaultConfig returns default configuration for the draft gateway
func DefaultConfig() Config {
	return Config{
		ConnectionConfig: DefaultConnectionConfig(),
		JetStreamConfig:  DefaultJetStreamConsumerConfig(),
	}
}

// NewService creates a new draft gateway service
func NewService(config Config, stateProvider StateProvider) (*Service, error) {
	// Create connection manager
	connectionManager := NewConnectionManager(config.ConnectionConfig)

	// Create WebSocket handler
	wsHandler := NewWebSocketHandler(connectionManager)

	// Create JetStream event consumer
	eventConsumer, err := NewEventConsumer(connectionManager, config.JetStreamConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create event consumer: %w", err)
	}

	// Create state handler
	stateHandler := NewStateHandler(stateProvider)

	return &Service{
		connectionManager: connectionManager,
		wsHandler:         wsHandler,
		eventConsumer:     eventConsumer,
		stateHandler:      stateHandler,
		stateProvider:     stateProvider,
	}, nil
}

// Start begins the gateway service
func (s *Service) Start(ctx context.Context) error {
	log.Info().Msg("starting draft gateway service")

	// Start connection manager
	go s.connectionManager.Start(ctx)

	// Start JetStream event consumer
	go func() {
		if err := s.eventConsumer.Start(ctx); err != nil {
			log.Error().Err(err).Msg("event consumer failed")
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()

	log.Info().Msg("draft gateway service shutting down")
	return s.Stop()
}

// Stop gracefully shuts down the gateway service
func (s *Service) Stop() error {
	// Stop event consumer
	if err := s.eventConsumer.Stop(); err != nil {
		log.Error().Err(err).Msg("failed to stop event consumer")
	}

	// Connection manager will stop when context is cancelled
	log.Info().Msg("draft gateway service stopped")
	return nil
}

// RegisterRoutes registers the WebSocket HTTP routes
func (s *Service) RegisterRoutes(mux *http.ServeMux) {
	s.wsHandler.RegisterRoutes(mux)
	s.stateHandler.RegisterStateRoutes(mux)
	log.Info().Msg("draft gateway routes registered")
}

// GetStats returns statistics about the gateway service
func (s *Service) GetStats() map[string]interface{} {
	stats := s.connectionManager.GetConnectionStats()
	stats["service"] = "draft_gateway"
	stats["status"] = "running"
	return stats
}

// HandleDraftConnection is a convenience method that delegates to the WebSocket handler
func (s *Service) HandleDraftConnection(w http.ResponseWriter, r *http.Request) {
	s.wsHandler.HandleDraftConnection(w, r)
}

// BroadcastEvent allows manual event broadcasting (useful for testing)
func (s *Service) BroadcastEvent(draftID uuid.UUID, event *DraftEvent) {
	s.connectionManager.BroadcastToDraft(draftID, event)
}

// IntegrationExample shows how to integrate the gateway into the main application
func IntegrationExample() {
	// This would go in your main.go or services.go
	/*
		// In setupServices function:

		// Create draft gateway
		gatewayConfig := gateway.DefaultConfig()
		gatewayConfig.JetStreamConfig.URL = "nats://localhost:4222" // Or from env/config

		// Create state provider using the draft app
		stateProvider := gateway.NewDraftStateProvider(services.Draft.app)

		draftGateway, err := gateway.NewService(gatewayConfig, stateProvider)
		if err != nil {
			log.Fatal().Err(err).Msg("failed to create draft gateway")
		}

		// Add to services struct
		services.DraftGateway = draftGateway

		// In main.go after setting up services:

		// Start draft gateway in background
		go func() {
			if err := services.DraftGateway.Start(ctx); err != nil {
				log.Error().Err(err).Msg("draft gateway failed")
			}
		}()

		// In registerServices function in server.go:

		// Register draft gateway WebSocket routes
		if services.DraftGateway != nil {
			services.DraftGateway.RegisterRoutes(mux)
		}
	*/
}
