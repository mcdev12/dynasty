package teams

import (
	"context"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	teamv1 "github.com/mcdev12/dynasty/go/internal/genproto/team/v1"
	"github.com/mcdev12/dynasty/go/internal/genproto/team/v1/teamv1connect"
	"github.com/mcdev12/dynasty/go/internal/models"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// TeamsApp defines what the service layer needs from the teams application
type TeamsApp interface {
	CreateTeam(ctx context.Context, req CreateTeamRequest) (*models.Team, error)
	GetTeam(ctx context.Context, id uuid.UUID) (*models.Team, error)
	GetTeamByExternalID(ctx context.Context, sportID, externalID string) (*models.Team, error)
	GetTeamBySportIdAndCode(ctx context.Context, sportID, code string) (*models.Team, error)
	ListTeamsBySport(ctx context.Context, sportID string) ([]models.Team, error)
	ListAllTeams(ctx context.Context) ([]models.Team, error)
	UpdateTeam(ctx context.Context, id uuid.UUID, req UpdateTeamRequest) (*models.Team, error)
	DeleteTeam(ctx context.Context, id uuid.UUID) error
	SyncTeamsFromAPI(ctx context.Context, sportID string) (*SyncResult, error)
	GetTeamsWithFilter(ctx context.Context, filter TeamFilter, pagination PaginationParams) (*TeamListResponse, error)
}

// Service implements the TeamService gRPC interface
type Service struct {
	app TeamsApp
}

// NewService creates a new teams gRPC service
func NewService(app TeamsApp) *Service {
	return &Service{
		app: app,
	}
}

// Verify that Service implements the TeamServiceHandler interface
var _ teamv1connect.TeamServiceHandler = (*Service)(nil)

// CreateTeam creates a new team
func (s *Service) CreateTeam(ctx context.Context, req *connect.Request[teamv1.CreateTeamRequest]) (*connect.Response[teamv1.CreateTeamResponse], error) {
	appReq := s.protoToCreateTeamRequest(req.Msg)

	team, err := s.app.CreateTeam(ctx, appReq)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	protoTeam := s.teamToProto(team)

	return connect.NewResponse(&teamv1.CreateTeamResponse{
		Team: protoTeam,
	}), nil
}

// GetTeam retrieves a team by ID
func (s *Service) GetTeam(ctx context.Context, req *connect.Request[teamv1.GetTeamRequest]) (*connect.Response[teamv1.GetTeamResponse], error) {
	id, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	team, err := s.app.GetTeam(ctx, id)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	protoTeam := s.teamToProto(team)

	return connect.NewResponse(&teamv1.GetTeamResponse{
		Team: protoTeam,
	}), nil
}

// GetTeamByExternalID retrieves a team by sport ID and external ID
func (s *Service) GetTeamByExternalID(ctx context.Context, req *connect.Request[teamv1.GetTeamByExternalIDRequest]) (*connect.Response[teamv1.GetTeamByExternalIDResponse], error) {
	team, err := s.app.GetTeamByExternalID(ctx, req.Msg.SportId, req.Msg.ExternalId)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	protoTeam := s.teamToProto(team)

	return connect.NewResponse(&teamv1.GetTeamByExternalIDResponse{
		Team: protoTeam,
	}), nil
}

func (s *Service) GetTeamBySportIDAndCode(ctx context.Context, req *connect.Request[teamv1.GetTeamBySportIDAndCodeRequest]) (*connect.Response[teamv1.GetTeamBySportIDAndCodeResponse], error) {
	team, err := s.app.GetTeamBySportIdAndCode(ctx, req.Msg.SportId, req.Msg.TeamCode)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	protoTeam := s.teamToProto(team)

	return connect.NewResponse(&teamv1.GetTeamBySportIDAndCodeResponse{
		Team: protoTeam,
	}), nil
}

// ListTeamsBySport retrieves all teams for a specific sport
func (s *Service) ListTeamsBySport(ctx context.Context, req *connect.Request[teamv1.ListTeamsBySportRequest]) (*connect.Response[teamv1.ListTeamsBySportResponse], error) {
	teams, err := s.app.ListTeamsBySport(ctx, req.Msg.SportId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	protoTeams := make([]*teamv1.Team, len(teams))
	for i, team := range teams {
		protoTeams[i] = s.teamToProto(&team)
	}

	return connect.NewResponse(&teamv1.ListTeamsBySportResponse{
		Teams: protoTeams,
	}), nil
}

// ListAllTeams retrieves all teams with optional filtering and pagination
func (s *Service) ListAllTeams(ctx context.Context, req *connect.Request[teamv1.ListAllTeamsRequest]) (*connect.Response[teamv1.ListAllTeamsResponse], error) {
	teams, err := s.app.ListAllTeams(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Apply pagination if provided
	total := len(teams)
	if req.Msg.Pagination != nil {
		offset := int(req.Msg.Pagination.Offset)
		limit := int(req.Msg.Pagination.Limit)

		if offset >= len(teams) {
			teams = []models.Team{}
		} else {
			end := offset + limit
			if end > len(teams) {
				end = len(teams)
			}
			teams = teams[offset:end]
		}
	}

	protoTeams := make([]*teamv1.Team, len(teams))
	for i, team := range teams {
		protoTeams[i] = s.teamToProto(&team)
	}

	hasMore := false
	if req.Msg.Pagination != nil {
		hasMore = int(req.Msg.Pagination.Offset)+len(teams) < total
	}

	return connect.NewResponse(&teamv1.ListAllTeamsResponse{
		Teams:   protoTeams,
		Total:   int32(total),
		HasMore: hasMore,
	}), nil
}

// UpdateTeam updates an existing team
func (s *Service) UpdateTeam(ctx context.Context, req *connect.Request[teamv1.UpdateTeamRequest]) (*connect.Response[teamv1.UpdateTeamResponse], error) {
	id, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	appReq := s.protoToUpdateTeamRequest(req.Msg)

	team, err := s.app.UpdateTeam(ctx, id, appReq)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	protoTeam := s.teamToProto(team)

	return connect.NewResponse(&teamv1.UpdateTeamResponse{
		Team: protoTeam,
	}), nil
}

// DeleteTeam deletes a team by ID
func (s *Service) DeleteTeam(ctx context.Context, req *connect.Request[teamv1.DeleteTeamRequest]) (*connect.Response[teamv1.DeleteTeamResponse], error) {
	id, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	err = s.app.DeleteTeam(ctx, id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&teamv1.DeleteTeamResponse{
		Success: true,
	}), nil
}

// SyncTeamsFromAPI synchronizes teams from external sports API
func (s *Service) SyncTeamsFromAPI(ctx context.Context, req *connect.Request[teamv1.SyncTeamsFromAPIRequest]) (*connect.Response[teamv1.SyncTeamsFromAPIResponse], error) {
	result, err := s.app.SyncTeamsFromAPI(ctx, req.Msg.SportId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	protoResult := s.syncResultToProto(result)

	return connect.NewResponse(&teamv1.SyncTeamsFromAPIResponse{
		Result: protoResult,
	}), nil
}

// GetTeamsWithFilter retrieves teams with filtering and pagination
func (s *Service) GetTeamsWithFilter(ctx context.Context, req *connect.Request[teamv1.GetTeamsWithFilterRequest]) (*connect.Response[teamv1.GetTeamsWithFilterResponse], error) {
	filter := TeamFilter{}
	pagination := PaginationParams{Limit: 50, Offset: 0} // defaults

	if req.Msg.Filter != nil {
		filter = s.protoToTeamFilter(req.Msg.Filter)
	}

	if req.Msg.Pagination != nil {
		pagination = s.protoToPaginationParams(req.Msg.Pagination)
	}

	response, err := s.app.GetTeamsWithFilter(ctx, filter, pagination)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	protoResponse := s.teamListResponseToProto(response)

	return connect.NewResponse(&teamv1.GetTeamsWithFilterResponse{
		Response: protoResponse,
	}), nil
}

// Conversion methods between proto and app layer models

func (s *Service) teamToProto(team *models.Team) *teamv1.Team {
	proto := &teamv1.Team{
		Id:         team.ID.String(),
		SportId:    team.SportID,
		ExternalId: team.ExternalID,
		Name:       team.Name,
		Code:       team.Code,
		City:       team.City,
		CreatedAt:  timestamppb.New(team.CreatedAt),
	}

	if team.Coach != nil {
		proto.Coach = team.Coach
	}
	if team.Owner != nil {
		proto.Owner = team.Owner
	}
	if team.Stadium != nil {
		proto.Stadium = team.Stadium
	}
	if team.EstablishedYear != nil {
		year := int32(*team.EstablishedYear)
		proto.EstablishedYear = &year
	}

	return proto
}

func (s *Service) protoToCreateTeamRequest(proto *teamv1.CreateTeamRequest) CreateTeamRequest {
	req := CreateTeamRequest{
		SportID:    proto.SportId,
		ExternalID: proto.ExternalId,
		Name:       proto.Name,
		Code:       proto.Code,
		City:       proto.City,
	}

	if proto.Coach != nil {
		req.Coach = proto.Coach
	}
	if proto.Owner != nil {
		req.Owner = proto.Owner
	}
	if proto.Stadium != nil {
		req.Stadium = proto.Stadium
	}
	if proto.EstablishedYear != nil {
		year := int(*proto.EstablishedYear)
		req.EstablishedYear = &year
	}

	return req
}

func (s *Service) protoToUpdateTeamRequest(proto *teamv1.UpdateTeamRequest) UpdateTeamRequest {
	req := UpdateTeamRequest{}

	if proto.Name != nil {
		req.Name = proto.Name
	}
	if proto.Code != nil {
		req.Code = proto.Code
	}
	if proto.City != nil {
		req.City = proto.City
	}
	if proto.Coach != nil {
		req.Coach = proto.Coach
	}
	if proto.Owner != nil {
		req.Owner = proto.Owner
	}
	if proto.Stadium != nil {
		req.Stadium = proto.Stadium
	}
	if proto.EstablishedYear != nil {
		year := int(*proto.EstablishedYear)
		req.EstablishedYear = &year
	}

	return req
}

func (s *Service) protoToTeamFilter(proto *teamv1.TeamFilter) TeamFilter {
	filter := TeamFilter{}

	if proto.SportId != nil {
		filter.SportID = proto.SportId
	}
	if proto.City != nil {
		filter.City = proto.City
	}
	if proto.Code != nil {
		filter.Code = proto.Code
	}

	return filter
}

func (s *Service) protoToPaginationParams(proto *teamv1.PaginationParams) PaginationParams {
	return PaginationParams{
		Limit:  int(proto.Limit),
		Offset: int(proto.Offset),
	}
}

func (s *Service) syncResultToProto(result *SyncResult) *teamv1.SyncResult {
	errors := make([]string, len(result.Errors))
	for i, err := range result.Errors {
		errors[i] = err.Error()
	}

	return &teamv1.SyncResult{
		TotalProcessed: int32(result.TotalProcessed),
		Created:        int32(result.Created),
		Updated:        int32(result.Updated),
		Errors:         errors,
	}
}

func (s *Service) teamListResponseToProto(response *TeamListResponse) *teamv1.TeamListResponse {
	protoTeams := make([]*teamv1.Team, len(response.Teams))
	for i, team := range response.Teams {
		protoTeams[i] = s.teamToProto(&team)
	}

	return &teamv1.TeamListResponse{
		Teams:   protoTeams,
		Total:   int32(response.Total),
		Limit:   int32(response.Limit),
		Offset:  int32(response.Offset),
		HasMore: response.HasMore,
	}
}
