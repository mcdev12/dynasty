package draft

import (
	"context"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	draftv1 "github.com/mcdev12/dynasty/go/internal/genproto/draft/v1"
	"github.com/mcdev12/dynasty/go/internal/genproto/draft/v1/draftv1connect"
	"github.com/mcdev12/dynasty/go/internal/models"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// DraftApp defines what the service layer needs from the draft application
type DraftApp interface {
	CreateDraft(ctx context.Context, req CreateDraftRequest) (*models.Draft, error)
	GetDraft(ctx context.Context, id uuid.UUID) (*models.Draft, error)
	UpdateDraftStatus(ctx context.Context, id uuid.UUID, req UpdateDraftStatusRequest) (*models.Draft, error)
	DeleteDraft(ctx context.Context, id uuid.UUID) error
	PrepopulateDraftPicks(ctx context.Context, draftID uuid.UUID) error
}

// Service implements the DraftService gRPC interface
type Service struct {
	app DraftApp
}

// NewService creates a new draft gRPC service
func NewService(app DraftApp) *Service {
	return &Service{
		app: app,
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

	draft, err := s.app.CreateDraft(ctx, appReq)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Populate Picks
	err = s.app.PrepopulateDraftPicks(ctx, draft.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

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

	draft, err := s.app.GetDraft(ctx, id)
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

// ListDraftsForLeague - not implemented in app layer yet, return unimplemented
func (s *Service) ListDraftsForLeague(ctx context.Context, req *connect.Request[draftv1.ListDraftsForLeagueRequest]) (*connect.Response[draftv1.ListDraftsForLeagueResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}

// UpdateDraft updates a draft (currently only supports status updates)
func (s *Service) UpdateDraft(ctx context.Context, req *connect.Request[draftv1.UpdateDraftRequest]) (*connect.Response[draftv1.UpdateDraftResponse], error) {
	id, err := uuid.Parse(req.Msg.DraftId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	// Currently only supporting status updates
	if req.Msg.Status == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	appReq := UpdateDraftStatusRequest{
		Status: s.protoToDraftStatus(*req.Msg.Status),
	}

	draft, err := s.app.UpdateDraftStatus(ctx, id, appReq)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	protoDraft, err := s.draftToProto(draft)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&draftv1.UpdateDraftResponse{
		Draft: protoDraft,
	}), nil
}

// DeleteDraft deletes a draft by ID
func (s *Service) DeleteDraft(ctx context.Context, req *connect.Request[draftv1.DeleteDraftRequest]) (*connect.Response[draftv1.DeleteDraftResponse], error) {
	id, err := uuid.Parse(req.Msg.DraftId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	err = s.app.DeleteDraft(ctx, id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&draftv1.DeleteDraftResponse{}), nil
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
