package gateway

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/mcdev12/dynasty/go/internal/draft"
	"github.com/mcdev12/dynasty/go/internal/models"
)

// DraftStateProvider implements StateProvider using the draft app
type DraftStateProvider struct {
	draftApp draft.DraftApp
}

// NewDraftStateProvider creates a new draft state provider
func NewDraftStateProvider(app draft.DraftApp) *DraftStateProvider {
	return &DraftStateProvider{
		draftApp: app,
	}
}

// GetDraftState retrieves the complete state of a draft
func (p *DraftStateProvider) GetDraftState(ctx context.Context, draftID uuid.UUID) (*DraftStateResponse, error) {
	// Get draft details
	draftModel, err := p.draftApp.GetDraft(ctx, draftID)
	if err != nil {
		return nil, fmt.Errorf("failed to get draft: %w", err)
	}

	// Build response
	response := &DraftStateResponse{
		DraftID: draftID.String(),
		Status:  string(draftModel.Status),
		Metadata: map[string]interface{}{
			"league_id":    draftModel.LeagueID.String(),
			"draft_type":   string(draftModel.DraftType),
			"total_rounds": draftModel.Settings.Rounds,
			"total_teams":  len(draftModel.Settings.DraftOrder),
		},
	}

	// Calculate total picks
	response.TotalPicks = draftModel.Settings.Rounds * len(draftModel.Settings.DraftOrder)

	// Get pick information if draft is in progress
	if draftModel.Status == models.DraftStatusInProgress {
		// Get current pick on the clock
		currentPick, err := p.draftApp.GetNextPickForDraft(ctx, draftID)
		// TODO Better error handling here
		if err == nil && currentPick != nil && currentPick.PlayerID == nil {
			// This pick hasn't been made yet, so it's the current pick
			response.CurrentPick = &CurrentPickInfo{
				PickID:      currentPick.ID.String(),
				TeamID:      currentPick.TeamID.String(),
				TeamName:    fmt.Sprintf("Team %s", currentPick.TeamID.String()[:8]), // TODO: Get actual team name
				Round:       currentPick.Round,
				Pick:        currentPick.Pick,
				OverallPick: currentPick.OverallPick,
				TimePerPick: draftModel.Settings.TimePerPickSec,
			}

			// Set timer information if available
			if draftModel.NextDeadline != nil {
				response.CurrentPick.TimeoutAt = *draftModel.NextDeadline
				// StartedAt would be TimeoutAt minus TimePerPick
				response.CurrentPick.StartedAt = draftModel.NextDeadline.Add(-timeDurationFromSeconds(draftModel.Settings.TimePerPickSec))
			}
		}

		// TODO: Get recent picks (last 5-10 picks)
		// This would require a new method in DraftApp to fetch recent picks
		response.RecentPicks = []RecentPickInfo{}

		// TODO: Count completed picks
		// This would require querying picks where PlayerID is not nil
		response.CompletedPicks = 0
	}

	return response, nil
}

// GetActiveDrafts retrieves all active drafts
func (p *DraftStateProvider) GetActiveDrafts(ctx context.Context) ([]DraftSummary, error) {
	// TODO: This would require a new method in DraftApp to list drafts by status
	// For now, return empty list
	return []DraftSummary{}, nil
}

// timeDurationFromSeconds converts seconds to time.Duration
func timeDurationFromSeconds(seconds int) time.Duration {
	return time.Duration(seconds) * time.Second
}
