package fantasyteam

import (
	"context"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	fantasyteamv1 "github.com/mcdev12/dynasty/go/internal/genproto/fantasyteam/v1"
	"github.com/mcdev12/dynasty/go/internal/genproto/fantasyteam/v1/fantasyteamv1connect"
	"github.com/mcdev12/dynasty/go/internal/models"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// FantasyTeamApp defines what the service layer needs from the fantasy teams application
type FantasyTeamApp interface {
	CreateFantasyTeam(ctx context.Context, req CreateFantasyTeamRequest) (*models.FantasyTeam, error)
	GetFantasyTeam(ctx context.Context, id uuid.UUID) (*models.FantasyTeam, error)
	GetFantasyTeamsByLeague(ctx context.Context, leagueID uuid.UUID) ([]models.FantasyTeam, error)
	GetFantasyTeamsByOwner(ctx context.Context, ownerID uuid.UUID) ([]models.FantasyTeam, error)
	GetFantasyTeamByLeagueAndOwner(ctx context.Context, ownerID, leagueID uuid.UUID) (*models.FantasyTeam, error)
	UpdateFantasyTeam(ctx context.Context, id uuid.UUID, req UpdateFantasyTeamRequest) (*models.FantasyTeam, error)
	DeleteFantasyTeam(ctx context.Context, id uuid.UUID) error
}

// Service implements the FantasyTeamService gRPC interface
type Service struct {
	app FantasyTeamApp
}

// NewService creates a new fantasy teams gRPC service
func NewService(app FantasyTeamApp) *Service {
	return &Service{
		app: app,
	}
}

// Verify that Service implements the FantasyTeamServiceHandler interface
var _ fantasyteamv1connect.FantasyTeamServiceHandler = (*Service)(nil)

// CreateFantasyTeam creates a new fantasy team
func (s *Service) CreateFantasyTeam(ctx context.Context, req *connect.Request[fantasyteamv1.CreateFantasyTeamRequest]) (*connect.Response[fantasyteamv1.CreateFantasyTeamResponse], error) {
	appReq, err := s.protoToCreateFantasyTeamRequest(req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	team, err := s.app.CreateFantasyTeam(ctx, appReq)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	protoTeam := s.fantasyTeamToProto(team)

	return connect.NewResponse(&fantasyteamv1.CreateFantasyTeamResponse{
		FantasyTeam: protoTeam,
	}), nil
}

// GetFantasyTeam retrieves a fantasy team by ID
func (s *Service) GetFantasyTeam(ctx context.Context, req *connect.Request[fantasyteamv1.GetFantasyTeamRequest]) (*connect.Response[fantasyteamv1.GetFantasyTeamResponse], error) {
	id, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	team, err := s.app.GetFantasyTeam(ctx, id)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	protoTeam := s.fantasyTeamToProto(team)

	return connect.NewResponse(&fantasyteamv1.GetFantasyTeamResponse{
		FantasyTeam: protoTeam,
	}), nil
}

// GetFantasyTeamsByLeague retrieves fantasy teams by league ID
func (s *Service) GetFantasyTeamsByLeague(ctx context.Context, req *connect.Request[fantasyteamv1.GetFantasyTeamsByLeagueRequest]) (*connect.Response[fantasyteamv1.GetFantasyTeamsByLeagueResponse], error) {
	leagueID, err := uuid.Parse(req.Msg.LeagueId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	teams, err := s.app.GetFantasyTeamsByLeague(ctx, leagueID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	protoTeams := s.fantasyTeamsToProto(teams)

	return connect.NewResponse(&fantasyteamv1.GetFantasyTeamsByLeagueResponse{
		FantasyTeams: protoTeams,
	}), nil
}

// GetFantasyTeamsByOwner retrieves fantasy teams by owner ID
func (s *Service) GetFantasyTeamsByOwner(ctx context.Context, req *connect.Request[fantasyteamv1.GetFantasyTeamsByOwnerRequest]) (*connect.Response[fantasyteamv1.GetFantasyTeamsByOwnerResponse], error) {
	ownerID, err := uuid.Parse(req.Msg.OwnerId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	teams, err := s.app.GetFantasyTeamsByOwner(ctx, ownerID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	protoTeams := s.fantasyTeamsToProto(teams)

	return connect.NewResponse(&fantasyteamv1.GetFantasyTeamsByOwnerResponse{
		FantasyTeams: protoTeams,
	}), nil
}

// GetFantasyTeamByLeagueAndOwner retrieves a fantasy team by league and owner
func (s *Service) GetFantasyTeamByLeagueAndOwner(ctx context.Context, req *connect.Request[fantasyteamv1.GetFantasyTeamByLeagueAndOwnerRequest]) (*connect.Response[fantasyteamv1.GetFantasyTeamByLeagueAndOwnerResponse], error) {
	ownerID, err := uuid.Parse(req.Msg.OwnerId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	leagueID, err := uuid.Parse(req.Msg.LeagueId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	team, err := s.app.GetFantasyTeamByLeagueAndOwner(ctx, ownerID, leagueID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	protoTeam := s.fantasyTeamToProto(team)

	return connect.NewResponse(&fantasyteamv1.GetFantasyTeamByLeagueAndOwnerResponse{
		FantasyTeam: protoTeam,
	}), nil
}

// UpdateFantasyTeam updates an existing fantasy team
func (s *Service) UpdateFantasyTeam(ctx context.Context, req *connect.Request[fantasyteamv1.UpdateFantasyTeamRequest]) (*connect.Response[fantasyteamv1.UpdateFantasyTeamResponse], error) {
	id, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	appReq := s.protoToUpdateFantasyTeamRequest(req.Msg)

	team, err := s.app.UpdateFantasyTeam(ctx, id, appReq)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	protoTeam := s.fantasyTeamToProto(team)

	return connect.NewResponse(&fantasyteamv1.UpdateFantasyTeamResponse{
		FantasyTeam: protoTeam,
	}), nil
}

// DeleteFantasyTeam deletes a fantasy team by ID
func (s *Service) DeleteFantasyTeam(ctx context.Context, req *connect.Request[fantasyteamv1.DeleteFantasyTeamRequest]) (*connect.Response[fantasyteamv1.DeleteFantasyTeamResponse], error) {
	id, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	err = s.app.DeleteFantasyTeam(ctx, id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&fantasyteamv1.DeleteFantasyTeamResponse{
		Success: true,
	}), nil
}

// Conversion methods between proto and app layer models

func (s *Service) fantasyTeamToProto(team *models.FantasyTeam) *fantasyteamv1.FantasyTeam {
	return &fantasyteamv1.FantasyTeam{
		Id:        team.ID.String(),
		LeagueId:  team.LeagueID.String(),
		OwnerId:   team.OwnerID.String(),
		Name:      team.Name,
		LogoUrl:   team.LogoURL,
		CreatedAt: timestamppb.New(team.CreatedAt),
	}
}

func (s *Service) fantasyTeamsToProto(teams []models.FantasyTeam) []*fantasyteamv1.FantasyTeam {
	protoTeams := make([]*fantasyteamv1.FantasyTeam, len(teams))
	for i, team := range teams {
		protoTeams[i] = s.fantasyTeamToProto(&team)
	}
	return protoTeams
}

func (s *Service) protoToCreateFantasyTeamRequest(proto *fantasyteamv1.CreateFantasyTeamRequest) (CreateFantasyTeamRequest, error) {
	leagueID, err := uuid.Parse(proto.LeagueId)
	if err != nil {
		return CreateFantasyTeamRequest{}, err
	}

	ownerID, err := uuid.Parse(proto.OwnerId)
	if err != nil {
		return CreateFantasyTeamRequest{}, err
	}

	return CreateFantasyTeamRequest{
		LeagueID: leagueID,
		OwnerID:  ownerID,
		Name:     proto.Name,
		LogoURL:  proto.LogoUrl,
	}, nil
}

func (s *Service) protoToUpdateFantasyTeamRequest(proto *fantasyteamv1.UpdateFantasyTeamRequest) UpdateFantasyTeamRequest {
	return UpdateFantasyTeamRequest{
		Name:    proto.Name,
		LogoURL: proto.LogoUrl,
	}
}
