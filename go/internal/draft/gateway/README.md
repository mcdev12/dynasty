# Draft Gateway Architecture & Event Schemas

## Overview

The Draft Gateway is a real-time event distribution service that bridges the draft orchestrator and client applications. It consumes events from the outbox pattern and broadcasts them to connected clients via WebSockets.

## Architecture Components

### 1. Event Schema Design

#### Core Event Structure
```go
type DraftEvent struct {
    ID        string          `json:"id"`         // Event UUID
    DraftID   string          `json:"draft_id"`   // Draft UUID
    Type      string          `json:"type"`       // Event type
    Timestamp time.Time       `json:"timestamp"`  // Event creation time
    Data      json.RawMessage `json:"data"`       // Event-specific payload
}
```

#### Event Types

1. **PickMade** - When a pick is completed
2. **PickStarted** - When a pick timer begins
3. **DraftStarted** - When draft begins
4. **DraftPaused** - When draft is paused
5. **DraftResumed** - When draft is resumed
6. **DraftCompleted** - When draft ends
7. **TimerTick** - Periodic timer updates (optional)

### 2. Event Payloads

#### PickMade Event
```go
type PickMadePayload struct {
    PickID      string `json:"pick_id"`
    PlayerID    string `json:"player_id"`
    TeamID      string `json:"team_id"`
    Round       int    `json:"round"`
    Pick        int    `json:"pick"`
    OverallPick int    `json:"overall_pick"`
    PlayerName  string `json:"player_name"`
    TeamName    string `json:"team_name"`
    PickedAt    time.Time `json:"picked_at"`
}
```

#### PickStarted Event
```go
type PickStartedPayload struct {
    PickID         string    `json:"pick_id"`
    TeamID         string    `json:"team_id"`
    Round          int       `json:"round"`
    Pick           int       `json:"pick"`
    OverallPick    int       `json:"overall_pick"`
    StartedAt      time.Time `json:"started_at"`
    TimeoutAt      time.Time `json:"timeout_at"`
    TimePerPickSec int       `json:"time_per_pick_sec"`
}
```

#### DraftStarted Event
```go
type DraftStartedPayload struct {
    DraftID     string    `json:"draft_id"`
    DraftType   string    `json:"draft_type"`
    StartedAt   time.Time `json:"started_at"`
    TotalRounds int       `json:"total_rounds"`
    TotalPicks  int       `json:"total_picks"`
}
```

#### DraftCompleted Event
```go
type DraftCompletedPayload struct {
    DraftID     string    `json:"draft_id"`
    CompletedAt time.Time `json:"completed_at"`
    Duration    string    `json:"duration"`
    TotalPicks  int       `json:"total_picks"`
}
```

### 3. Service Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│ Draft           │    │ Draft Gateway    │    │ Client Apps     │
│ Orchestrator    │───▶│                  │───▶│ (WebSocket)     │
│                 │    │                  │    │                 │
│ • Outbox Events │    │ • Event Consumer │    │ • React App     │
│ • Timer Logic   │    │ • WebSocket Hub  │    │ • Mobile App    │
│ • Pick Logic    │    │ • State Sync API │    │ • Admin Panel   │
└─────────────────┘    └──────────────────┘    └─────────────────┘
```

### 4. Message Flow

1. **Event Creation**: Orchestrator writes events to `draft_outbox` table
2. **Event Consumption**: Gateway polls/subscribes to outbox events
3. **Event Broadcasting**: Gateway fans out events to connected clients
4. **State Sync**: Clients can query current state via REST API

### 5. WebSocket Protocol

#### Connection Endpoint
```
ws://localhost:8080/drafts/{draftId}/connect
```

#### Authentication
- JWT token via `Authorization` header or query param
- Validates user has access to the specific draft

#### Message Format
```json
{
  "id": "event-uuid",
  "draft_id": "draft-uuid",
  "type": "PickMade",
  "timestamp": "2023-12-01T10:00:00Z",
  "data": {
    "pick_id": "pick-uuid",
    "player_id": "player-uuid",
    "team_id": "team-uuid",
    "round": 1,
    "pick": 1,
    "overall_pick": 1,
    "player_name": "Patrick Mahomes",
    "team_name": "Team 1",
    "picked_at": "2023-12-01T10:00:00Z"
  }
}
```

### 6. State Synchronization API

#### Get Current Draft State
```
GET /api/drafts/{draftId}/state
```

Response:
```json
{
  "draft_id": "draft-uuid",
  "status": "IN_PROGRESS",
  "current_pick": {
    "pick_id": "pick-uuid",
    "team_id": "team-uuid", 
    "round": 2,
    "pick": 3,
    "overall_pick": 15,
    "started_at": "2023-12-01T10:05:00Z",
    "timeout_at": "2023-12-01T10:06:00Z",
    "time_remaining_sec": 45
  },
  "total_picks": 180,
  "completed_picks": 14
}
```

#### Get Pick History
```
GET /api/drafts/{draftId}/picks?limit=20&cursor=pick-uuid
```

### 7. Scaling Considerations

- **Horizontal Scaling**: Multiple gateway instances behind load balancer
- **Sticky Sessions**: Route clients to same instance for WebSocket persistence
- **Redis Pub/Sub**: Cross-instance event broadcasting for true stateless design
- **Connection Limits**: Rate limiting and maximum connections per instance

### 8. Error Handling

- **Connection Drops**: Automatic reconnection with state resync
- **Event Delivery Failures**: Dead letter queue for failed broadcasts
- **Authentication Failures**: Proper error codes and reconnection flows

### 9. Monitoring & Observability

- **Metrics**: Connection count, event throughput, error rates
- **Tracing**: End-to-end event flow from orchestrator to client
- **Logging**: Structured logs with correlation IDs

## Implementation Phases

1. **Phase 1**: Core event schemas and outbox consumer
2. **Phase 2**: WebSocket connection management
3. **Phase 3**: State synchronization API
4. **Phase 4**: Authentication and authorization
5. **Phase 5**: Scaling and monitoring