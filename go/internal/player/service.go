package player

import (
	"context"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	playerv1 "github.com/mcdev12/dynasty/go/internal/genproto/player/v1"
	"github.com/mcdev12/dynasty/go/internal/genproto/player/v1/playerv1connect"
	teamv1 "github.com/mcdev12/dynasty/go/internal/genproto/team/v1"
	"github.com/mcdev12/dynasty/go/internal/genproto/team/v1/teamv1connect"
	"github.com/mcdev12/dynasty/go/internal/models"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// PlayerApp defines what the service layer needs from the player application
type PlayerApp interface {
	CreatePlayer(ctx context.Context, player *models.Player) (*models.Player, error)
	GetPlayer(ctx context.Context, id uuid.UUID) (*models.Player, error)
	GetPlayerByExternalID(ctx context.Context, sportID, externalID string) (*models.Player, error)
	DeletePlayer(ctx context.Context, id uuid.UUID) error
	SyncPlayersFromAPI(ctx context.Context, teamID uuid.UUID, teamCode string, sportID string) (*SyncResult, error)
	SyncAllNFLPlayersFromAPI(ctx context.Context) (*SyncResult, error)
}

// Service implements the PlayerService gRPC interface
type Service struct {
	app         PlayerApp
	teamService teamv1connect.TeamServiceClient
}

// NewService creates a new player gRPC service
func NewService(app PlayerApp, teamService teamv1connect.TeamServiceClient) *Service {
	return &Service{
		app:         app,
		teamService: teamService,
	}
}

// Verify that Service implements the PlayerServiceHandler interface
var _ playerv1connect.PlayerServiceHandler = (*Service)(nil)

// CreatePlayer creates a new player
func (s *Service) CreatePlayer(ctx context.Context, req *connect.Request[playerv1.CreatePlayerRequest]) (*connect.Response[playerv1.CreatePlayerResponse], error) {
	// Simple proto to models conversion
	player := &models.Player{
		SportID:    req.Msg.SportId,
		ExternalID: req.Msg.ExternalId,
		FullName:   req.Msg.FullName,
	}

	// Parse optional team ID
	if req.Msg.TeamId != "" {
		if teamID, err := uuid.Parse(req.Msg.TeamId); err == nil {
			player.TeamID = &teamID
		}
	}

	// Handle profile if provided
	if req.Msg.GetPlayerProfile() != nil {
		if nflProfile := req.Msg.GetPlayerProfile().GetNflProfile(); nflProfile != nil {
			player.NFLPlayerProfile = s.protoToNFLProfile(nflProfile)
		}
	}

	createdPlayer, err := s.app.CreatePlayer(ctx, player)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	protoPlayer := s.playerToProto(createdPlayer)

	return connect.NewResponse(&playerv1.CreatePlayerResponse{
		Player: protoPlayer,
	}), nil
}

// GetPlayer retrieves a player by ID
func (s *Service) GetPlayer(ctx context.Context, req *connect.Request[playerv1.GetPlayerRequest]) (*connect.Response[playerv1.GetPlayerResponse], error) {
	id, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	player, err := s.app.GetPlayer(ctx, id)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	protoPlayer := s.playerToProto(player)

	return connect.NewResponse(&playerv1.GetPlayerResponse{
		Player: protoPlayer,
	}), nil
}

// GetPlayerByExternalID retrieves a player by sport ID and external ID
func (s *Service) GetPlayerByExternalID(ctx context.Context, req *connect.Request[playerv1.GetPlayerByExternalIDRequest]) (*connect.Response[playerv1.GetPlayerByExternalIDResponse], error) {
	player, err := s.app.GetPlayerByExternalID(ctx, req.Msg.SportId, req.Msg.ExternalId)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	protoPlayer := s.playerToProto(player)

	return connect.NewResponse(&playerv1.GetPlayerByExternalIDResponse{
		Player: protoPlayer,
	}), nil
}

// UpdatePlayer updates an existing player
func (s *Service) UpdatePlayer(ctx context.Context, req *connect.Request[playerv1.UpdatePlayerRequest]) (*connect.Response[playerv1.UpdatePlayerResponse], error) {
	// TODO: Implement when UpdatePlayer is added to app interface
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}

// DeletePlayer deletes a player by ID
func (s *Service) DeletePlayer(ctx context.Context, req *connect.Request[playerv1.DeletePlayerRequest]) (*connect.Response[playerv1.DeletePlayerResponse], error) {
	id, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	err = s.app.DeletePlayer(ctx, id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&playerv1.DeletePlayerResponse{
		Success: true,
	}), nil
}

// SyncPlayersFromAPI synchronizes players from external sports API for a specific team
func (s *Service) SyncPlayersFromAPI(ctx context.Context, req *connect.Request[playerv1.SyncPlayersFromAPIRequest]) (*connect.Response[playerv1.SyncPlayersFromAPIResponse], error) {
	teamResp, err := s.teamService.GetTeamBySportIDAndCode(ctx, connect.NewRequest(&teamv1.GetTeamBySportIDAndCodeRequest{
		SportId:  req.Msg.SportId,
		TeamCode: req.Msg.TeamAlias,
	}))
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	// Extract only the fields we need from the team proto
	teamID, _ := uuid.Parse(teamResp.Msg.Team.Id)
	teamCode := teamResp.Msg.Team.Code

	// Call app layer with only the data we need
	result, err := s.app.SyncPlayersFromAPI(ctx, teamID, teamCode, req.Msg.SportId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Convert result to proto
	protoResult := s.syncResultToProto(result)

	return connect.NewResponse(&playerv1.SyncPlayersFromAPIResponse{
		Result: protoResult,
	}), nil
}

// SyncAllNFLPlayersFromAPI synchronizes all NFL players from external sports API
func (s *Service) SyncAllNFLPlayersFromAPI(ctx context.Context, req *connect.Request[playerv1.SyncAllNFLPlayersFromAPIRequest]) (*connect.Response[playerv1.SyncAllNFLPlayersFromAPIResponse], error) {
	result, err := s.app.SyncAllNFLPlayersFromAPI(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	protoResult := s.syncResultToProto(result)

	return connect.NewResponse(&playerv1.SyncAllNFLPlayersFromAPIResponse{
		Result: protoResult,
	}), nil
}

// GetPlayersWithFilter retrieves players with filtering and pagination
func (s *Service) GetPlayersWithFilter(ctx context.Context, req *connect.Request[playerv1.GetPlayersWithFilterRequest]) (*connect.Response[playerv1.GetPlayersWithFilterResponse], error) {
	// TODO: Implement when filtering is added to app layer
	return nil, connect.NewError(connect.CodeUnimplemented, nil)
}

// Conversion methods between proto and app layer models

func (s *Service) playerToProto(player *models.Player) *playerv1.Player {
	proto := &playerv1.Player{
		Id:         player.ID.String(),
		SportId:    player.SportID,
		ExternalId: player.ExternalID,
		FullName:   player.FullName,
		CreatedAt:  timestamppb.New(player.CreatedAt),
	}

	if player.TeamID != nil {
		proto.TeamId = player.TeamID.String()
	}

	// Handle sport-specific profiles
	if player.NFLPlayerProfile != nil {
		proto.Profile = &playerv1.Player_PlayerProfile{
			PlayerProfile: &playerv1.PlayerProfile{
				Profile: &playerv1.PlayerProfile_NflProfile{
					NflProfile: s.nflProfileToProto(player.NFLPlayerProfile),
				},
			},
		}
	}

	return proto
}

func (s *Service) nflProfileToProto(profile *models.NFLPlayerProfile) *playerv1.NFLPlayerProfile {
	proto := &playerv1.NFLPlayerProfile{
		PlayerId:     profile.PlayerID.String(),
		Position:     profile.Position,
		Status:       profile.Status,
		College:      profile.College,
		JerseyNumber: int32(profile.JerseyNumber),
		Experience:   int32(profile.Experience),
		HeightCm:     int32(profile.HeightCm),
		WeightKg:     int32(profile.WeightKg),
		HeightDesc:   profile.HeightDesc,
		WeightDesc:   profile.WeightDesc,
	}

	if profile.BirthDate != nil {
		proto.BirthDate = profile.BirthDate.Format("2006-01-02")
	}

	return proto
}

func (s *Service) protoToNFLProfile(proto *playerv1.NFLPlayerProfile) *models.NFLPlayerProfile {
	playerID, _ := uuid.Parse(proto.PlayerId)

	profile := &models.NFLPlayerProfile{
		PlayerID:     playerID,
		Position:     proto.Position,
		Status:       proto.Status,
		College:      proto.College,
		JerseyNumber: int(proto.JerseyNumber),
		Experience:   int(proto.Experience),
		HeightCm:     int(proto.HeightCm),
		WeightKg:     int(proto.WeightKg),
		HeightDesc:   proto.HeightDesc,
		WeightDesc:   proto.WeightDesc,
	}

	if proto.BirthDate != "" {
		if birthDate, err := time.Parse("2006-01-02", proto.BirthDate); err == nil {
			profile.BirthDate = &birthDate
		}
	}

	return profile
}

func (s *Service) syncResultToProto(result *SyncResult) *playerv1.SyncResult {
	errors := make([]string, len(result.Errors))
	for i, err := range result.Errors {
		errors[i] = err.Error()
	}

	return &playerv1.SyncResult{
		TotalProcessed: int32(result.TotalProcessed),
		Created:        int32(result.Created),
		Updated:        int32(result.Updated),
		Errors:         errors,
	}
}
