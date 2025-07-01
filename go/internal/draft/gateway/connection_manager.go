package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"
)

// ConnectionManager manages WebSocket connections for draft events
type ConnectionManager struct {
	// Connection pools organized by draft ID
	draftConnections map[uuid.UUID]map[*Connection]bool
	mu               sync.RWMutex

	// Upgrader for WebSocket connections
	upgrader websocket.Upgrader

	// Connection configuration
	config ConnectionConfig

	// Event broadcasting
	broadcastCh chan BroadcastMessage
}

// Connection represents a WebSocket connection to a client
type Connection struct {
	ID       string
	UserID   string
	DraftID  uuid.UUID
	Conn     *websocket.Conn
	Send     chan []byte
	Manager  *ConnectionManager
	
	// Connection metadata
	ConnectedAt time.Time
	LastPing    time.Time
}

// ConnectionConfig holds configuration for WebSocket connections
type ConnectionConfig struct {
	WriteTimeout     time.Duration
	ReadTimeout      time.Duration
	PingInterval     time.Duration
	MaxMessageSize   int64
	ReadBufferSize   int
	WriteBufferSize  int
	CheckOrigin      func(r *http.Request) bool
}

// BroadcastMessage represents a message to broadcast to connections
type BroadcastMessage struct {
	DraftID uuid.UUID
	Event   *DraftEvent
	UserID  string // Optional: if set, only send to this user
}

// DefaultConnectionConfig returns default WebSocket configuration
func DefaultConnectionConfig() ConnectionConfig {
	return ConnectionConfig{
		WriteTimeout:    10 * time.Second,
		ReadTimeout:     60 * time.Second,
		PingInterval:    30 * time.Second,
		MaxMessageSize:  1024, // 1KB max message size
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			// Allow all origins in development - restrict in production
			return true
		},
	}
}

// NewConnectionManager creates a new WebSocket connection manager
func NewConnectionManager(config ConnectionConfig) *ConnectionManager {
	cm := &ConnectionManager{
		draftConnections: make(map[uuid.UUID]map[*Connection]bool),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  config.ReadBufferSize,
			WriteBufferSize: config.WriteBufferSize,
			CheckOrigin:     config.CheckOrigin,
		},
		config:      config,
		broadcastCh: make(chan BroadcastMessage, 1000), // Buffer for high throughput
	}

	return cm
}

// Start begins processing broadcast messages
func (cm *ConnectionManager) Start(ctx context.Context) {
	log.Info().Msg("connection manager started")
	
	for {
		select {
		case <-ctx.Done():
			log.Info().Msg("connection manager shutting down")
			return
		case message := <-cm.broadcastCh:
			cm.handleBroadcast(message)
		}
	}
}

// UpgradeConnection upgrades an HTTP connection to WebSocket
func (cm *ConnectionManager) UpgradeConnection(w http.ResponseWriter, r *http.Request, userID string, draftID uuid.UUID) error {
	conn, err := cm.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Error().Err(err).Msg("failed to upgrade WebSocket connection")
		return fmt.Errorf("failed to upgrade connection: %w", err)
	}

	// Create connection object
	connection := &Connection{
		ID:          uuid.New().String(),
		UserID:      userID,
		DraftID:     draftID,
		Conn:        conn,
		Send:        make(chan []byte, 256),
		Manager:     cm,
		ConnectedAt: time.Now(),
		LastPing:    time.Now(),
	}

	// Register the connection
	cm.registerConnection(connection)

	// Start connection handlers
	go connection.writePump()
	go connection.readPump()

	log.Info().
		Str("connection_id", connection.ID).
		Str("user_id", userID).
		Str("draft_id", draftID.String()).
		Msg("WebSocket connection established")

	return nil
}

// registerConnection adds a connection to the manager
func (cm *ConnectionManager) registerConnection(conn *Connection) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.draftConnections[conn.DraftID] == nil {
		cm.draftConnections[conn.DraftID] = make(map[*Connection]bool)
	}
	cm.draftConnections[conn.DraftID][conn] = true

	log.Debug().
		Str("connection_id", conn.ID).
		Str("draft_id", conn.DraftID.String()).
		Int("total_connections", len(cm.draftConnections[conn.DraftID])).
		Msg("connection registered")
}

// unregisterConnection removes a connection from the manager
func (cm *ConnectionManager) unregisterConnection(conn *Connection) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if connections, exists := cm.draftConnections[conn.DraftID]; exists {
		if _, exists := connections[conn]; exists {
			delete(connections, conn)
			close(conn.Send)

			// Clean up empty draft connection pools
			if len(connections) == 0 {
				delete(cm.draftConnections, conn.DraftID)
			}

			log.Info().
				Str("connection_id", conn.ID).
				Str("user_id", conn.UserID).
				Str("draft_id", conn.DraftID.String()).
				Msg("connection unregistered")
		}
	}
}

// BroadcastToDraft sends an event to all connections for a specific draft
func (cm *ConnectionManager) BroadcastToDraft(draftID uuid.UUID, event *DraftEvent) {
	select {
	case cm.broadcastCh <- BroadcastMessage{DraftID: draftID, Event: event}:
	default:
		log.Warn().Str("draft_id", draftID.String()).Msg("broadcast channel full, dropping message")
	}
}

// BroadcastToUser sends an event to a specific user in a draft
func (cm *ConnectionManager) BroadcastToUser(draftID uuid.UUID, userID string, event *DraftEvent) {
	select {
	case cm.broadcastCh <- BroadcastMessage{DraftID: draftID, Event: event, UserID: userID}:
	default:
		log.Warn().
			Str("draft_id", draftID.String()).
			Str("user_id", userID).
			Msg("broadcast channel full, dropping user message")
	}
}

// handleBroadcast processes a broadcast message
func (cm *ConnectionManager) handleBroadcast(message BroadcastMessage) {
	cm.mu.RLock()
	connections, exists := cm.draftConnections[message.DraftID]
	if !exists {
		cm.mu.RUnlock()
		return
	}

	// Create a snapshot of connections to avoid holding lock during broadcast
	var targetConnections []*Connection
	for conn := range connections {
		// Filter by user if specified
		if message.UserID != "" && conn.UserID != message.UserID {
			continue
		}
		targetConnections = append(targetConnections, conn)
	}
	cm.mu.RUnlock()

	// Marshal the event once
	eventData, err := json.Marshal(message.Event)
	if err != nil {
		log.Error().Err(err).Msg("failed to marshal event for broadcast")
		return
	}

	// Send to all target connections
	for _, conn := range targetConnections {
		select {
		case conn.Send <- eventData:
		default:
			// Connection is slow/dead, close it
			log.Warn().
				Str("connection_id", conn.ID).
				Str("user_id", conn.UserID).
				Msg("connection send buffer full, closing connection")
			cm.unregisterConnection(conn)
			conn.Conn.Close()
		}
	}

	log.Debug().
		Str("event_type", string(message.Event.Type)).
		Str("draft_id", message.DraftID.String()).
		Int("connections", len(targetConnections)).
		Msg("event broadcasted")
}

// GetConnectionStats returns statistics about active connections
func (cm *ConnectionManager) GetConnectionStats() map[string]interface{} {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	totalConnections := 0
	draftCounts := make(map[string]int)

	for draftID, connections := range cm.draftConnections {
		count := len(connections)
		totalConnections += count
		draftCounts[draftID.String()] = count
	}

	return map[string]interface{}{
		"total_connections": totalConnections,
		"active_drafts":     len(cm.draftConnections),
		"draft_connections": draftCounts,
	}
}

// writePump handles sending messages to the WebSocket connection
func (c *Connection) writePump() {
	ticker := time.NewTicker(c.Manager.config.PingInterval)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
		c.Manager.unregisterConnection(c)
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(c.Manager.config.WriteTimeout))
			if !ok {
				// Channel was closed
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Error().
					Err(err).
					Str("connection_id", c.ID).
					Msg("failed to write message to WebSocket")
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(c.Manager.config.WriteTimeout))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Error().
					Err(err).
					Str("connection_id", c.ID).
					Msg("failed to send ping")
				return
			}
			c.LastPing = time.Now()
		}
	}
}

// readPump handles reading messages from the WebSocket connection
func (c *Connection) readPump() {
	defer func() {
		c.Manager.unregisterConnection(c)
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(c.Manager.config.MaxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(c.Manager.config.ReadTimeout))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(c.Manager.config.ReadTimeout))
		c.LastPing = time.Now()
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Error().
					Err(err).
					Str("connection_id", c.ID).
					Msg("unexpected WebSocket close error")
			}
			break
		}

		// Handle incoming messages (ping/pong, client commands, etc.)
		c.handleClientMessage(message)
		c.Conn.SetReadDeadline(time.Now().Add(c.Manager.config.ReadTimeout))
	}
}

// handleClientMessage processes messages received from the client
func (c *Connection) handleClientMessage(message []byte) {
	// For now, just log client messages
	// In the future, this could handle client commands like "subscribe to draft", "unsubscribe", etc.
	log.Debug().
		Str("connection_id", c.ID).
		Str("user_id", c.UserID).
		RawJSON("message", message).
		Msg("received client message")
}