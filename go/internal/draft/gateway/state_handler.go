package gateway

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// StateProvider interface defines methods for retrieving draft state
type StateProvider interface {
	GetDraftState(ctx context.Context, draftID uuid.UUID) (*DraftStateResponse, error)
	GetActiveDrafts(ctx context.Context) ([]DraftSummary, error)
}

// DraftStateResponse represents the complete state of a draft
type DraftStateResponse struct {
	DraftID        string                 `json:"draft_id"`
	Status         string                 `json:"status"`
	CurrentPick    *CurrentPickInfo       `json:"current_pick,omitempty"`
	RecentPicks    []RecentPickInfo       `json:"recent_picks"`
	TimeRemaining  *int                   `json:"time_remaining_sec,omitempty"`
	TotalPicks     int                    `json:"total_picks"`
	CompletedPicks int                    `json:"completed_picks"`
	Metadata       map[string]interface{} `json:"metadata"`
}

// CurrentPickInfo represents the current pick on the clock
type CurrentPickInfo struct {
	PickID      string    `json:"pick_id"`
	TeamID      string    `json:"team_id"`
	TeamName    string    `json:"team_name"`
	Round       int       `json:"round"`
	Pick        int       `json:"pick"`
	OverallPick int       `json:"overall_pick"`
	StartedAt   time.Time `json:"started_at"`
	TimeoutAt   time.Time `json:"timeout_at"`
	TimePerPick int       `json:"time_per_pick_sec"`
}

// RecentPickInfo represents a recently made pick
type RecentPickInfo struct {
	PickID      string    `json:"pick_id"`
	TeamID      string    `json:"team_id"`
	TeamName    string    `json:"team_name"`
	PlayerID    string    `json:"player_id"`
	PlayerName  string    `json:"player_name"`
	Round       int       `json:"round"`
	Pick        int       `json:"pick"`
	OverallPick int       `json:"overall_pick"`
	MadeAt      time.Time `json:"made_at"`
}

// DraftSummary represents a summary of an active draft
type DraftSummary struct {
	DraftID      string     `json:"draft_id"`
	LeagueID     string     `json:"league_id"`
	Status       string     `json:"status"`
	StartedAt    *time.Time `json:"started_at,omitempty"`
	CurrentRound int        `json:"current_round"`
	CurrentPick  int        `json:"current_pick"`
	TotalTeams   int        `json:"total_teams"`
	TotalRounds  int        `json:"total_rounds"`
}

// StateHandler handles HTTP requests for draft state
type StateHandler struct {
	stateProvider StateProvider
}

// NewStateHandler creates a new state handler
func NewStateHandler(provider StateProvider) *StateHandler {
	return &StateHandler{
		stateProvider: provider,
	}
}

// HandleGetDraftState handles GET /api/drafts/{id}/state
func (h *StateHandler) HandleGetDraftState(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract draft ID from path
	// Expecting path like /api/drafts/{id}/state
	path := r.URL.Path
	draftIDStr := extractDraftIDFromPath(path)
	if draftIDStr == "" {
		http.Error(w, "Draft ID is required", http.StatusBadRequest)
		return
	}

	draftID, err := uuid.Parse(draftIDStr)
	if err != nil {
		http.Error(w, "Invalid draft ID format", http.StatusBadRequest)
		return
	}

	// Get draft state
	state, err := h.stateProvider.GetDraftState(r.Context(), draftID)
	if err != nil {
		log.Error().Err(err).Str("draft_id", draftID.String()).Msg("failed to get draft state")
		http.Error(w, "Failed to get draft state", http.StatusInternalServerError)
		return
	}

	// Calculate time remaining if draft is in progress
	if state.CurrentPick != nil {
		remaining := int(time.Until(state.CurrentPick.TimeoutAt).Seconds())
		if remaining > 0 {
			state.TimeRemaining = &remaining
		}
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(state); err != nil {
		log.Error().Err(err).Msg("failed to encode draft state response")
	}
}

// HandleGetActiveDrafts handles GET /api/drafts/active
func (h *StateHandler) HandleGetActiveDrafts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	drafts, err := h.stateProvider.GetActiveDrafts(r.Context())
	if err != nil {
		log.Error().Err(err).Msg("failed to get active drafts")
		http.Error(w, "Failed to get active drafts", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(drafts); err != nil {
		log.Error().Err(err).Msg("failed to encode active drafts response")
	}
}

// RegisterStateRoutes registers state-related HTTP routes
func (h *StateHandler) RegisterStateRoutes(mux *http.ServeMux) {
	// Register specific routes
	mux.HandleFunc("/api/drafts/active", h.HandleGetActiveDrafts)

	// Register pattern for draft state - note the trailing slash
	mux.HandleFunc("/api/drafts/", func(w http.ResponseWriter, r *http.Request) {
		log.Debug().Str("path", r.URL.Path).Msg("state handler received request")

		// Check if path ends with /state
		if len(r.URL.Path) > len("/api/drafts/") && r.URL.Path[len(r.URL.Path)-6:] == "/state" {
			h.HandleGetDraftState(w, r)
		} else {
			http.NotFound(w, r)
		}
	})
}

// extractDraftIDFromPath extracts draft ID from path like /api/drafts/{id}/state
func extractDraftIDFromPath(path string) string {
	// Remove prefix and suffix
	const prefix = "/api/drafts/"
	const suffix = "/state"

	if len(path) <= len(prefix)+len(suffix) {
		return ""
	}

	if path[:len(prefix)] != prefix || path[len(path)-len(suffix):] != suffix {
		return ""
	}

	return path[len(prefix) : len(path)-len(suffix)]
}
