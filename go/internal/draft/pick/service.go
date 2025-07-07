package pick

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/mcdev12/dynasty/go/internal/draft/events"
	draftv1 "github.com/mcdev12/dynasty/go/internal/genproto/draft/v1"
	"github.com/mcdev12/dynasty/go/internal/genproto/draft/v1/draftv1connect"
	"github.com/mcdev12/dynasty/go/internal/models"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// PickApp defines what the service layer needs from the pick application
type PickApp interface {
	PrepopulateDraftPicks(ctx context.Context, draftID uuid.UUID, draftType models.DraftType, settings models.DraftSettings) error
	MakePick(ctx context.Context, req MakePickRequest) error
	GetDraftPick(ctx context.Context, pickID uuid.UUID) (*models.DraftPick, error)
	GetDraftPicksByDraft(ctx context.Context, draftID uuid.UUID) ([]models.DraftPick, error)
	GetDraftPicksByRound(ctx context.Context, draftID uuid.UUID, round int) ([]models.DraftPick, error)
	GetNextPickForDraft(ctx context.Context, draftID uuid.UUID) (*models.DraftPick, error)
	CountRemainingPicks(ctx context.Context, draftID uuid.UUID) (int, error)
	ClaimNextPickSlot(ctx context.Context, draftID uuid.UUID) (*Slot, error)
	ListAvailablePlayersForDraft(ctx context.Context, draftID uuid.UUID) ([]AvailablePlayer, error)
	UpdateDraftPickPlayer(ctx context.Context, pickID uuid.UUID, req UpdateDraftPickPlayerRequest) (*models.DraftPick, error)
	DeleteDraftPicksByDraft(ctx context.Context, draftID uuid.UUID) (int, error)
}

// OutboxApp defines what the service layer needs from the outbox
type OutboxApp interface {
	InsertPickMadeEvent(ctx context.Context, draftID uuid.UUID, payload []byte) error
}

// Service implements the DraftPickService gRPC interface
type Service struct {
	app          PickApp
	draftService draftv1connect.DraftServiceClient
	outboxApp    OutboxApp
}

// NewService creates a new draft pick gRPC service
func NewService(app PickApp, draftService draftv1connect.DraftServiceClient, outboxApp OutboxApp) *Service {
	return &Service{
		app:          app,
		draftService: draftService,
		outboxApp:    outboxApp,
	}
}

// Verify that Service implements the DraftPickServiceHandler interface
var _ draftv1connect.DraftPickServiceHandler = (*Service)(nil)

// MakePick makes a draft pick
func (s *Service) MakePick(ctx context.Context, req *connect.Request[draftv1.MakePickRequest]) (*connect.Response[draftv1.MakePickResponse], error) {
	appReq, err := s.protoToMakePickRequest(req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	err = s.app.MakePick(ctx, appReq)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Get the updated pick to return
	pick, err := s.app.GetDraftPick(ctx, appReq.PickID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	protoPick, err := s.draftPickToProto(pick)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Emit PickMade domain event
	if err := s.emitPickMadeEvent(ctx, appReq.DraftID, protoPick); err != nil {
		log.Printf("Failed to emit PickMade event: %v", err)
		// Don't fail the operation, just log
	}

	log.Printf("Pick made: %s for team %s in draft %s", appReq.PlayerID, appReq.TeamID, appReq.DraftID)

	return connect.NewResponse(&draftv1.MakePickResponse{
		Pick: protoPick,
	}), nil
}

// GetDraftPick retrieves a draft pick by ID
func (s *Service) GetDraftPick(ctx context.Context, req *connect.Request[draftv1.GetDraftPickRequest]) (*connect.Response[draftv1.GetDraftPickResponse], error) {
	pickID, err := uuid.Parse(req.Msg.PickId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	pick, err := s.app.GetDraftPick(ctx, pickID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	protoPick, err := s.draftPickToProto(pick)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&draftv1.GetDraftPickResponse{
		Pick: protoPick,
	}), nil
}

// GetDraftPicksByDraft retrieves all picks for a draft
func (s *Service) GetDraftPicksByDraft(ctx context.Context, req *connect.Request[draftv1.GetDraftPicksByDraftRequest]) (*connect.Response[draftv1.GetDraftPicksByDraftResponse], error) {
	draftID, err := uuid.Parse(req.Msg.DraftId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	picks, err := s.app.GetDraftPicksByDraft(ctx, draftID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	protoPicks := make([]*draftv1.DraftPick, len(picks))
	for i, pick := range picks {
		protoPick, err := s.draftPickToProto(&pick)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		protoPicks[i] = protoPick
	}

	return connect.NewResponse(&draftv1.GetDraftPicksByDraftResponse{
		Picks: protoPicks,
	}), nil
}

// GetDraftPicksByRound retrieves picks for a specific round
func (s *Service) GetDraftPicksByRound(ctx context.Context, req *connect.Request[draftv1.GetDraftPicksByRoundRequest]) (*connect.Response[draftv1.GetDraftPicksByRoundResponse], error) {
	draftID, err := uuid.Parse(req.Msg.DraftId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	picks, err := s.app.GetDraftPicksByRound(ctx, draftID, int(req.Msg.Round))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	protoPicks := make([]*draftv1.DraftPick, len(picks))
	for i, pick := range picks {
		protoPick, err := s.draftPickToProto(&pick)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		protoPicks[i] = protoPick
	}

	return connect.NewResponse(&draftv1.GetDraftPicksByRoundResponse{
		Picks: protoPicks,
	}), nil
}

// GetNextPickForDraft retrieves the next pick for a draft
func (s *Service) GetNextPickForDraft(ctx context.Context, req *connect.Request[draftv1.GetNextPickForDraftRequest]) (*connect.Response[draftv1.GetNextPickForDraftResponse], error) {
	draftID, err := uuid.Parse(req.Msg.DraftId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	pick, err := s.app.GetNextPickForDraft(ctx, draftID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	protoPick, err := s.draftPickToProto(pick)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&draftv1.GetNextPickForDraftResponse{
		Pick: protoPick,
	}), nil
}

// CountRemainingPicks counts remaining picks for a draft
func (s *Service) CountRemainingPicks(ctx context.Context, req *connect.Request[draftv1.CountRemainingPicksRequest]) (*connect.Response[draftv1.CountRemainingPicksResponse], error) {
	draftID, err := uuid.Parse(req.Msg.DraftId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	count, err := s.app.CountRemainingPicks(ctx, draftID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&draftv1.CountRemainingPicksResponse{
		RemainingPicks: int32(count),
	}), nil
}

// ClaimNextPickSlot claims the next pick slot for auto-pick
func (s *Service) ClaimNextPickSlot(ctx context.Context, req *connect.Request[draftv1.ClaimNextPickSlotRequest]) (*connect.Response[draftv1.ClaimNextPickSlotResponse], error) {
	draftID, err := uuid.Parse(req.Msg.DraftId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	// Validate draft exists and is in progress via draft service
	getDraftReq := &draftv1.GetDraftRequest{
		DraftId: draftID.String(),
	}
	draftResp, err := s.draftService.GetDraft(ctx, connect.NewRequest(getDraftReq))
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("draft not found: %w", err))
	}

	if draftResp.Msg.Draft.Status != draftv1.DraftStatus_DRAFT_STATUS_IN_PROGRESS {
		return nil, connect.NewError(connect.CodeFailedPrecondition,
			fmt.Errorf("can only claim pick slots for drafts in progress, current status is %s",
				draftResp.Msg.Draft.Status.String()))
	}

	slot, err := s.app.ClaimNextPickSlot(ctx, draftID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	protoSlot := &draftv1.PickSlot{
		PickId:      slot.PickID.String(),
		TeamId:      slot.TeamID.String(),
		OverallPick: int32(slot.OverallPick),
	}

	return connect.NewResponse(&draftv1.ClaimNextPickSlotResponse{
		Slot: protoSlot,
	}), nil
}

// PrepopulateDraftPicks prepopulates draft picks
func (s *Service) PrepopulateDraftPicks(ctx context.Context, req *connect.Request[draftv1.PrepopulateDraftPicksRequest]) (*connect.Response[draftv1.PrepopulateDraftPicksResponse], error) {
	draftID, err := uuid.Parse(req.Msg.DraftId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	draftType := s.protoToDraftType(req.Msg.DraftType)
	settings := s.protoToDraftSettings(req.Msg.Settings)

	err = s.app.PrepopulateDraftPicks(ctx, draftID, draftType, settings)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Count picks created for response
	totalPicks := settings.Rounds * len(settings.DraftOrder)

	return connect.NewResponse(&draftv1.PrepopulateDraftPicksResponse{
		PicksCreated: int32(totalPicks),
	}), nil
}

// ListAvailablePlayersForDraft lists available players for a draft
func (s *Service) ListAvailablePlayersForDraft(ctx context.Context, req *connect.Request[draftv1.ListAvailablePlayersForDraftRequest]) (*connect.Response[draftv1.ListAvailablePlayersForDraftResponse], error) {
	draftID, err := uuid.Parse(req.Msg.DraftId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	// Validate draft exists via draft service
	getDraftReq := &draftv1.GetDraftRequest{
		DraftId: draftID.String(),
	}
	_, err = s.draftService.GetDraft(ctx, connect.NewRequest(getDraftReq))
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("draft not found: %w", err))
	}

	players, err := s.app.ListAvailablePlayersForDraft(ctx, draftID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	protoPlayers := make([]*draftv1.AvailablePlayer, len(players))
	for i, player := range players {
		protoPlayers[i] = &draftv1.AvailablePlayer{
			Id:       player.ID.String(),
			FullName: player.FullName,
			TeamId:   player.TeamID.String(),
		}
	}

	return connect.NewResponse(&draftv1.ListAvailablePlayersForDraftResponse{
		Players: protoPlayers,
	}), nil
}

// UpdateDraftPickPlayer updates a draft pick's player
func (s *Service) UpdateDraftPickPlayer(ctx context.Context, req *connect.Request[draftv1.UpdateDraftPickPlayerRequest]) (*connect.Response[draftv1.UpdateDraftPickPlayerResponse], error) {
	pickID, err := uuid.Parse(req.Msg.PickId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	playerID, err := uuid.Parse(req.Msg.PlayerId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	updateReq := UpdateDraftPickPlayerRequest{
		PlayerID:   playerID,
		KeeperPick: req.Msg.KeeperPick,
	}

	if req.Msg.AuctionAmount != nil {
		updateReq.AuctionAmount = req.Msg.AuctionAmount
	}

	pick, err := s.app.UpdateDraftPickPlayer(ctx, pickID, updateReq)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	protoPick, err := s.draftPickToProto(pick)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&draftv1.UpdateDraftPickPlayerResponse{
		Pick: protoPick,
	}), nil
}

// DeleteDraftPicksByDraft deletes all picks for a draft
func (s *Service) DeleteDraftPicksByDraft(ctx context.Context, req *connect.Request[draftv1.DeleteDraftPicksByDraftRequest]) (*connect.Response[draftv1.DeleteDraftPicksByDraftResponse], error) {
	draftID, err := uuid.Parse(req.Msg.DraftId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	count, err := s.app.DeleteDraftPicksByDraft(ctx, draftID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&draftv1.DeleteDraftPicksByDraftResponse{
		DeletedCount: int32(count),
	}), nil
}

// Conversion methods between proto and app layer models

func (s *Service) protoToMakePickRequest(proto *draftv1.MakePickRequest) (MakePickRequest, error) {
	pickID, err := uuid.Parse(proto.PickId)
	if err != nil {
		return MakePickRequest{}, err
	}
	draftID, err := uuid.Parse(proto.DraftId)
	if err != nil {
		return MakePickRequest{}, err
	}
	teamID, err := uuid.Parse(proto.TeamId)
	if err != nil {
		return MakePickRequest{}, err
	}
	playerID, err := uuid.Parse(proto.PlayerId)
	if err != nil {
		return MakePickRequest{}, err
	}

	return MakePickRequest{
		PickID:      pickID,
		DraftID:     draftID,
		TeamID:      teamID,
		PlayerID:    playerID,
		OverallPick: int(proto.OverallPick),
	}, nil
}

func (s *Service) draftPickToProto(pick *models.DraftPick) (*draftv1.DraftPick, error) {
	protoPick := &draftv1.DraftPick{
		Id:          pick.ID.String(),
		DraftId:     pick.DraftID.String(),
		Round:       int32(pick.Round),
		Pick:        int32(pick.Pick),
		OverallPick: int32(pick.OverallPick),
		TeamId:      pick.TeamID.String(),
		KeeperPick:  pick.KeeperPick,
	}

	if pick.PlayerID != nil {
		protoPick.PlayerId = pick.PlayerID.String()
	}

	if pick.PickedAt != nil {
		protoPick.PickedAt = timestamppb.New(*pick.PickedAt)
	}

	if pick.AuctionAmount != nil {
		protoPick.AuctionAmount = pick.AuctionAmount
	}

	return protoPick, nil
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

// Event emission helper method

// emitPickMadeEvent emits a PickMade event to the outbox
func (s *Service) emitPickMadeEvent(ctx context.Context, draftID uuid.UUID, pick *draftv1.DraftPick) error {
	madeAt := time.Now()

	// Create PickMade payload
	payload := events.PickMadePayload{
		PickID:      pick.Id,
		TeamID:      pick.TeamId,
		PlayerID:    pick.PlayerId,
		Round:       int(pick.Round),
		Pick:        int(pick.Pick),
		OverallPick: int(pick.OverallPick),
		MadeAt:      madeAt,
	}

	// Marshal payload to JSON
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal PickMade payload: %w", err)
	}

	// Insert into outbox
	return s.outboxApp.InsertPickMadeEvent(ctx, draftID, payloadBytes)
}
