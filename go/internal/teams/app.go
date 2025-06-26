package teams

import (
	"context"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/mcdev12/dynasty/go/clients/sports_api_client"
	"github.com/mcdev12/dynasty/go/internal/models"
	"github.com/mcdev12/dynasty/go/internal/sports/base"
)

// TeamsRepository defines what the app layer needs from the repository
type TeamsRepository interface {
	CreateTeam(ctx context.Context, req CreateTeamRequest) (*models.Team, error)
	GetTeam(ctx context.Context, id uuid.UUID) (*models.Team, error)
	GetTeamByExternalID(ctx context.Context, sportID, externalID string) (*models.Team, error)
	GetTeamBySportIdAndCode(ctx context.Context, sportID, code string) (*models.Team, error)
	ListTeamsBySport(ctx context.Context, sportID string) ([]models.Team, error)
	ListAllTeams(ctx context.Context) ([]models.Team, error)
	UpdateTeam(ctx context.Context, id uuid.UUID, req UpdateTeamRequest) (*models.Team, error)
	DeleteTeam(ctx context.Context, id uuid.UUID) error
}

// SyncResult represents the result of syncing teams from external API
type SyncResult struct {
	TotalProcessed int     `json:"total_processed"`
	Created        int     `json:"created"`
	Updated        int     `json:"updated"`
	Errors         []error `json:"errors,omitempty"`
}

// App handles teams business logic
type App struct {
	repo    TeamsRepository
	plugins map[string]base.SportPlugin
}

// NewApp creates a new teams App
func NewApp(repo TeamsRepository, plugins map[string]base.SportPlugin) *App {
	return &App{
		repo:    repo,
		plugins: plugins,
	}
}

// CreateTeam creates a new team with validation
func (a *App) CreateTeam(ctx context.Context, req CreateTeamRequest) (*models.Team, error) {
	if err := a.validateCreateTeamRequest(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Check if team with same external ID already exists
	existingTeam, err := a.repo.GetTeamByExternalID(ctx, req.SportID, req.ExternalID)
	if err == nil && existingTeam != nil {
		return nil, fmt.Errorf("team with external ID %s already exists for sport %s", req.ExternalID, req.SportID)
	}

	team, err := a.repo.CreateTeam(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create team: %w", err)
	}

	log.Printf("Created team: %s (%s) in sport %s", team.Name, team.Code, team.SportID)
	return team, nil
}

// GetTeam retrieves a team by ID
func (a *App) GetTeam(ctx context.Context, id uuid.UUID) (*models.Team, error) {
	team, err := a.repo.GetTeam(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get team: %w", err)
	}
	return team, nil
}

// GetTeamByExternalID retrieves a team by sport ID and external ID
func (a *App) GetTeamByExternalID(ctx context.Context, sportID, externalID string) (*models.Team, error) {
	team, err := a.repo.GetTeamByExternalID(ctx, sportID, externalID)
	if err != nil {
		return nil, fmt.Errorf("failed to get team by external ID: %w", err)
	}
	return team, nil
}

func (a *App) GetTeamBySportIdAndCode(ctx context.Context, sportID, code string) (*models.Team, error) {
	team, err := a.repo.GetTeamBySportIdAndCode(ctx, sportID, code)
	if err != nil {
		return nil, fmt.Errorf("failed to get team by sport ID and code: %w", err)
	}
	return team, nil
}

// ListTeamsBySport retrieves all teams for a specific sport
func (a *App) ListTeamsBySport(ctx context.Context, sportID string) ([]models.Team, error) {
	teams, err := a.repo.ListTeamsBySport(ctx, sportID)
	if err != nil {
		return nil, fmt.Errorf("failed to list teams by sport: %w", err)
	}
	return teams, nil
}

// ListAllTeams retrieves all teams
func (a *App) ListAllTeams(ctx context.Context) ([]models.Team, error) {
	teams, err := a.repo.ListAllTeams(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list all teams: %w", err)
	}
	return teams, nil
}

// UpdateTeam updates an existing team with validation
func (a *App) UpdateTeam(ctx context.Context, id uuid.UUID, req UpdateTeamRequest) (*models.Team, error) {
	if err := a.validateUpdateTeamRequest(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Verify team exists
	_, err := a.repo.GetTeam(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("team not found: %w", err)
	}

	team, err := a.repo.UpdateTeam(ctx, id, req)
	if err != nil {
		return nil, fmt.Errorf("failed to update team: %w", err)
	}

	log.Printf("Updated team: %s (%s)", team.Name, team.Code)
	return team, nil
}

// DeleteTeam deletes a team by ID
func (a *App) DeleteTeam(ctx context.Context, id uuid.UUID) error {
	// Verify team exists
	team, err := a.repo.GetTeam(ctx, id)
	if err != nil {
		return fmt.Errorf("team not found: %w", err)
	}

	if err := a.repo.DeleteTeam(ctx, id); err != nil {
		return fmt.Errorf("failed to delete team: %w", err)
	}

	log.Printf("Deleted team: %s (%s)", team.Name, team.Code)
	return nil
}

// SyncTeamsFromAPI synchronizes teams from external sports API
func (a *App) SyncTeamsFromAPI(ctx context.Context, sportID string) (*SyncResult, error) {
	result := &SyncResult{}

	plugin, ok := a.plugins[sportID]
	if !ok {
		return nil, fmt.Errorf("no plugin registered for sport %q", sportID)
	}

	// Use injected plugin to fetch teams
	apiTeams, err := plugin.FetchTeams(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch teams from plugin: %w", err)
	}

	result.TotalProcessed = len(apiTeams)

	for _, apiTeam := range apiTeams {
		isNew, err := a.upsertTeam(ctx, sportID, apiTeam)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("failed to upsert team %s: %w", apiTeam.Name, err))
			continue
		}

		if isNew {
			result.Created++
		} else {
			result.Updated++
		}
	}

	log.Printf("Sync completed for %s: %d processed, %d created, %d updated, %d errors",
		sportID, result.TotalProcessed, result.Created, result.Updated, len(result.Errors))

	return result, nil
}

// GetTeamsWithFilter retrieves teams with filtering and pagination
func (a *App) GetTeamsWithFilter(ctx context.Context, filter TeamFilter, pagination PaginationParams) (*TeamListResponse, error) {
	// For now, implement basic filtering - extend with more sophisticated filtering later
	var teams []models.Team
	var err error

	if filter.SportID != nil {
		teams, err = a.repo.ListTeamsBySport(ctx, *filter.SportID)
	} else {
		teams, err = a.repo.ListAllTeams(ctx)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get teams: %w", err)
	}

	// Apply client-side filtering (should be moved to repository/database level for better performance)
	filteredTeams := a.applyFilters(teams, filter)

	// Apply pagination
	paginatedTeams := a.applyPagination(filteredTeams, pagination)

	return &TeamListResponse{
		Teams:   paginatedTeams,
		Total:   len(filteredTeams),
		Limit:   pagination.Limit,
		Offset:  pagination.Offset,
		HasMore: pagination.Offset+len(paginatedTeams) < len(filteredTeams),
	}, nil
}

// upsertTeam performs an upsert operation for a team (create if not exists, update if exists)
// Returns true if team was created (new), false if updated
func (a *App) upsertTeam(ctx context.Context, sportID string, apiTeam sports_api_client.Team) (bool, error) {
	// Use plugin to map external team to our domain model
	plugin, ok := a.plugins[sportID]
	if !ok {
		return false, fmt.Errorf("no plugin registered for sport %q", sportID)
	}
	mappedTeam, err := plugin.MapExternalTeam(apiTeam, sportID)
	if err != nil {
		return false, fmt.Errorf("failed to map external team: %w", err)
	}

	// Check if team already exists using the mapped team's external ID
	existingTeam, err := a.repo.GetTeamByExternalID(ctx, sportID, mappedTeam.ExternalID)
	if err != nil {
		// Team doesn't exist, create it
		createReq := a.teamToCreateRequest(mappedTeam)
		_, err := a.repo.CreateTeam(ctx, createReq)
		if err != nil {
			return false, fmt.Errorf("failed to create team: %w", err)
		}
		return true, nil // Created new team
	}

	// Team exists, update it
	updateReq := a.teamToUpdateRequest(mappedTeam)
	_, err = a.repo.UpdateTeam(ctx, existingTeam.ID, updateReq)
	if err != nil {
		return false, fmt.Errorf("failed to update team: %w", err)
	}
	return false, nil // Updated existing team
}

// teamToCreateRequest converts Team domain model to CreateTeamRequest
func (a *App) teamToCreateRequest(team *models.Team) CreateTeamRequest {
	return CreateTeamRequest{
		SportID:         team.SportID,
		ExternalID:      team.ExternalID,
		Name:            team.Name,
		Code:            team.Code,
		City:            team.City,
		Coach:           team.Coach,
		Owner:           team.Owner,
		Stadium:         team.Stadium,
		EstablishedYear: team.EstablishedYear,
	}
}

// teamToUpdateRequest converts Team domain model to UpdateTeamRequest
func (a *App) teamToUpdateRequest(team *models.Team) UpdateTeamRequest {
	return UpdateTeamRequest{
		Name:            &team.Name,
		Code:            &team.Code,
		City:            &team.City,
		Coach:           team.Coach,
		Owner:           team.Owner,
		Stadium:         team.Stadium,
		EstablishedYear: team.EstablishedYear,
	}
}

// applyFilters applies client-side filters to teams
func (a *App) applyFilters(teams []models.Team, filter TeamFilter) []models.Team {
	if filter.City == nil && filter.Code == nil {
		return teams
	}

	var filtered []models.Team
	for _, team := range teams {
		if filter.City != nil && team.City != *filter.City {
			continue
		}
		if filter.Code != nil && team.Code != *filter.Code {
			continue
		}
		filtered = append(filtered, team)
	}

	return filtered
}

// applyPagination applies pagination to teams slice
func (a *App) applyPagination(teams []models.Team, pagination PaginationParams) []models.Team {
	if pagination.Offset >= len(teams) {
		return []models.Team{}
	}

	end := pagination.Offset + pagination.Limit
	if end > len(teams) {
		end = len(teams)
	}

	return teams[pagination.Offset:end]
}

// validateCreateTeamRequest validates create team request
func (a *App) validateCreateTeamRequest(req CreateTeamRequest) error {
	if req.SportID == "" {
		return fmt.Errorf("sport_id is required")
	}
	if req.ExternalID == "" {
		return fmt.Errorf("external_id is required")
	}
	if req.Name == "" {
		return fmt.Errorf("name is required")
	}
	if req.Code == "" {
		return fmt.Errorf("code is required")
	}
	if req.City == "" {
		return fmt.Errorf("city is required")
	}
	return nil
}

// validateUpdateTeamRequest validates update team request
func (a *App) validateUpdateTeamRequest(req UpdateTeamRequest) error {
	if req.Name != nil && *req.Name == "" {
		return fmt.Errorf("name cannot be empty")
	}
	if req.Code != nil && *req.Code == "" {
		return fmt.Errorf("code cannot be empty")
	}
	if req.City != nil && *req.City == "" {
		return fmt.Errorf("city cannot be empty")
	}
	return nil
}
