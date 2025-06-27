package users

import (
	"context"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	userv1 "github.com/mcdev12/dynasty/go/internal/genproto/user/v1"
	"github.com/mcdev12/dynasty/go/internal/genproto/user/v1/userv1connect"
	"github.com/mcdev12/dynasty/go/internal/models"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// UsersApp defines what the service layer needs from the users application
type UsersApp interface {
	CreateUser(ctx context.Context, req CreateUserRequest) (*models.User, error)
	GetUser(ctx context.Context, id uuid.UUID) (*models.User, error)
	GetUserByUsername(ctx context.Context, username string) (*models.User, error)
	GetUserByEmail(ctx context.Context, email string) (*models.User, error)
	UpdateUser(ctx context.Context, id uuid.UUID, req UpdateUserRequest) (*models.User, error)
	DeleteUser(ctx context.Context, id uuid.UUID) error
}

// Service implements the UserService gRPC interface
type Service struct {
	app UsersApp
}

// NewService creates a new users gRPC service
func NewService(app UsersApp) *Service {
	return &Service{
		app: app,
	}
}

// Verify that Service implements the UserServiceHandler interface
var _ userv1connect.UserServiceHandler = (*Service)(nil)

// CreateUser creates a new user
func (s *Service) CreateUser(ctx context.Context, req *connect.Request[userv1.CreateUserRequest]) (*connect.Response[userv1.CreateUserResponse], error) {
	appReq := s.protoToCreateUserRequest(req.Msg)

	user, err := s.app.CreateUser(ctx, appReq)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	protoUser := s.userToProto(user)

	return connect.NewResponse(&userv1.CreateUserResponse{
		User: protoUser,
	}), nil
}

// GetUser retrieves a user by ID
func (s *Service) GetUser(ctx context.Context, req *connect.Request[userv1.GetUserRequest]) (*connect.Response[userv1.GetUserResponse], error) {
	id, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	user, err := s.app.GetUser(ctx, id)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	protoUser := s.userToProto(user)

	return connect.NewResponse(&userv1.GetUserResponse{
		User: protoUser,
	}), nil
}

// GetUserByUsername retrieves a user by username
func (s *Service) GetUserByUsername(ctx context.Context, req *connect.Request[userv1.GetUserByUsernameRequest]) (*connect.Response[userv1.GetUserByUsernameResponse], error) {
	user, err := s.app.GetUserByUsername(ctx, req.Msg.Username)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	protoUser := s.userToProto(user)

	return connect.NewResponse(&userv1.GetUserByUsernameResponse{
		User: protoUser,
	}), nil
}

// GetUserByEmail retrieves a user by email
func (s *Service) GetUserByEmail(ctx context.Context, req *connect.Request[userv1.GetUserByEmailRequest]) (*connect.Response[userv1.GetUserByEmailResponse], error) {
	user, err := s.app.GetUserByEmail(ctx, req.Msg.Email)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	protoUser := s.userToProto(user)

	return connect.NewResponse(&userv1.GetUserByEmailResponse{
		User: protoUser,
	}), nil
}

// UpdateUser updates an existing user
func (s *Service) UpdateUser(ctx context.Context, req *connect.Request[userv1.UpdateUserRequest]) (*connect.Response[userv1.UpdateUserResponse], error) {
	id, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	appReq := s.protoToUpdateUserRequest(req.Msg)

	user, err := s.app.UpdateUser(ctx, id, appReq)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	protoUser := s.userToProto(user)

	return connect.NewResponse(&userv1.UpdateUserResponse{
		User: protoUser,
	}), nil
}

// DeleteUser deletes a user by ID
func (s *Service) DeleteUser(ctx context.Context, req *connect.Request[userv1.DeleteUserRequest]) (*connect.Response[userv1.DeleteUserResponse], error) {
	id, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	err = s.app.DeleteUser(ctx, id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&userv1.DeleteUserResponse{
		Success: true,
	}), nil
}

// Conversion methods between proto and app layer models

func (s *Service) userToProto(user *models.User) *userv1.User {
	return &userv1.User{
		Id:        user.ID.String(),
		Username:  user.Username,
		Email:     user.Email,
		CreatedAt: timestamppb.New(user.CreatedAt),
	}
}

func (s *Service) protoToCreateUserRequest(proto *userv1.CreateUserRequest) CreateUserRequest {
	return CreateUserRequest{
		Username: proto.Username,
		Email:    proto.Email,
	}
}

func (s *Service) protoToUpdateUserRequest(proto *userv1.UpdateUserRequest) UpdateUserRequest {
	return UpdateUserRequest{
		Username: proto.Username,
		Email:    proto.Email,
	}
}
