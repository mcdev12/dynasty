package gateway

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/mcdev12/dynasty/go/internal/draft/events"
)

// StateSyncExample demonstrates how a client would use both state API and WebSocket
type StateSyncExample struct {
	baseURL     string
	wsURL       string
	draftID     string
	userID      string
	conn        *websocket.Conn
	state       *DraftStateResponse
}

// NewStateSyncExample creates a new example client
func NewStateSyncExample(baseURL, draftID, userID string) *StateSyncExample {
	return &StateSyncExample{
		baseURL: baseURL,
		wsURL:   fmt.Sprintf("ws://%s/ws/draft?draft_id=%s&user_id=%s", baseURL, draftID, userID),
		draftID: draftID,
		userID:  userID,
	}
}

// Connect demonstrates the connection flow
func (e *StateSyncExample) Connect() error {
	// 1. First fetch current state via REST API
	if err := e.fetchInitialState(); err != nil {
		return fmt.Errorf("failed to fetch initial state: %w", err)
	}

	// 2. Then connect to WebSocket for real-time updates
	if err := e.connectWebSocket(); err != nil {
		return fmt.Errorf("failed to connect WebSocket: %w", err)
	}

	// 3. Start listening for events
	go e.listenForEvents()

	return nil
}

// fetchInitialState fetches the current draft state
func (e *StateSyncExample) fetchInitialState() error {
	url := fmt.Sprintf("http://%s/api/drafts/%s/state", e.baseURL, e.draftID)
	
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(&e.state); err != nil {
		return err
	}

	log.Printf("Fetched initial state: Draft %s, Status: %s", e.state.DraftID, e.state.Status)
	if e.state.CurrentPick != nil {
		log.Printf("Current pick: Team %s, Round %d Pick %d, %d seconds remaining",
			e.state.CurrentPick.TeamName,
			e.state.CurrentPick.Round,
			e.state.CurrentPick.Pick,
			*e.state.TimeRemaining)
	}

	return nil
}

// connectWebSocket establishes WebSocket connection
func (e *StateSyncExample) connectWebSocket() error {
	dialer := websocket.DefaultDialer
	conn, _, err := dialer.Dial(e.wsURL, nil)
	if err != nil {
		return err
	}

	e.conn = conn
	log.Printf("Connected to WebSocket for draft %s", e.draftID)
	return nil
}

// listenForEvents listens for real-time draft events
func (e *StateSyncExample) listenForEvents() {
	defer e.conn.Close()

	for {
		_, message, err := e.conn.ReadMessage()
		if err != nil {
			log.Printf("WebSocket read error: %v", err)
			break
		}

		var event DraftEvent
		if err := json.Unmarshal(message, &event); err != nil {
			log.Printf("Failed to parse event: %v", err)
			continue
		}

		e.handleEvent(&event)
	}
}

// handleEvent processes incoming events and updates local state
func (e *StateSyncExample) handleEvent(event *DraftEvent) {
	log.Printf("Received event: %s", event.Type)

	switch event.Type {
	case EventTypePickStarted:
		// Update current pick info
		var payload events.PickStartedPayload
		if err := json.Unmarshal(event.Data, &payload); err == nil {
			log.Printf("Pick started: Team %s, Round %d Pick %d",
				payload.TeamID, payload.Round, payload.Pick)
			
			// Update local state
			if e.state.CurrentPick == nil {
				e.state.CurrentPick = &CurrentPickInfo{}
			}
			e.state.CurrentPick.PickID = payload.PickID
			e.state.CurrentPick.TeamID = payload.TeamID
			e.state.CurrentPick.Round = payload.Round
			e.state.CurrentPick.Pick = payload.Pick
			e.state.CurrentPick.OverallPick = payload.OverallPick
			e.state.CurrentPick.StartedAt = payload.StartedAt
			e.state.CurrentPick.TimeoutAt = payload.TimeoutAt
			e.state.CurrentPick.TimePerPick = payload.TimePerPickSec
		}

	case EventTypePickMade:
		// Move current pick to recent picks
		var payload events.PickMadePayload
		if err := json.Unmarshal(event.Data, &payload); err == nil {
			log.Printf("Pick made: %s selected by %s", payload.PlayerName, payload.TeamName)
			
			// Add to recent picks
			recentPick := RecentPickInfo{
				PickID:      payload.PickID,
				TeamID:      payload.TeamID,
				TeamName:    payload.TeamName,
				PlayerID:    payload.PlayerID,
				PlayerName:  payload.PlayerName,
				Round:       payload.Round,
				Pick:        payload.Pick,
				OverallPick: payload.OverallPick,
				MadeAt:      time.Now(),
			}
			
			// Prepend to recent picks (keep last 10)
			e.state.RecentPicks = append([]RecentPickInfo{recentPick}, e.state.RecentPicks...)
			if len(e.state.RecentPicks) > 10 {
				e.state.RecentPicks = e.state.RecentPicks[:10]
			}
			
			e.state.CompletedPicks++
		}

	case EventTypeDraftCompleted:
		log.Printf("Draft completed!")
		e.state.Status = "completed"
		e.state.CurrentPick = nil
		e.state.TimeRemaining = nil

	case EventTypeDraftPaused:
		log.Printf("Draft paused")
		e.state.Status = "paused"
		e.state.TimeRemaining = nil

	case EventTypeDraftResumed:
		log.Printf("Draft resumed")
		e.state.Status = "in_progress"
	}
}

// Reconnect demonstrates reconnection flow
func (e *StateSyncExample) Reconnect() error {
	log.Printf("Reconnecting...")
	
	// Close existing connection if any
	if e.conn != nil {
		e.conn.Close()
	}

	// Fetch fresh state and reconnect
	return e.Connect()
}

// ClientUsageExample shows how a web client would use the gateway
func ClientUsageExample() {
	/*
	// JavaScript/TypeScript client example:
	
	class DraftClient {
		constructor(baseURL, draftID, userID) {
			this.baseURL = baseURL;
			this.draftID = draftID;
			this.userID = userID;
			this.ws = null;
			this.state = null;
		}

		async connect() {
			// 1. Fetch initial state
			const stateResp = await fetch(`${this.baseURL}/api/drafts/${this.draftID}/state`);
			this.state = await stateResp.json();
			
			// 2. Connect WebSocket
			const wsURL = `ws://${this.baseURL}/ws/draft?draft_id=${this.draftID}&user_id=${this.userID}`;
			this.ws = new WebSocket(wsURL);
			
			this.ws.onmessage = (event) => {
				const draftEvent = JSON.parse(event.data);
				this.handleEvent(draftEvent);
			};
			
			this.ws.onerror = (error) => {
				console.error('WebSocket error:', error);
				this.scheduleReconnect();
			};
			
			this.ws.onclose = () => {
				console.log('WebSocket closed');
				this.scheduleReconnect();
			};
		}

		handleEvent(event) {
			console.log('Received event:', event.type);
			
			// Update local state based on event
			switch(event.type) {
				case 'pick_started':
					this.updateCurrentPick(event.data);
					this.startTimer();
					break;
				case 'pick_made':
					this.addRecentPick(event.data);
					this.stopTimer();
					break;
				// ... handle other events
			}
			
			// Notify UI to re-render
			this.notifyStateChange();
		}

		scheduleReconnect() {
			setTimeout(() => this.connect(), 5000);
		}
	}
	*/
}