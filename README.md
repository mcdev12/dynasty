# Dynasty Fantasy Sports Platform

A comprehensive fantasy sports backend built with Go, implementing clean architecture patterns with gRPC/Connect APIs, SQLC for type-safe database operations, and Protocol Buffers for service definitions.

## 🏗️ Architecture Overview

The platform follows a clean architecture pattern with clear separation of concerns:

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   gRPC Service  │────│   App Layer     │────│   Repository    │
│   (Transport)   │    │  (Business)     │    │   (Database)    │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         │                       │                       │
         │                       │                       │
    Protobuf APIs          Domain Models            SQLC Queries
```

### Technology Stack

- **Language**: Go 1.21+
- **API**: gRPC with Connect-Go
- **Database**: PostgreSQL with SQLC
- **Schema**: Protocol Buffers
- **Architecture**: Clean Architecture / Hexagonal Architecture

## 🎯 Core Modules

### 1. **User Management** (`/go/internal/users/`)
- User registration and authentication
- Profile management
- CRUD operations for user entities

### 2. **League Management** (`/go/internal/leagues/`)
- Fantasy league creation and configuration
- League settings and rules management
- Commissioner controls
- Support for different league types (Redraft, Keeper, Dynasty)

### 3. **Fantasy Team Management** (`/go/internal/fantasyteams/`)
- Team creation within leagues
- Team ownership and roster management
- Team metadata (names, logos, etc.)

### 4. **Roster Management** (`/go/internal/roster/`)
- Player roster assignments
- Position management (Starting, Bench, IR, Taxi Squad)
- Acquisition tracking (Draft, Waiver, Free Agent, Trade, Keeper)
- Keeper data handling for dynasty leagues

### 5. **Draft System** (`/go/internal/draft/`)
- **Multiple Draft Types**:
  - Snake Draft (with reversal logic)
  - Auction Draft (linear order)
  - Rookie Draft (dynasty leagues)
- **Draft Management**:
  - Draft creation and configuration
  - Status transitions (Not Started → In Progress → Completed)
  - Settings validation per draft type
- **Pick Management**:
  - Automated pick slot generation (prepopulation)
  - Pick tracking and assignment
  - Round and overall pick calculations

### 6. **Player Database** (`/go/internal/models/`)
- Player profiles and statistics
- Team affiliations
- Sport-specific data (NFL profiles)

## 🔧 Component Structure

Each module follows a consistent 4-layer architecture:

### **Service Layer** (`service.go`)
- gRPC endpoint implementations
- Protocol Buffer message conversion
- HTTP/gRPC error handling
- Request validation

### **App Layer** (`app.go`)
- Business logic and validation
- Cross-entity relationship management
- Domain rule enforcement
- Transaction coordination

### **Repository Layer** (`repository.go`)
- Database abstraction
- SQLC query integration
- Data model conversion
- CRUD operations

### **Database Layer** (`/db/`)
- **Queries** (`/queries/*.sql`): SQLC query definitions
- **Models** (`models.go`): Generated database types
- **Migrations**: Database schema definitions

## 📋 Domain Models

### Core Entities

```go
// User entity
type User struct {
    ID        uuid.UUID `json:"id"`
    Username  string    `json:"username"`
    Email     string    `json:"email"`
    CreatedAt time.Time `json:"created_at"`
}

// League entity with comprehensive settings
type League struct {
    ID             uuid.UUID       `json:"id"`
    Name           string          `json:"name"`
    SportID        string          `json:"sport_id"`
    LeagueType     LeagueType      `json:"league_type"`    // REDRAFT, KEEPER, DYNASTY
    CommissionerID uuid.UUID       `json:"commissioner_id"`
    LeagueSettings interface{}     `json:"league_settings"` // JSON configuration
    Status         LeagueStatus    `json:"status"`          // PENDING, ACTIVE, COMPLETED
    Season         string          `json:"season"`
    CreatedAt      time.Time       `json:"created_at"`
    UpdatedAt      time.Time       `json:"updated_at"`
}

// Draft entity with type-specific settings
type Draft struct {
    ID          uuid.UUID     `json:"id"`
    LeagueID    uuid.UUID     `json:"league_id"`
    DraftType   DraftType     `json:"draft_type"`    // SNAKE, AUCTION, ROOKIE
    Status      DraftStatus   `json:"status"`        // NOT_STARTED, IN_PROGRESS, COMPLETED
    Settings    DraftSettings `json:"settings"`      // Type-specific configuration
    ScheduledAt *time.Time    `json:"scheduled_at,omitempty"`
    StartedAt   *time.Time    `json:"started_at,omitempty"`
    CompletedAt *time.Time    `json:"completed_at,omitempty"`
    CreatedAt   time.Time     `json:"created_at"`
    UpdatedAt   time.Time     `json:"updated_at"`
}

// Draft pick tracking
type DraftPick struct {
    ID            uuid.UUID  `json:"id"`
    DraftID       uuid.UUID  `json:"draft_id"`
    Round         int        `json:"round"`
    Pick          int        `json:"pick"`          // Pick within round
    OverallPick   int        `json:"overall_pick"`  // Overall pick number
    TeamID        uuid.UUID  `json:"team_id"`
    PlayerID      *uuid.UUID `json:"player_id,omitempty"`     // Set when picked
    PickedAt      *time.Time `json:"picked_at,omitempty"`     // Timestamp of pick
    AuctionAmount *float64   `json:"auction_amount,omitempty"` // Auction drafts
    KeeperPick    bool       `json:"keeper_pick"`             // Keeper designation
}
```

## 🚀 Key Features

### **Draft System Highlights**

#### **Smart Pick Generation**
- **Automated prepopulation** of all pick slots based on:
  - Number of rounds
  - Number of teams (from draft order)
  - Draft type-specific logic

#### **Draft Type Support**
1. **Snake Draft**:
   - Alternating round direction (1→12, 12→1, 1→12...)
   - Optional third-round reversal
   - Proper overall pick calculation

2. **Auction Draft**:
   - Linear team order (no snake reversal)
   - Budget and bid increment validation
   - Nomination time tracking

3. **Rookie Draft**:
   - Similar to snake draft
   - Typically shorter (≤5 rounds)
   - Dynasty league specific

#### **Status Management**
- **State machine validation** for draft progression
- **Allowed transitions**:
  - `NOT_STARTED` → `IN_PROGRESS`, `CANCELLED`
  - `IN_PROGRESS` → `PAUSED`, `COMPLETED`, `CANCELLED`
  - `PAUSED` → `IN_PROGRESS`, `CANCELLED`
  - `COMPLETED` / `CANCELLED` → No transitions

### **Roster Management**
- **Position tracking**: Starting, Bench, IR, Taxi Squad
- **Acquisition history**: Draft, Waiver, Free Agent, Trade, Keeper
- **Keeper data**: JSON storage for dynasty league rules
- **Cross-validation**: Prevents duplicate assignments

### **Database Features**
- **Type-safe queries** with SQLC generation
- **Efficient batch operations** for bulk inserts
- **UUID primary keys** throughout
- **Comprehensive indexing** for performance
- **PostgreSQL array support** for batch operations

## 🔌 API Endpoints

### Draft Service (`/draft/v1/`)
```protobuf
service DraftService {
  rpc CreateDraft(CreateDraftRequest) returns (CreateDraftResponse);
  rpc GetDraft(GetDraftRequest) returns (GetDraftResponse);
  rpc UpdateDraft(UpdateDraftRequest) returns (UpdateDraftResponse);
  rpc DeleteDraft(DeleteDraftRequest) returns (DeleteDraftResponse);
  rpc ListDraftsForLeague(ListDraftsForLeagueRequest) returns (ListDraftsForLeagueResponse);
}
```

### Roster Service (`/roster/v1/`)
```protobuf
service RosterService {
  rpc CreateRosterPlayer(CreateRosterPlayerRequest) returns (CreateRosterPlayerResponse);
  rpc GetRosterPlayersByFantasyTeam(GetRosterPlayersByFantasyTeamRequest) returns (GetRosterPlayersByFantasyTeamResponse);
  rpc UpdateRosterPlayerPosition(UpdateRosterPlayerPositionRequest) returns (UpdateRosterPlayerPositionResponse);
  // ... additional 13 operations
}
```

## 🗃️ Database Schema

### Core Tables
- `users` - User accounts
- `leagues` - Fantasy league definitions
- `fantasy_teams` - Team instances within leagues
- `players` - Player database
- `roster_players` - Player-team assignments
- `draft` - Draft configurations
- `draft_picks` - Individual pick tracking

### Key Relationships
```sql
leagues (1) ←→ (N) fantasy_teams
leagues (1) ←→ (N) draft
fantasy_teams (1) ←→ (N) roster_players
draft (1) ←→ (N) draft_picks
players (1) ←→ (N) roster_players
```

## 🚦 Getting Started

### Prerequisites
- Go 1.21+
- PostgreSQL 14+
- Protocol Buffers compiler
- SQLC CLI tool

### Development Setup
```bash
# Install dependencies
go mod download

# Generate SQLC code
sqlc generate

# Generate Protocol Buffers
buf generate

# Run tests
go test ./...
```

### Environment Variables
```env
DATABASE_URL=postgres://user:pass@localhost/dynasty
GRPC_PORT=8080
LOG_LEVEL=info
```

## 🧪 Testing Strategy

- **Unit tests** for business logic
- **Integration tests** for database operations
- **Contract tests** for gRPC services
- **End-to-end tests** for complete workflows

## 📈 Performance Considerations

- **Batch operations** for bulk data insertion
- **Database indexing** on frequently queried fields
- **Connection pooling** for database efficiency
- **Structured logging** for observability

## 🔮 Future Roadmap

- [ ] Real-time draft updates with WebSockets
- [ ] Advanced analytics and statistics
- [ ] Mobile API optimization
- [ ] Multi-sport support expansion
- [ ] Advanced keeper/dynasty rules engine

## 🤝 Contributing

1. Follow clean architecture principles
2. Maintain consistent error handling
3. Add comprehensive tests
4. Update Protocol Buffer definitions as needed
5. Document API changes

## 📝 License

[License information to be added]

---

**Dynasty** - Building the future of fantasy sports management with modern Go architecture and type-safe development practices.