package users

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/mcdev12/dynasty/go/internal/models"
	"github.com/mcdev12/dynasty/go/internal/users/db"
)

// Querier defines what the repository needs from the database layer
type Querier interface {
	CreateUser(ctx context.Context, arg db.CreateUserParams) (db.User, error)
	GetUser(ctx context.Context, id uuid.UUID) (db.User, error)
	GetUserByUsername(ctx context.Context, username string) (db.User, error)
	GetUserByEmail(ctx context.Context, email string) (db.User, error)
	UpdateUser(ctx context.Context, arg db.UpdateUserParams) (db.User, error)
	DeleteUser(ctx context.Context, id uuid.UUID) error
}

// Repository implements user data access operations
type Repository struct {
	queries Querier
}

// NewRepository creates a new users repository
func NewRepository(querier Querier) *Repository {
	return &Repository{
		queries: querier,
	}
}

// CreateUser creates a new user
func (r *Repository) CreateUser(ctx context.Context, req CreateUserRequest) (*models.User, error) {
	user, err := r.queries.CreateUser(ctx, db.CreateUserParams{
		Username: req.Username,
		Email:    req.Email,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return r.dbUserToModel(user), nil
}

// GetUser retrieves a user by ID
func (r *Repository) GetUser(ctx context.Context, id uuid.UUID) (*models.User, error) {
	user, err := r.queries.GetUser(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return r.dbUserToModel(user), nil
}

// GetUserByUsername retrieves a user by username
func (r *Repository) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	user, err := r.queries.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by username: %w", err)
	}

	return r.dbUserToModel(user), nil
}

// GetUserByEmail retrieves a user by email
func (r *Repository) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	user, err := r.queries.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	return r.dbUserToModel(user), nil
}

// UpdateUser updates an existing user
func (r *Repository) UpdateUser(ctx context.Context, id uuid.UUID, req UpdateUserRequest) (*models.User, error) {
	user, err := r.queries.UpdateUser(ctx, db.UpdateUserParams{
		ID:       id,
		Username: req.Username,
		Email:    req.Email,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	return r.dbUserToModel(user), nil
}

// DeleteUser deletes a user by ID
func (r *Repository) DeleteUser(ctx context.Context, id uuid.UUID) error {
	if err := r.queries.DeleteUser(ctx, id); err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}
	return nil
}

// dbUserToModel converts a database user to domain model
func (r *Repository) dbUserToModel(dbUser db.User) *models.User {
	return &models.User{
		ID:        dbUser.ID,
		Username:  dbUser.Username,
		Email:     dbUser.Email,
		CreatedAt: dbUser.CreatedAt,
	}
}
