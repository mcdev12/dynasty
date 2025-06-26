# Player Repository Architecture

This package implements an extensible player repository system that supports sport-specific profiles while maintaining a clean, testable architecture.

## Overview

The architecture separates generic player data (name, team, etc.) from sport-specific attributes (NFL position, NBA stats, etc.) using a plugin-style pattern. This allows adding new sports without modifying existing code.

## Key Components

### 1. Core Player Model (`/internal/models/player.go`)
```go
type Player struct {
    ID         uuid.UUID
    SportID    string
    ExternalID string
    FullName   string
    TeamID     *uuid.UUID
    CreatedAt  time.Time
    
    // Sport-specific profiles (only one will be populated)
    NFLPlayerProfile *NFLPlayerProfile
    // Future: NBAPlayerProfile *NBAPlayerProfile
}
```

### 2. Profile Repository Interface (`profile_repo.go`)
```go
type ProfileRepository interface {
    CreateProfile(ctx context.Context, qtx db.Querier, playerID uuid.UUID, profile models.Profile) error
    LoadProfile(ctx context.Context, q db.Querier, playerID uuid.UUID) (models.Profile, error)
    DeleteProfile(ctx context.Context, qtx db.Querier, playerID uuid.UUID) error
}
```

Key design decisions:
- **Explicit Querier passing**: Instead of context injection, we pass `db.Querier` explicitly. This makes transaction boundaries clear and enables easy mocking.
- **Strong typing with Profile interface**: Uses `models.Profile` interface for type safety while maintaining registry flexibility.
- **Sentinel error handling**: Uses `ErrNoProfile` for consistent "not found" error handling.

### 3. Sport-Specific Implementations (`nfl_profile_repo.go`)
Each sport implements the `ProfileRepository` interface:
```go
type NFLProfileRepository struct{}

func (r *NFLProfileRepository) CreateProfile(ctx context.Context, qtx db.Querier, playerID uuid.UUID, profile models.Profile) error {
    nflProfile, ok := profile.(*models.NFLPlayerProfile)
    // ... convert and save to nfl_player_profiles table using sqlutil helpers
}

// Self-registration on package initialization
func init() {
    if err := RegisterProfileRepo("nfl", NewNFLProfileRepository()); err != nil {
        panic(fmt.Sprintf("Failed to register NFL profile repository: %v", err))
    }
}
```

### 4. Main Repository (`repository.go`)
Orchestrates player + profile operations:
```go
func (r *Repository) CreatePlayer(ctx context.Context, req CreatePlayerRequest) (*models.Player, error) {
    tx, err := r.pool.Begin(ctx)
    defer tx.Rollback(ctx)
    
    qtx := r.queries.(*db.Queries).WithTx(tx)
    
    // 1. Create base player
    dbPlayer, err := qtx.CreatePlayer(ctx, params)
    
    // 2. Create sport-specific profile
    if req.Profile != nil {
        profileRepo, err := GetProfileRepo(req.SportID)
        err = profileRepo.CreateProfile(ctx, qtx, player.ID, req.Profile)
    }
    
    tx.Commit(ctx)
}
```

## How It Works

### Registration Flow
1. **Self-registration**: Each sport package registers itself automatically via `init()` functions
2. No manual initialization required - simply import the sport package and it registers itself
3. The registry maintains a map: `sportID -> ProfileRepository`

### Create Player Flow
1. Client creates a `CreatePlayerRequest` with base data + optional profile
2. Repository begins a transaction
3. Creates base player record in `players` table
4. Looks up the sport's profile repository from registry
5. Delegates profile creation to sport-specific implementation
6. Commits transaction

### Load Player Flow
1. Load base player from `players` table
2. Look up sport's profile repository
3. Load profile from sport-specific table (e.g., `nfl_player_profiles`)
4. Attach profile to player model

## Adding a New Sport

1. **Create profile model** in `/internal/models/player.go`:
```go
type NBAPlayerProfile struct {
    PlayerID     uuid.UUID
    Position     string
    Height       string
    // ... NBA-specific fields
}

// Implement Profile interface
func (p *NBAPlayerProfile) SportID() string {
    return "nba"
}
```

2. **Add to Player struct**:
```go
type Player struct {
    // ... existing fields
    NBAPlayerProfile *NBAPlayerProfile `json:"nba_player_profile,omitempty"`
}
```

3. **Create profile repository with self-registration**:
```go
type NBAProfileRepository struct{}

func (r *NBAProfileRepository) CreateProfile(ctx context.Context, qtx db.Querier, playerID uuid.UUID, profile models.Profile) error {
    nbaProfile, ok := profile.(*models.NBAPlayerProfile)
    // Implementation using sqlutil helpers
}

// Self-register on package initialization
func init() {
    if err := RegisterProfileRepo("nba", NewNBAProfileRepository()); err != nil {
        panic(fmt.Sprintf("Failed to register NBA profile repository: %v", err))
    }
}
```

4. **Update LoadProfileIntoPlayer** in `profile_repo.go`:
```go
switch p := profile.(type) {
case *models.NFLPlayerProfile:
    player.NFLPlayerProfile = p
case *models.NBAPlayerProfile:
    player.NBAPlayerProfile = p
}
```

## Testing & Mocking

The architecture supports easy testing through the `db.Querier` interface:

```go
// Mock implementation
type mockQuerier struct {
    db.Querier
    createPlayerFunc func(ctx context.Context, arg db.CreatePlayerParams) (db.Player, error)
}

func (m *mockQuerier) CreatePlayer(ctx context.Context, arg db.CreatePlayerParams) (db.Player, error) {
    return m.createPlayerFunc(ctx, arg)
}

// Test
func TestCreatePlayer(t *testing.T) {
    mockQ := &mockQuerier{
        createPlayerFunc: func(ctx context.Context, arg db.CreatePlayerParams) (db.Player, error) {
            // Return test data
        },
    }
    
    repo := NewRepository(mockQ, nil) // database not needed for this test
    // ... test logic
}
```

### Type System and SQL Utilities

The latest version uses standard Go types with sql.Null* wrappers instead of pgtype:
- `uuid.UUID` instead of `pgtype.UUID`
- `sql.NullString` instead of `pgtype.Text`
- `sql.NullInt32` instead of `pgtype.Int4`
- `uuid.NullUUID` instead of `pgtype.UUID` for optional UUIDs
- `time.Time` instead of `pgtype.Timestamptz`

Helper functions in `/internal/sqlutil/converters.go` provide conversion utilities:
- `ToSqlInt32()`, `FromSqlInt32()` for int pointer conversions
- `ToSqlString()`, `FromSqlString()` for string conversions
- `ToNullUUID()`, `FromNullUUID()` for UUID pointer conversions
- All profile repositories use these helpers for consistent type conversion

## Best Practices

1. **Transaction Scope**: Always use transactions for operations that touch multiple tables
2. **Error Handling**: Profile operations return `ErrNoProfile` for "not found" scenarios for consistent handling
3. **Idempotent Deletes**: Delete operations succeed even if the record doesn't exist
4. **Type Safety**: Use concrete types in your application code, leverage the `models.Profile` interface for type-safe registrations
5. **Querier Pattern**: Always accept `db.Querier` to support both direct queries and transactions

## Migration Guide

When migrating from the old context-based transaction approach:

1. **Interface Updates**: Update profile methods to use `models.Profile` instead of `interface{}`
2. **SQL Utilities**: Replace inline sql.Null* conversions with `sqlutil` helper functions
3. **Self-Registration**: Remove manual initialization calls, add `init()` functions to profile repositories
4. **Error Handling**: Use `ErrNoProfile` sentinel instead of checking `sql.ErrNoRows` directly
5. **Import sqlutil**: Import the new `/internal/sqlutil` package for type conversion helpers