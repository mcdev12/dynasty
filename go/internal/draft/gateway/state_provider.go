package gateway

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	draftv1 "github.com/mcdev12/dynasty/go/internal/genproto/draft/v1"
	"github.com/mcdev12/dynasty/go/internal/genproto/draft/v1/draftv1connect"
)

// DraftStateProvider implements StateProvider using the draft service client
type DraftStateProvider struct {
	draftService     draftv1connect.DraftServiceClient
	draftPickService draftv1connect.DraftPickServiceClient
}

// NewDraftStateProvider creates a new draft state provider
func NewDraftStateProvider(draftService draftv1connect.DraftServiceClient, draftPickService draftv1connect.DraftPickServiceClient) *DraftStateProvider {
	return &DraftStateProvider{
		draftService:     draftService,
		draftPickService: draftPickService,
	}
}

// GetDraftState retrieves the complete state of a draft
func (p *DraftStateProvider) GetDraftState(ctx context.Context, draftID uuid.UUID) (*DraftStateResponse, error) {
	// Get draft details via service
	getDraftReq := &draftv1.GetDraftRequest{
		DraftId: draftID.String(),
	}
	draftResp, err := p.draftService.GetDraft(ctx, connect.NewRequest(getDraftReq))
	if err != nil {
		return nil, fmt.Errorf("failed to get draft: %w", err)
	}
	draft := draftResp.Msg.Draft

	// Build response
	response := &DraftStateResponse{
		DraftID: draftID.String(),
		Status:  draft.Status.String(),
		Metadata: map[string]interface{}{
			"league_id":    draft.LeagueId,
			"draft_type":   draft.DraftType.String(),
			"total_rounds": draft.Settings.Rounds,
			"total_teams":  len(draft.Settings.DraftOrder),
		},
	}

	// Calculate total picks
	response.TotalPicks = int(draft.Settings.Rounds) * len(draft.Settings.DraftOrder)

	// Get pick information if draft is in progress
	if draft.Status == draftv1.DraftStatus_DRAFT_STATUS_IN_PROGRESS {
		// Get current pick on the clock via draft pick service
		getNextPickReq := &draftv1.GetNextPickForDraftRequest{
			DraftId: draftID.String(),
		}
		nextPickResp, err := p.draftPickService.GetNextPickForDraft(ctx, connect.NewRequest(getNextPickReq))
		if err == nil && nextPickResp.Msg.Pick != nil && nextPickResp.Msg.Pick.PlayerId == "" {
			// This pick hasn't been made yet, so it's the current pick
			currentPick := nextPickResp.Msg.Pick
			response.CurrentPick = &CurrentPickInfo{
				PickID:      currentPick.Id,
				TeamID:      currentPick.TeamId,
				TeamName:    fmt.Sprintf("Team %s", currentPick.TeamId[:8]), // TODO: Get actual team name
				Round:       int(currentPick.Round),
				Pick:        int(currentPick.Pick),
				OverallPick: int(currentPick.OverallPick),
				TimePerPick: int(draft.Settings.TimePerPickSec),
			}

			// Get next deadline separately for timer information
			deadlineResp, err := p.draftService.FetchNextDeadline(ctx, connect.NewRequest(&draftv1.FetchNextDeadlineRequest{}))
			if err == nil && deadlineResp.Msg.NextDeadline != nil && deadlineResp.Msg.NextDeadline.DraftId == draftID.String() {
				if deadlineResp.Msg.NextDeadline.Deadline != nil {
					response.CurrentPick.TimeoutAt = deadlineResp.Msg.NextDeadline.Deadline.AsTime()
					// StartedAt would be TimeoutAt minus TimePerPick
					response.CurrentPick.StartedAt = deadlineResp.Msg.NextDeadline.Deadline.AsTime().Add(-timeDurationFromSeconds(int(draft.Settings.TimePerPickSec)))
				}
			}
		}

		// TODO: Get recent picks (last 5-10 picks)
		// This would require a new method in DraftPickService to fetch recent picks
		response.RecentPicks = []RecentPickInfo{}

		// Count completed picks via draft pick service
		countReq := &draftv1.CountRemainingPicksRequest{
			DraftId: draftID.String(),
		}
		countResp, err := p.draftPickService.CountRemainingPicks(ctx, connect.NewRequest(countReq))
		if err == nil {
			response.CompletedPicks = response.TotalPicks - int(countResp.Msg.RemainingPicks)
		}
	}

	return response, nil
}

// GetActiveDrafts retrieves all active drafts
func (p *DraftStateProvider) GetActiveDrafts(ctx context.Context) ([]DraftSummary, error) {
	// TODO: This would require a new method in DraftService to list drafts by status
	// For now, return empty list until we add ListDraftsByStatus to the protobuf service
	return []DraftSummary{}, nil
}

// timeDurationFromSeconds converts seconds to time.Duration
func timeDurationFromSeconds(seconds int) time.Duration {
	return time.Duration(seconds) * time.Second
}
