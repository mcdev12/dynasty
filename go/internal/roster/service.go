package roster

import (
	"context"
	"encoding/json"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	rosterv1 "github.com/mcdev12/dynasty/go/internal/genproto/roster/v1"
	"github.com/mcdev12/dynasty/go/internal/genproto/roster/v1/rosterv1connect"
	"github.com/mcdev12/dynasty/go/internal/models"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// RosterApp defines what the service layer needs from the roster application
type RosterApp interface {
	CreateRoster(ctx context.Context, req CreateRosterRequest) (*models.Roster, error)
	GetRoster(ctx context.Context, id uuid.UUID) (*models.Roster, error)
	GetRosterPlayersByFantasyTeam(ctx context.Context, fantasyTeamID uuid.UUID) ([]models.Roster, error)
	GetRosterPlayersByFantasyTeamAndPosition(ctx context.Context, fantasyTeamID uuid.UUID, position models.RosterPosition) ([]models.Roster, error)
	GetPlayerOnRoster(ctx context.Context, fantasyTeamID, playerID uuid.UUID) (*models.Roster, error)
	GetStartingRosterPlayers(ctx context.Context, fantasyTeamID uuid.UUID) ([]models.Roster, error)
	GetBenchRosterPlayers(ctx context.Context, fantasyTeamID uuid.UUID) ([]models.Roster, error)
	GetRosterPlayersByAcquisitionType(ctx context.Context, fantasyTeamID uuid.UUID, acquisitionType models.AcquisitionType) ([]models.Roster, error)
	UpdateRosterPlayerPosition(ctx context.Context, id uuid.UUID, req UpdateRosterPositionRequest) (*models.Roster, error)
	UpdateRosterPlayerKeeperData(ctx context.Context, id uuid.UUID, req UpdateRosterKeeperDataRequest) (*models.Roster, error)
	UpdateRosterPositionAndKeeperData(ctx context.Context, id uuid.UUID, req UpdateRosterPositionAndKeeperDataRequest) (*models.Roster, error)
	DeleteRosterEntry(ctx context.Context, id uuid.UUID) error
	DeletePlayerFromRoster(ctx context.Context, fantasyTeamID, playerID uuid.UUID) error
	DeleteTeamRoster(ctx context.Context, fantasyTeamID uuid.UUID) error
}

// Service implements the RosterService gRPC interface
type Service struct {
	app RosterApp
}

// NewService creates a new roster gRPC service
func NewService(app RosterApp) *Service {
	return &Service{
		app: app,
	}
}

// Verify that Service implements the RosterServiceHandler interface
var _ rosterv1connect.RosterServiceHandler = (*Service)(nil)

// CreateRoster adds a player to a fantasy team's roster
func (s *Service) CreateRoster(ctx context.Context, req *connect.Request[rosterv1.CreateRosterRequest]) (*connect.Response[rosterv1.CreateRosterResponse], error) {
	appReq, err := s.protoToCreateRosterRequest(req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	roster, err := s.app.CreateRoster(ctx, appReq)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	protoRoster, err := s.rosterToProto(roster)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&rosterv1.CreateRosterResponse{
		Roster: protoRoster,
	}), nil
}

// GetRoster retrieves a roster entry by ID
func (s *Service) GetRoster(ctx context.Context, req *connect.Request[rosterv1.GetRosterRequest]) (*connect.Response[rosterv1.GetRosterResponse], error) {
	id, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	roster, err := s.app.GetRoster(ctx, id)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	protoRoster, err := s.rosterToProto(roster)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&rosterv1.GetRosterResponse{
		Roster: protoRoster,
	}), nil
}

// GetRosterPlayersByFantasyTeam retrieves all players on a team's roster
func (s *Service) GetRosterPlayersByFantasyTeam(ctx context.Context, req *connect.Request[rosterv1.GetRosterPlayersByFantasyTeamRequest]) (*connect.Response[rosterv1.GetRosterPlayersByFantasyTeamResponse], error) {
	fantasyTeamID, err := uuid.Parse(req.Msg.FantasyTeamId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	rosters, err := s.app.GetRosterPlayersByFantasyTeam(ctx, fantasyTeamID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	protoRosters, err := s.rostersToProto(rosters)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&rosterv1.GetRosterPlayersByFantasyTeamResponse{
		Rosters: protoRosters,
	}), nil
}

// GetRosterPlayersByFantasyTeamAndPosition retrieves players by team and position
func (s *Service) GetRosterPlayersByFantasyTeamAndPosition(ctx context.Context, req *connect.Request[rosterv1.GetRosterPlayersByFantasyTeamAndPositionRequest]) (*connect.Response[rosterv1.GetRosterPlayersByFantasyTeamAndPositionResponse], error) {
	fantasyTeamID, err := uuid.Parse(req.Msg.FantasyTeamId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	position := s.protoToRosterPosition(req.Msg.Position)

	rosters, err := s.app.GetRosterPlayersByFantasyTeamAndPosition(ctx, fantasyTeamID, position)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	protoRosters, err := s.rostersToProto(rosters)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&rosterv1.GetRosterPlayersByFantasyTeamAndPositionResponse{
		Rosters: protoRosters,
	}), nil
}

// GetPlayerOnRoster checks if a specific player is on a team's roster
func (s *Service) GetPlayerOnRoster(ctx context.Context, req *connect.Request[rosterv1.GetPlayerOnRosterRequest]) (*connect.Response[rosterv1.GetPlayerOnRosterResponse], error) {
	fantasyTeamID, err := uuid.Parse(req.Msg.FantasyTeamId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	playerID, err := uuid.Parse(req.Msg.PlayerId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	roster, err := s.app.GetPlayerOnRoster(ctx, fantasyTeamID, playerID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	protoRoster, err := s.rosterToProto(roster)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&rosterv1.GetPlayerOnRosterResponse{
		Roster: protoRoster,
	}), nil
}

// GetStartingRosterPlayers retrieves all starting players for a team
func (s *Service) GetStartingRosterPlayers(ctx context.Context, req *connect.Request[rosterv1.GetStartingRosterPlayersRequest]) (*connect.Response[rosterv1.GetStartingRosterPlayersResponse], error) {
	fantasyTeamID, err := uuid.Parse(req.Msg.FantasyTeamId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	rosters, err := s.app.GetStartingRosterPlayers(ctx, fantasyTeamID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	protoRosters, err := s.rostersToProto(rosters)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&rosterv1.GetStartingRosterPlayersResponse{
		Rosters: protoRosters,
	}), nil
}

// GetBenchRosterPlayers retrieves all bench players for a team
func (s *Service) GetBenchRosterPlayers(ctx context.Context, req *connect.Request[rosterv1.GetBenchRosterPlayersRequest]) (*connect.Response[rosterv1.GetBenchRosterPlayersResponse], error) {
	fantasyTeamID, err := uuid.Parse(req.Msg.FantasyTeamId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	rosters, err := s.app.GetBenchRosterPlayers(ctx, fantasyTeamID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	protoRosters, err := s.rostersToProto(rosters)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&rosterv1.GetBenchRosterPlayersResponse{
		Rosters: protoRosters,
	}), nil
}

// GetRosterPlayersByAcquisitionType retrieves players by how they were acquired
func (s *Service) GetRosterPlayersByAcquisitionType(ctx context.Context, req *connect.Request[rosterv1.GetRosterPlayersByAcquisitionTypeRequest]) (*connect.Response[rosterv1.GetRosterPlayersByAcquisitionTypeResponse], error) {
	fantasyTeamID, err := uuid.Parse(req.Msg.FantasyTeamId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	acquisitionType := s.protoToAcquisitionType(req.Msg.AcquisitionType)

	rosters, err := s.app.GetRosterPlayersByAcquisitionType(ctx, fantasyTeamID, acquisitionType)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	protoRosters, err := s.rostersToProto(rosters)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&rosterv1.GetRosterPlayersByAcquisitionTypeResponse{
		Rosters: protoRosters,
	}), nil
}

// UpdateRosterPlayerPosition updates a player's position on the roster
func (s *Service) UpdateRosterPlayerPosition(ctx context.Context, req *connect.Request[rosterv1.UpdateRosterPlayerPositionRequest]) (*connect.Response[rosterv1.UpdateRosterPlayerPositionResponse], error) {
	id, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	appReq := UpdateRosterPositionRequest{
		Position: s.protoToRosterPosition(req.Msg.Position),
	}

	roster, err := s.app.UpdateRosterPlayerPosition(ctx, id, appReq)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	protoRoster, err := s.rosterToProto(roster)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&rosterv1.UpdateRosterPlayerPositionResponse{
		Roster: protoRoster,
	}), nil
}

// UpdateRosterPlayerKeeperData updates a player's keeper data
func (s *Service) UpdateRosterPlayerKeeperData(ctx context.Context, req *connect.Request[rosterv1.UpdateRosterPlayerKeeperDataRequest]) (*connect.Response[rosterv1.UpdateRosterPlayerKeeperDataResponse], error) {
	id, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	var keeperData json.RawMessage
	if req.Msg.KeeperData != nil {
		keeperDataBytes, err := req.Msg.KeeperData.MarshalJSON()
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		keeperData = keeperDataBytes
	}

	appReq := UpdateRosterKeeperDataRequest{
		KeeperData: keeperData,
	}

	roster, err := s.app.UpdateRosterPlayerKeeperData(ctx, id, appReq)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	protoRoster, err := s.rosterToProto(roster)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&rosterv1.UpdateRosterPlayerKeeperDataResponse{
		Roster: protoRoster,
	}), nil
}

// UpdateRosterPositionAndKeeperData updates both position and keeper data
func (s *Service) UpdateRosterPositionAndKeeperData(ctx context.Context, req *connect.Request[rosterv1.UpdateRosterPositionAndKeeperDataRequest]) (*connect.Response[rosterv1.UpdateRosterPositionAndKeeperDataResponse], error) {
	id, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	var keeperData json.RawMessage
	if req.Msg.KeeperData != nil {
		keeperDataBytes, err := req.Msg.KeeperData.MarshalJSON()
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		keeperData = keeperDataBytes
	}

	appReq := UpdateRosterPositionAndKeeperDataRequest{
		Position:   s.protoToRosterPosition(req.Msg.Position),
		KeeperData: keeperData,
	}

	roster, err := s.app.UpdateRosterPositionAndKeeperData(ctx, id, appReq)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	protoRoster, err := s.rosterToProto(roster)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&rosterv1.UpdateRosterPositionAndKeeperDataResponse{
		Roster: protoRoster,
	}), nil
}

// DeleteRosterEntry removes a specific roster entry
func (s *Service) DeleteRosterEntry(ctx context.Context, req *connect.Request[rosterv1.DeleteRosterEntryRequest]) (*connect.Response[rosterv1.DeleteRosterEntryResponse], error) {
	id, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	err = s.app.DeleteRosterEntry(ctx, id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&rosterv1.DeleteRosterEntryResponse{
		Success: true,
	}), nil
}

// DeletePlayerFromRoster removes a player from a team's roster
func (s *Service) DeletePlayerFromRoster(ctx context.Context, req *connect.Request[rosterv1.DeletePlayerFromRosterRequest]) (*connect.Response[rosterv1.DeletePlayerFromRosterResponse], error) {
	fantasyTeamID, err := uuid.Parse(req.Msg.FantasyTeamId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	playerID, err := uuid.Parse(req.Msg.PlayerId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	err = s.app.DeletePlayerFromRoster(ctx, fantasyTeamID, playerID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&rosterv1.DeletePlayerFromRosterResponse{
		Success: true,
	}), nil
}

// DeleteTeamRoster clears an entire team's roster
func (s *Service) DeleteTeamRoster(ctx context.Context, req *connect.Request[rosterv1.DeleteTeamRosterRequest]) (*connect.Response[rosterv1.DeleteTeamRosterResponse], error) {
	fantasyTeamID, err := uuid.Parse(req.Msg.FantasyTeamId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	err = s.app.DeleteTeamRoster(ctx, fantasyTeamID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&rosterv1.DeleteTeamRosterResponse{
		Success: true,
	}), nil
}

// Conversion methods between proto and app layer models

func (s *Service) rosterToProto(roster *models.Roster) (*rosterv1.Roster, error) {
	var keeperDataStruct *structpb.Struct
	if len(roster.KeeperData) > 0 {
		var keeperDataMap map[string]interface{}
		if err := json.Unmarshal(roster.KeeperData, &keeperDataMap); err != nil {
			return nil, err
		}
		var err error
		keeperDataStruct, err = structpb.NewStruct(keeperDataMap)
		if err != nil {
			return nil, err
		}
	}

	return &rosterv1.Roster{
		Id:              roster.ID.String(),
		FantasyTeamId:   roster.FantasyTeamID.String(),
		PlayerId:        roster.PlayerID.String(),
		RosterPosition:  s.rosterPositionToProto(roster.Position),
		AcquisitionType: s.acquisitionTypeToProto(roster.AcquisitionType),
		CreatedAt:       timestamppb.New(roster.AcquiredAt),
		KeeperData:      keeperDataStruct,
	}, nil
}

func (s *Service) rostersToProto(rosters []models.Roster) ([]*rosterv1.Roster, error) {
	protoRosters := make([]*rosterv1.Roster, len(rosters))
	for i, roster := range rosters {
		protoRoster, err := s.rosterToProto(&roster)
		if err != nil {
			return nil, err
		}
		protoRosters[i] = protoRoster
	}
	return protoRosters, nil
}

func (s *Service) protoToCreateRosterRequest(proto *rosterv1.CreateRosterRequest) (CreateRosterRequest, error) {
	fantasyTeamID, err := uuid.Parse(proto.FantasyTeamId)
	if err != nil {
		return CreateRosterRequest{}, err
	}

	playerID, err := uuid.Parse(proto.PlayerId)
	if err != nil {
		return CreateRosterRequest{}, err
	}

	var keeperData json.RawMessage
	if proto.KeeperData != nil {
		keeperDataBytes, err := proto.KeeperData.MarshalJSON()
		if err != nil {
			return CreateRosterRequest{}, err
		}
		keeperData = keeperDataBytes
	}

	return CreateRosterRequest{
		FantasyTeamID:   fantasyTeamID,
		PlayerID:        playerID,
		Position:        s.protoToRosterPosition(proto.Position),
		AcquisitionType: s.protoToAcquisitionType(proto.AcquisitionType),
		KeeperData:      keeperData,
	}, nil
}

// Enum conversion methods

func (s *Service) rosterPositionToProto(position models.RosterPosition) rosterv1.RosterPosition {
	switch position {
	case models.RosterPositionStarter:
		return rosterv1.RosterPosition_ROSTER_POSITION_STARTING
	case models.RosterPositionBench:
		return rosterv1.RosterPosition_ROSTER_POSITION_BENCH
	case models.RosterPositionIR:
		return rosterv1.RosterPosition_ROSTER_POSITION_IR
	case models.RosterPositionTaxi:
		return rosterv1.RosterPosition_ROSTER_POSITION_TAXI
	default:
		return rosterv1.RosterPosition_ROSTER_POSITION_UNSPECIFIED
	}
}

func (s *Service) protoToRosterPosition(protoPosition rosterv1.RosterPosition) models.RosterPosition {
	switch protoPosition {
	case rosterv1.RosterPosition_ROSTER_POSITION_STARTING:
		return models.RosterPositionStarter
	case rosterv1.RosterPosition_ROSTER_POSITION_BENCH:
		return models.RosterPositionBench
	case rosterv1.RosterPosition_ROSTER_POSITION_IR:
		return models.RosterPositionIR
	case rosterv1.RosterPosition_ROSTER_POSITION_TAXI:
		return models.RosterPositionTaxi
	default:
		return models.RosterPositionBench // default fallback
	}
}

func (s *Service) acquisitionTypeToProto(acquisitionType models.AcquisitionType) rosterv1.AcquisitionType {
	switch acquisitionType {
	case models.AcquisitionTypeDraft:
		return rosterv1.AcquisitionType_ACQUISITION_TYPE_DRAFT
	case models.AcquisitionTypeWaiver:
		return rosterv1.AcquisitionType_ACQUISITION_TYPE_WAIVER
	case models.AcquisitionTypeFreeAgent:
		return rosterv1.AcquisitionType_ACQUISITION_TYPE_FREE_AGENT
	case models.AcquisitionTypeTrade:
		return rosterv1.AcquisitionType_ACQUISITION_TYPE_TRADE
	case models.AcquisitionTypeKeeper:
		return rosterv1.AcquisitionType_ACQUISITION_TYPE_KEEPER
	default:
		return rosterv1.AcquisitionType_ACQUISITION_TYPE_UNSPECIFIED
	}
}

func (s *Service) protoToAcquisitionType(protoType rosterv1.AcquisitionType) models.AcquisitionType {
	switch protoType {
	case rosterv1.AcquisitionType_ACQUISITION_TYPE_DRAFT:
		return models.AcquisitionTypeDraft
	case rosterv1.AcquisitionType_ACQUISITION_TYPE_WAIVER:
		return models.AcquisitionTypeWaiver
	case rosterv1.AcquisitionType_ACQUISITION_TYPE_FREE_AGENT:
		return models.AcquisitionTypeFreeAgent
	case rosterv1.AcquisitionType_ACQUISITION_TYPE_TRADE:
		return models.AcquisitionTypeTrade
	case rosterv1.AcquisitionType_ACQUISITION_TYPE_KEEPER:
		return models.AcquisitionTypeKeeper
	default:
		return models.AcquisitionTypeFreeAgent // default fallback
	}
}
