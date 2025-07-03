package leagues

import (
	"context"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	leaguev1 "github.com/mcdev12/dynasty/go/internal/genproto/league/v1"
	"github.com/mcdev12/dynasty/go/internal/genproto/league/v1/leaguev1connect"
	userv1 "github.com/mcdev12/dynasty/go/internal/genproto/user/v1"
	"github.com/mcdev12/dynasty/go/internal/genproto/user/v1/userv1connect"
	"github.com/mcdev12/dynasty/go/internal/models"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// LeaguesApp defines what the service layer needs from the leagues application
type LeaguesApp interface {
	CreateLeague(ctx context.Context, req CreateLeagueRequest) (*models.League, error)
	GetLeague(ctx context.Context, id uuid.UUID) (*models.League, error)
	GetLeaguesByCommissioner(ctx context.Context, commissionerID uuid.UUID) ([]models.League, error)
	UpdateLeague(ctx context.Context, id uuid.UUID, req UpdateLeagueRequest) (*models.League, error)
	UpdateLeagueStatus(ctx context.Context, id uuid.UUID, status models.LeagueStatus) (*models.League, error)
	UpdateLeagueSettings(ctx context.Context, id uuid.UUID, settings interface{}) (*models.League, error)
	DeleteLeague(ctx context.Context, id uuid.UUID) error
}

// Service implements the LeagueService gRPC interface
type Service struct {
	app         LeaguesApp
	userService userv1connect.UserServiceClient
}

// NewService creates a new leagues gRPC service
func NewService(app LeaguesApp, userService userv1connect.UserServiceClient) *Service {
	return &Service{
		app:         app,
		userService: userService,
	}
}

// Verify that Service implements the LeagueServiceHandler interface
var _ leaguev1connect.LeagueServiceHandler = (*Service)(nil)

// CreateLeague creates a new league
func (s *Service) CreateLeague(ctx context.Context, req *connect.Request[leaguev1.CreateLeagueRequest]) (*connect.Response[leaguev1.CreateLeagueResponse], error) {
	appReq, err := s.protoToCreateLeagueRequest(req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	// Cross-domain orchestration: validate commissioner exists first
	_, err = s.userService.GetUser(ctx, connect.NewRequest(&userv1.GetUserRequest{
		Id: appReq.CommissionerID.String(),
	}))
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	league, err := s.app.CreateLeague(ctx, appReq)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	protoLeague, err := s.leagueToProto(league)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&leaguev1.CreateLeagueResponse{
		League: protoLeague,
	}), nil
}

// GetLeague retrieves a league by ID
func (s *Service) GetLeague(ctx context.Context, req *connect.Request[leaguev1.GetLeagueRequest]) (*connect.Response[leaguev1.GetLeagueResponse], error) {
	id, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	league, err := s.app.GetLeague(ctx, id)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	protoLeague, err := s.leagueToProto(league)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&leaguev1.GetLeagueResponse{
		League: protoLeague,
	}), nil
}

// GetLeaguesByCommissioner retrieves leagues by commissioner ID
func (s *Service) GetLeaguesByCommissioner(ctx context.Context, req *connect.Request[leaguev1.GetLeaguesByCommissionerRequest]) (*connect.Response[leaguev1.GetLeaguesByCommissionerResponse], error) {
	commissionerID, err := uuid.Parse(req.Msg.CommissionerId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	leagues, err := s.app.GetLeaguesByCommissioner(ctx, commissionerID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	protoLeagues, err := s.leaguesToProto(leagues)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&leaguev1.GetLeaguesByCommissionerResponse{
		Leagues: protoLeagues,
	}), nil
}

// UpdateLeague updates an existing league
func (s *Service) UpdateLeague(ctx context.Context, req *connect.Request[leaguev1.UpdateLeagueRequest]) (*connect.Response[leaguev1.UpdateLeagueResponse], error) {
	id, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	appReq, err := s.protoToUpdateLeagueRequest(req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	// Cross-domain orchestration: validate commissioner exists first
	_, err = s.userService.GetUser(ctx, connect.NewRequest(&userv1.GetUserRequest{
		Id: appReq.CommissionerID.String(),
	}))
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	league, err := s.app.UpdateLeague(ctx, id, appReq)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	protoLeague, err := s.leagueToProto(league)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&leaguev1.UpdateLeagueResponse{
		League: protoLeague,
	}), nil
}

// UpdateLeagueStatus updates only the status of a league
func (s *Service) UpdateLeagueStatus(ctx context.Context, req *connect.Request[leaguev1.UpdateLeagueStatusRequest]) (*connect.Response[leaguev1.UpdateLeagueStatusResponse], error) {
	id, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	league, err := s.app.UpdateLeagueStatus(ctx, id, s.protoToLeagueStatus(req.Msg.Status))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	protoLeague, err := s.leagueToProto(league)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&leaguev1.UpdateLeagueStatusResponse{
		League: protoLeague,
	}), nil
}

// UpdateLeagueSettings updates only the settings of a league
func (s *Service) UpdateLeagueSettings(ctx context.Context, req *connect.Request[leaguev1.UpdateLeagueSettingsRequest]) (*connect.Response[leaguev1.UpdateLeagueSettingsResponse], error) {
	id, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	// Convert protobuf Struct to interface{}
	settings := req.Msg.LeagueSettings.AsMap()

	league, err := s.app.UpdateLeagueSettings(ctx, id, settings)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	protoLeague, err := s.leagueToProto(league)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&leaguev1.UpdateLeagueSettingsResponse{
		League: protoLeague,
	}), nil
}

// DeleteLeague deletes a league by ID
func (s *Service) DeleteLeague(ctx context.Context, req *connect.Request[leaguev1.DeleteLeagueRequest]) (*connect.Response[leaguev1.DeleteLeagueResponse], error) {
	id, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	err = s.app.DeleteLeague(ctx, id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&leaguev1.DeleteLeagueResponse{
		Success: true,
	}), nil
}

// Conversion methods between proto and app layer models

func (s *Service) leagueToProto(league *models.League) (*leaguev1.League, error) {
	// Convert league settings to protobuf Struct
	settingsStruct, err := structpb.NewStruct(league.LeagueSettings.(map[string]interface{}))
	if err != nil {
		return nil, err
	}

	return &leaguev1.League{
		Id:             league.ID.String(),
		Name:           league.Name,
		SportId:        league.SportID,
		LeagueType:     s.leagueTypeToProto(league.LeagueType),
		CommissionerId: league.CommissionerID.String(),
		LeagueSettings: settingsStruct,
		Status:         s.leagueStatusToProto(league.Status),
		Season:         league.Season,
		CreatedAt:      timestamppb.New(league.CreatedAt),
		UpdatedAt:      timestamppb.New(league.UpdatedAt),
	}, nil
}

func (s *Service) leaguesToProto(leagues []models.League) ([]*leaguev1.League, error) {
	protoLeagues := make([]*leaguev1.League, len(leagues))
	for i, league := range leagues {
		protoLeague, err := s.leagueToProto(&league)
		if err != nil {
			return nil, err
		}
		protoLeagues[i] = protoLeague
	}
	return protoLeagues, nil
}

func (s *Service) protoToCreateLeagueRequest(proto *leaguev1.CreateLeagueRequest) (CreateLeagueRequest, error) {
	commissionerID, err := uuid.Parse(proto.CommissionerId)
	if err != nil {
		return CreateLeagueRequest{}, err
	}

	return CreateLeagueRequest{
		Name:           proto.Name,
		SportID:        proto.SportId,
		LeagueType:     s.protoToLeagueType(proto.LeagueType),
		CommissionerID: commissionerID,
		LeagueSettings: proto.LeagueSettings.AsMap(),
		Status:         s.protoToLeagueStatus(proto.LeagueStatus),
		Season:         proto.Season,
	}, nil
}

func (s *Service) protoToUpdateLeagueRequest(proto *leaguev1.UpdateLeagueRequest) (UpdateLeagueRequest, error) {
	commissionerID, err := uuid.Parse(proto.CommissionerId)
	if err != nil {
		return UpdateLeagueRequest{}, err
	}

	return UpdateLeagueRequest{
		Name:           proto.Name,
		SportID:        proto.SportId,
		LeagueType:     s.protoToLeagueType(proto.LeagueType),
		CommissionerID: commissionerID,
		LeagueSettings: proto.LeagueSettings.AsMap(),
		Status:         s.protoToLeagueStatus(proto.Status),
		Season:         proto.Season,
	}, nil
}

func (s *Service) leagueTypeToProto(leagueType models.LeagueType) leaguev1.LeagueType {
	switch leagueType {
	case models.LeagueTypeRedraft:
		return leaguev1.LeagueType_LEAGUE_TYPE_REDRAFT
	case models.LeagueTypeKeeper:
		return leaguev1.LeagueType_LEAGUE_TYPE_KEEPER
	case models.LeagueTypeDynasty:
		return leaguev1.LeagueType_LEAGUE_TYPE_DYNASTY
	default:
		return leaguev1.LeagueType_LEAGUE_TYPE_UNSPECIFIED
	}
}

func (s *Service) protoToLeagueType(protoType leaguev1.LeagueType) models.LeagueType {
	switch protoType {
	case leaguev1.LeagueType_LEAGUE_TYPE_REDRAFT:
		return models.LeagueTypeRedraft
	case leaguev1.LeagueType_LEAGUE_TYPE_KEEPER:
		return models.LeagueTypeKeeper
	case leaguev1.LeagueType_LEAGUE_TYPE_DYNASTY:
		return models.LeagueTypeDynasty
	default:
		return models.LeagueTypeRedraft // default fallback
	}
}

func (s *Service) leagueStatusToProto(leagueStatus models.LeagueStatus) leaguev1.LeagueStatus {
	switch leagueStatus {
	case models.LeagueStatusActive:
		return leaguev1.LeagueStatus_LEAGUE_STATUS_ACTIVE
	case models.LeagueStatusCancelled:
		return leaguev1.LeagueStatus_LEAGUE_STATUS_CANCELLED
	case models.LeagueStatusCompleted:
		return leaguev1.LeagueStatus_LEAGUE_STATUS_COMPLETED
	case models.LeagueStatusPending:
		return leaguev1.LeagueStatus_LEAGUE_STATUS_PENDING
	default:
		return leaguev1.LeagueStatus_LEAGUE_STATUS_UNSPECIFIED
	}
}

func (s *Service) protoToLeagueStatus(protoStatus leaguev1.LeagueStatus) models.LeagueStatus {
	switch protoStatus {
	case leaguev1.LeagueStatus_LEAGUE_STATUS_ACTIVE:
		return models.LeagueStatusActive
	case leaguev1.LeagueStatus_LEAGUE_STATUS_CANCELLED:
		return models.LeagueStatusCancelled
	case leaguev1.LeagueStatus_LEAGUE_STATUS_COMPLETED:
		return models.LeagueStatusCompleted
	case leaguev1.LeagueStatus_LEAGUE_STATUS_PENDING:
		return models.LeagueStatusPending
	default:
		return models.LeagueStatusActive
	}
}
