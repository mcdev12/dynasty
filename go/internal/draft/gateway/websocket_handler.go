package gateway

import (
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// WebSocketHandler handles WebSocket upgrade requests for draft connections
type WebSocketHandler struct {
	connectionManager *ConnectionManager
}

// NewWebSocketHandler creates a new WebSocket handler
func NewWebSocketHandler(cm *ConnectionManager) *WebSocketHandler {
	return &WebSocketHandler{
		connectionManager: cm,
	}
}

// HandleDraftConnection handles WebSocket connections for a specific draft
func (h *WebSocketHandler) HandleDraftConnection(w http.ResponseWriter, r *http.Request) {
	// Extract draft ID from URL path or query parameter
	draftIDStr := r.URL.Query().Get("draft_id")
	if draftIDStr == "" {
		http.Error(w, "draft_id is required", http.StatusBadRequest)
		return
	}

	draftID, err := uuid.Parse(draftIDStr)
	if err != nil {
		http.Error(w, "invalid draft_id format", http.StatusBadRequest)
		return
	}

	// Extract user ID from query parameter or header
	// In production, this would come from JWT token or session
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		// For development, allow anonymous connections
		userID = "anonymous"
	}

	// Upgrade the connection
	if err := h.connectionManager.UpgradeConnection(w, r, userID, draftID); err != nil {
		log.Error().
			Err(err).
			Str("draft_id", draftID.String()).
			Str("user_id", userID).
			Msg("failed to upgrade WebSocket connection")
		http.Error(w, "failed to upgrade connection", http.StatusInternalServerError)
		return
	}

	// Connection is now handled by the connection manager
}

// HandleConnectionStats returns statistics about active connections
func (h *WebSocketHandler) HandleConnectionStats(w http.ResponseWriter, r *http.Request) {
	stats := h.connectionManager.GetConnectionStats()
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	
	// Simple JSON response
	w.Write([]byte("{"))
	w.Write([]byte("\"total_connections\":" + strconv.Itoa(stats["total_connections"].(int)) + ","))
	w.Write([]byte("\"active_drafts\":" + strconv.Itoa(stats["active_drafts"].(int))))
	w.Write([]byte("}"))
}

// RegisterRoutes registers WebSocket routes with an HTTP mux
func (h *WebSocketHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/ws/draft", h.HandleDraftConnection)
	mux.HandleFunc("/ws/stats", h.HandleConnectionStats)
}