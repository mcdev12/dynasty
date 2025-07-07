package draft

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/mcdev12/dynasty/go/internal/draft/events"
	draftv1 "github.com/mcdev12/dynasty/go/internal/genproto/draft/v1"
	"github.com/mcdev12/dynasty/go/internal/genproto/draft/v1/draftv1connect"
	leaguev1 "github.com/mcdev12/dynasty/go/internal/genproto/league/v1"
	"github.com/mcdev12/dynasty/go/internal/genproto/league/v1/leaguev1connect"
	"github.com/mcdev12/dynasty/go/internal/models"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// DraftApp defines what the service layer needs from the draft application
type DraftApp interface {
	CreateDraft(ctx context.Context, req CreateDraftRequest) (*models.Draft, error)
	GetDraft(ctx context.Context, id uuid.UUID) (*models.Draft, error)
	UpdateDraftStatus(ctx context.Context, id uuid.UUID, status models.DraftStatus) (*models.Draft, error)
	UpdateDraft(ctx context.Context, id uuid.UUID, req UpdateDraftRequest) (*models.Draft, error)
	DeleteDraft(ctx context.Context, id uuid.UUID) error
	FetchNextDeadline(ctx context.Context) (*NextDeadline, error)
	FetchDraftsDueForPick(ctx context.Context, limit int32) ([]uuid.UUID, error)
	UpdateNextDeadline(ctx context.Context, draftID uuid.UUID, deadline *time.Time) error
	ClearNextDeadline(ctx context.Context, id uuid.UUID) error
}

// OutboxApp defines what the service layer needs from the outbox
type OutboxApp interface {
	InsertOutboxDraftStarted(ctx context.Context, draftID uuid.UUID, payload []byte) error
	InsertOutboxDraftCompleted(ctx context.Context, draftID uuid.UUID, payload []byte) error
	InsertOutboxDraftPaused(ctx context.Context, draftID uuid.UUID, payload []byte) error
	InsertOutboxDraftResumed(ctx context.Context, draftID uuid.UUID, payload []byte) error
}

// Service implements the DraftService gRPC interface
type Service struct {
	draftApp      DraftApp
	outboxApp     OutboxApp
	leagueService leaguev1connect.LeagueServiceClient
}

// NewService creates a new draft gRPC service
func NewService(draftApp DraftApp, outboxApp OutboxApp, leagueService leaguev1connect.LeagueServiceClient) *Service {
	return &Service{
		draftApp:      draftApp,
		outboxApp:     outboxApp,
		leagueService: leagueService,
	}
}

// Verify that Service implements the DraftServiceHandler interface
var _ draftv1connect.DraftServiceHandler = (*Service)(nil)

// CreateDraft creates a new draft
func (s *Service) CreateDraft(ctx context.Context, req *connect.Request[draftv1.CreateDraftRequest]) (*connect.Response[draftv1.CreateDraftResponse], error) {

	// TODO NEED TXN HANDLING HERE
	appReq, err := s.protoToCreateDraftRequest(req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	// Validate that the league exists via league service
	leagueReq := &leaguev1.GetLeagueRequest{
		Id: req.Msg.LeagueId,
	}
	leagueResp, err := s.leagueService.GetLeague(ctx, connect.NewRequest(leagueReq))
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("league not found: %w", err))
	}

	draft, err := s.draftApp.CreateDraft(ctx, appReq)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Log with league name from the service response
	log.Printf("Created draft: %s draft for league %s", draft.DraftType, leagueResp.Msg.League.Name)

	// Note: Pick prepopulation now handled by separate DraftPickService

	protoDraft, err := s.draftToProto(draft)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&draftv1.CreateDraftResponse{
		Draft: protoDraft,
	}), nil
}

// GetDraft retrieves a draft by ID
func (s *Service) GetDraft(ctx context.Context, req *connect.Request[draftv1.GetDraftRequest]) (*connect.Response[draftv1.GetDraftResponse], error) {
	id, err := uuid.Parse(req.Msg.DraftId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	draft, err := s.draftApp.GetDraft(ctx, id)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	protoDraft, err := s.draftToProto(draft)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&draftv1.GetDraftResponse{
		Draft: protoDraft,
	}), nil
}

func (s *Service) UpdateDraft(ctx context.Context, req *connect.Request[draftv1.UpdateDraftRequest]) (*connect.Response[draftv1.UpdateDraftResponse], error) {
	id, err := uuid.Parse(req.Msg.DraftId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	// Build update request
	updateReq := UpdateDraftRequest{}

	// Handle optional settings update
	if req.Msg.Settings != nil {
		settings := s.protoToDraftSettings(req.Msg.Settings)
		updateReq.Settings = &settings
	}

	// Handle optional scheduled_at update
	if req.Msg.ScheduledAt != nil {
		scheduledAt := req.Msg.ScheduledAt.AsTime()
		updateReq.ScheduledAt = &scheduledAt
	}

	// Perform the update
	draft, err := s.draftApp.UpdateDraft(ctx, id, updateReq)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Convert response to proto
	protoDraft, err := s.draftToProto(draft)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&draftv1.UpdateDraftResponse{
		Draft: protoDraft,
	}), nil
}

func (s *Service) PauseDraft(ctx context.Context, req *connect.Request[draftv1.PauseDraftRequest]) (*connect.Response[draftv1.PauseDraftResponse], error) {
	id, err := uuid.Parse(req.Msg.DraftId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	// Update draft status to paused
	draft, err := s.draftApp.UpdateDraftStatus(ctx, id, models.DraftStatusPaused)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Emit DraftPaused domain event
	if err := s.emitDraftPausedEvent(ctx, id, time.Now(), "Manual pause"); err != nil {
		log.Printf("Failed to emit DraftPaused event: %v", err)
		// Don't fail the operation, just log
	}

	log.Printf("Draft %s paused", draft.ID)
	return connect.NewResponse(&draftv1.PauseDraftResponse{}), nil
}

func (s *Service) StartDraft(ctx context.Context, req *connect.Request[draftv1.StartDraftRequest]) (*connect.Response[draftv1.StartDraftResponse], error) {
	id, err := uuid.Parse(req.Msg.DraftId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	// Update draft status to in progress
	draft, err := s.draftApp.UpdateDraftStatus(ctx, id, models.DraftStatusInProgress)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Emit DraftStarted domain event
	if err := s.emitDraftStartedEvent(ctx, id, time.Now()); err != nil {
		log.Printf("Failed to emit DraftStarted event: %v", err)
		// Don't fail the operation, just log
	}

	log.Printf("Draft %s started", draft.ID)
	return connect.NewResponse(&draftv1.StartDraftResponse{}), nil
}

func (s *Service) ResumeDraft(ctx context.Context, req *connect.Request[draftv1.ResumeDraftRequest]) (*connect.Response[draftv1.ResumeDraftResponse], error) {
	id, err := uuid.Parse(req.Msg.DraftId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	// Update draft status to in progress
	draft, err := s.draftApp.UpdateDraftStatus(ctx, id, models.DraftStatusInProgress)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Emit DraftResumed domain event
	if err := s.emitDraftResumedEvent(ctx, id, time.Now()); err != nil {
		log.Printf("Failed to emit DraftResumed event: %v", err)
		// Don't fail the operation, just log
	}

	log.Printf("Draft %s resumed", draft.ID)
	return connect.NewResponse(&draftv1.ResumeDraftResponse{}), nil
}

// DeleteDraft deletes a draft by ID
func (s *Service) DeleteDraft(ctx context.Context, req *connect.Request[draftv1.DeleteDraftRequest]) (*connect.Response[draftv1.DeleteDraftResponse], error) {
	id, err := uuid.Parse(req.Msg.DraftId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	err = s.draftApp.DeleteDraft(ctx, id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&draftv1.DeleteDraftResponse{}), nil
}

// RunScheduler is no longer part of DraftService - it belongs to Orchestrator
// This method is removed as part of the clean separation of concerns

// CompleteDraft completes a draft
func (s *Service) CompleteDraft(ctx context.Context, req *connect.Request[draftv1.CompleteDraftRequest]) (*connect.Response[draftv1.CompleteDraftResponse], error) {
	id, err := uuid.Parse(req.Msg.DraftId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	draft, err := s.draftApp.UpdateDraftStatus(ctx, id, models.DraftStatusCompleted)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Emit DraftCompleted domain event
	if err := s.emitDraftCompletedEvent(ctx, id, time.Now()); err != nil {
		log.Printf("Failed to emit DraftCompleted event: %v", err)
		// Don't fail the operation, just log
	}

	protoDraft, err := s.draftToProto(draft)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	log.Printf("Draft %s completed", draft.ID)
	return connect.NewResponse(&draftv1.CompleteDraftResponse{
		Draft: protoDraft,
	}), nil
}

// FetchNextDeadline fetches the next deadline across all active drafts
func (s *Service) FetchNextDeadline(ctx context.Context, req *connect.Request[draftv1.FetchNextDeadlineRequest]) (*connect.Response[draftv1.FetchNextDeadlineResponse], error) {
	deadline, err := s.draftApp.FetchNextDeadline(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// No deadline found - return empty response
			return connect.NewResponse(&draftv1.FetchNextDeadlineResponse{}), nil
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	protoDeadline := &draftv1.NextDeadline{
		DraftId: deadline.DraftID.String(),
	}
	if deadline.Deadline != nil {
		protoDeadline.Deadline = timestamppb.New(*deadline.Deadline)
	}

	return connect.NewResponse(&draftv1.FetchNextDeadlineResponse{
		NextDeadline: protoDeadline,
	}), nil
}

// FetchDraftsDueForPick fetches drafts that are due for a pick
func (s *Service) FetchDraftsDueForPick(ctx context.Context, req *connect.Request[draftv1.FetchDraftsDueForPickRequest]) (*connect.Response[draftv1.FetchDraftsDueForPickResponse], error) {
	draftIDs, err := s.draftApp.FetchDraftsDueForPick(ctx, req.Msg.Limit)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Convert UUIDs to strings
	draftIDStrings := make([]string, len(draftIDs))
	for i, id := range draftIDs {
		draftIDStrings[i] = id.String()
	}

	return connect.NewResponse(&draftv1.FetchDraftsDueForPickResponse{
		DraftIds: draftIDStrings,
	}), nil
}

// UpdateNextDeadline updates the next deadline for a draft
func (s *Service) UpdateNextDeadline(ctx context.Context, req *connect.Request[draftv1.UpdateNextDeadlineRequest]) (*connect.Response[draftv1.UpdateNextDeadlineResponse], error) {
	draftID, err := uuid.Parse(req.Msg.DraftId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	var deadline *time.Time
	if req.Msg.Deadline != nil {
		t := req.Msg.Deadline.AsTime()
		deadline = &t
	}

	if err := s.draftApp.UpdateNextDeadline(ctx, draftID, deadline); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&draftv1.UpdateNextDeadlineResponse{}), nil
}

// ClearNextDeadline clears the deadline for a draft
func (s *Service) ClearNextDeadline(ctx context.Context, req *connect.Request[draftv1.ClearNextDeadlineRequest]) (*connect.Response[draftv1.ClearNextDeadlineResponse], error) {
	draftID, err := uuid.Parse(req.Msg.DraftId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	if err := s.draftApp.ClearNextDeadline(ctx, draftID); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&draftv1.ClearNextDeadlineResponse{}), nil
}

// Conversion methods between proto and app layer models

func (s *Service) draftToProto(draft *models.Draft) (*draftv1.Draft, error) {
	protoDraft := &draftv1.Draft{
		Id:        draft.ID.String(),
		LeagueId:  draft.LeagueID.String(),
		DraftType: s.draftTypeToProto(draft.DraftType),
		Status:    s.draftStatusToProto(draft.Status),
		Settings:  s.draftSettingsToProto(draft.Settings),
		CreatedAt: timestamppb.New(draft.CreatedAt),
		UpdatedAt: timestamppb.New(draft.UpdatedAt),
	}

	if draft.ScheduledAt != nil {
		protoDraft.ScheduledAt = timestamppb.New(*draft.ScheduledAt)
	}
	if draft.StartedAt != nil {
		protoDraft.StartedAt = timestamppb.New(*draft.StartedAt)
	}
	if draft.CompletedAt != nil {
		protoDraft.CompletedAt = timestamppb.New(*draft.CompletedAt)
	}

	return protoDraft, nil
}

func (s *Service) protoToCreateDraftRequest(proto *draftv1.CreateDraftRequest) (CreateDraftRequest, error) {
	leagueID, err := uuid.Parse(proto.LeagueId)
	if err != nil {
		return CreateDraftRequest{}, err
	}

	req := CreateDraftRequest{
		ID:        uuid.New(), // Generate new UUID for draft
		LeagueID:  leagueID,
		DraftType: s.protoToDraftType(proto.DraftType),
		Status:    models.DraftStatusNotStarted, // Always start as NOT_STARTED
		Settings:  s.protoToDraftSettings(proto.Settings),
	}

	if proto.ScheduledAt != nil {
		scheduledAt := proto.ScheduledAt.AsTime()
		req.ScheduledAt = &scheduledAt
	}

	return req, nil
}

func (s *Service) draftSettingsToProto(settings models.DraftSettings) *draftv1.DraftSettings {
	protoSettings := &draftv1.DraftSettings{
		Rounds:             int32(settings.Rounds),
		TimePerPickSec:     int32(settings.TimePerPickSec),
		ThirdRoundReversal: settings.ThirdRoundReversal,
	}

	// Convert draft order UUIDs to strings
	if len(settings.DraftOrder) > 0 {
		protoSettings.DraftOrder = make([]string, len(settings.DraftOrder))
		for i, teamID := range settings.DraftOrder {
			protoSettings.DraftOrder[i] = teamID.String()
		}
	}

	// Set optional auction fields
	if settings.BudgetPerTeam != nil {
		protoSettings.BudgetPerTeam = settings.BudgetPerTeam
	}
	if settings.MinBidIncrement != nil {
		protoSettings.MinBidIncrement = settings.MinBidIncrement
	}
	if settings.TimePerNominationSec != nil {
		timePerNom := int32(*settings.TimePerNominationSec)
		protoSettings.TimePerNominationSec = &timePerNom
	}

	return protoSettings
}

func (s *Service) protoToDraftSettings(proto *draftv1.DraftSettings) models.DraftSettings {
	settings := models.DraftSettings{
		Rounds:             int(proto.Rounds),
		TimePerPickSec:     int(proto.TimePerPickSec),
		ThirdRoundReversal: proto.ThirdRoundReversal,
		BudgetPerTeam:      proto.BudgetPerTeam,
		MinBidIncrement:    proto.MinBidIncrement,
	}

	// Convert optional int32 to int pointer
	if proto.TimePerNominationSec != nil {
		timePerNom := int(*proto.TimePerNominationSec)
		settings.TimePerNominationSec = &timePerNom
	}

	// Convert draft order strings to UUIDs
	if len(proto.DraftOrder) > 0 {
		settings.DraftOrder = make([]uuid.UUID, len(proto.DraftOrder))
		for i, teamIDStr := range proto.DraftOrder {
			if teamID, err := uuid.Parse(teamIDStr); err == nil {
				settings.DraftOrder[i] = teamID
			}
		}
	}

	return settings
}

// Enum conversion methods

func (s *Service) draftTypeToProto(draftType models.DraftType) draftv1.DraftType {
	switch draftType {
	case models.DraftTypeSnake:
		return draftv1.DraftType_DRAFT_TYPE_SNAKE
	case models.DraftTypeAuction:
		return draftv1.DraftType_DRAFT_TYPE_AUCTION
	case models.DraftTypeRookie:
		return draftv1.DraftType_DRAFT_TYPE_ROOKIE
	default:
		return draftv1.DraftType_DRAFT_TYPE_UNSPECIFIED
	}
}

func (s *Service) protoToDraftType(protoType draftv1.DraftType) models.DraftType {
	switch protoType {
	case draftv1.DraftType_DRAFT_TYPE_SNAKE:
		return models.DraftTypeSnake
	case draftv1.DraftType_DRAFT_TYPE_AUCTION:
		return models.DraftTypeAuction
	case draftv1.DraftType_DRAFT_TYPE_ROOKIE:
		return models.DraftTypeRookie
	default:
		return models.DraftTypeSnake // default fallback
	}
}

func (s *Service) draftStatusToProto(status models.DraftStatus) draftv1.DraftStatus {
	switch status {
	case models.DraftStatusNotStarted:
		return draftv1.DraftStatus_DRAFT_STATUS_NOT_STARTED
	case models.DraftStatusInProgress:
		return draftv1.DraftStatus_DRAFT_STATUS_IN_PROGRESS
	case models.DraftStatusPaused:
		return draftv1.DraftStatus_DRAFT_STATUS_PAUSED
	case models.DraftStatusCompleted:
		return draftv1.DraftStatus_DRAFT_STATUS_COMPLETED
	case models.DraftStatusCancelled:
		return draftv1.DraftStatus_DRAFT_STATUS_CANCELLED
	default:
		return draftv1.DraftStatus_DRAFT_STATUS_UNSPECIFIED
	}
}

func (s *Service) protoToDraftStatus(protoStatus draftv1.DraftStatus) models.DraftStatus {
	switch protoStatus {
	case draftv1.DraftStatus_DRAFT_STATUS_NOT_STARTED:
		return models.DraftStatusNotStarted
	case draftv1.DraftStatus_DRAFT_STATUS_IN_PROGRESS:
		return models.DraftStatusInProgress
	case draftv1.DraftStatus_DRAFT_STATUS_PAUSED:
		return models.DraftStatusPaused
	case draftv1.DraftStatus_DRAFT_STATUS_COMPLETED:
		return models.DraftStatusCompleted
	case draftv1.DraftStatus_DRAFT_STATUS_CANCELLED:
		return models.DraftStatusCancelled
	default:
		return models.DraftStatusNotStarted // default fallback
	}
}

// Event emission helper methods

// emitDraftStartedEvent emits a DraftStarted event to the outbox
func (s *Service) emitDraftStartedEvent(ctx context.Context, draftID uuid.UUID, startedAt time.Time) error {
	// Get draft information to include in the event
	draft, err := s.draftApp.GetDraft(ctx, draftID)
	if err != nil {
		return fmt.Errorf("failed to get draft for DraftStarted event: %w", err)
	}

	// Count total picks for the draft
	totalPicks := draft.Settings.Rounds * len(draft.Settings.DraftOrder)

	// Create DraftStarted payload
	payload := events.DraftStartedPayload{
		DraftID:     draftID.String(),
		DraftType:   string(draft.DraftType),
		StartedAt:   startedAt,
		TotalRounds: draft.Settings.Rounds,
		TotalPicks:  totalPicks,
	}

	// Marshal payload to JSON
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal DraftStarted payload: %w", err)
	}

	// Insert into outbox
	return s.outboxApp.InsertOutboxDraftStarted(ctx, draftID, payloadBytes)
}

// emitDraftPausedEvent emits a DraftPaused event to the outbox
func (s *Service) emitDraftPausedEvent(ctx context.Context, draftID uuid.UUID, pausedAt time.Time, reason string) error {
	// Create DraftPaused payload
	payload := events.DraftPausedPayload{
		DraftID:  draftID.String(),
		PausedAt: pausedAt,
		Reason:   reason,
	}

	// Marshal payload to JSON
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal DraftPaused payload: %w", err)
	}

	// Insert into outbox
	return s.outboxApp.InsertOutboxDraftPaused(ctx, draftID, payloadBytes)
}

// emitDraftResumedEvent emits a DraftResumed event to the outbox
func (s *Service) emitDraftResumedEvent(ctx context.Context, draftID uuid.UUID, resumedAt time.Time) error {
	// Create DraftResumed payload
	payload := events.DraftResumedPayload{
		DraftID:   draftID.String(),
		ResumedAt: resumedAt,
	}

	// Marshal payload to JSON
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal DraftResumed payload: %w", err)
	}

	// Insert into outbox
	return s.outboxApp.InsertOutboxDraftResumed(ctx, draftID, payloadBytes)
}

// emitDraftCompletedEvent emits a DraftCompleted event to the outbox
func (s *Service) emitDraftCompletedEvent(ctx context.Context, draftID uuid.UUID, completedAt time.Time) error {
	// Get draft information to calculate duration
	draft, err := s.draftApp.GetDraft(ctx, draftID)
	if err != nil {
		return fmt.Errorf("failed to get draft for DraftCompleted event: %w", err)
	}

	// Calculate duration
	var duration string
	if draft.StartedAt != nil {
		duration = completedAt.Sub(*draft.StartedAt).String()
	}

	// Count total picks for the draft
	totalPicks := draft.Settings.Rounds * len(draft.Settings.DraftOrder)

	// Create DraftCompleted payload
	payload := events.DraftCompletedPayload{
		DraftID:     draftID.String(),
		CompletedAt: completedAt,
		Duration:    duration,
		TotalPicks:  totalPicks,
	}

	// Marshal payload to JSON
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal DraftCompleted payload: %w", err)
	}

	// Insert into outbox
	return s.outboxApp.InsertOutboxDraftCompleted(ctx, draftID, payloadBytes)
}
