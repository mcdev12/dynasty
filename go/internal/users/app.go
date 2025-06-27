package users

import (
	"context"
	"fmt"
	"log"

	"github.com/google/uuid"
	"github.com/mcdev12/dynasty/go/internal/models"
)

// UsersRepository defines what the app layer needs from the repository
type UsersRepository interface {
	CreateUser(ctx context.Context, req CreateUserRequest) (*models.User, error)
	GetUser(ctx context.Context, id uuid.UUID) (*models.User, error)
	GetUserByUsername(ctx context.Context, username string) (*models.User, error)
	GetUserByEmail(ctx context.Context, email string) (*models.User, error)
	UpdateUser(ctx context.Context, id uuid.UUID, req UpdateUserRequest) (*models.User, error)
	DeleteUser(ctx context.Context, id uuid.UUID) error
}

// App handles users business logic
type App struct {
	repo UsersRepository
}

// NewApp creates a new users App
func NewApp(repo UsersRepository) *App {
	return &App{
		repo: repo,
	}
}

// CreateUser creates a new user with validation
func (a *App) CreateUser(ctx context.Context, req CreateUserRequest) (*models.User, error) {
	if err := a.validateCreateUserRequest(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Check if user with same username already exists
	existingUser, err := a.repo.GetUserByUsername(ctx, req.Username)
	if err == nil && existingUser != nil {
		return nil, fmt.Errorf("user with username %s already exists", req.Username)
	}

	// Check if user with same email already exists
	existingUser, err = a.repo.GetUserByEmail(ctx, req.Email)
	if err == nil && existingUser != nil {
		return nil, fmt.Errorf("user with email %s already exists", req.Email)
	}

	user, err := a.repo.CreateUser(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	log.Printf("Created user: %s (%s)", user.Username, user.Email)
	return user, nil
}

// GetUser retrieves a user by ID
func (a *App) GetUser(ctx context.Context, id uuid.UUID) (*models.User, error) {
	user, err := a.repo.GetUser(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	return user, nil
}

// GetUserByUsername retrieves a user by username
func (a *App) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	user, err := a.repo.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by username: %w", err)
	}
	return user, nil
}

// GetUserByEmail retrieves a user by email
func (a *App) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	user, err := a.repo.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}
	return user, nil
}

// UpdateUser updates an existing user with validation
func (a *App) UpdateUser(ctx context.Context, id uuid.UUID, req UpdateUserRequest) (*models.User, error) {
	if err := a.validateUpdateUserRequest(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Verify user exists
	existingUser, err := a.repo.GetUser(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// Check if username is being changed and if new username already exists
	if req.Username != existingUser.Username {
		conflictUser, err := a.repo.GetUserByUsername(ctx, req.Username)
		if err == nil && conflictUser != nil {
			return nil, fmt.Errorf("user with username %s already exists", req.Username)
		}
	}

	// Check if email is being changed and if new email already exists
	if req.Email != existingUser.Email {
		conflictUser, err := a.repo.GetUserByEmail(ctx, req.Email)
		if err == nil && conflictUser != nil {
			return nil, fmt.Errorf("user with email %s already exists", req.Email)
		}
	}

	user, err := a.repo.UpdateUser(ctx, id, req)
	if err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	log.Printf("Updated user: %s (%s)", user.Username, user.Email)
	return user, nil
}

// DeleteUser deletes a user by ID
func (a *App) DeleteUser(ctx context.Context, id uuid.UUID) error {
	// Verify user exists
	user, err := a.repo.GetUser(ctx, id)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	if err := a.repo.DeleteUser(ctx, id); err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	log.Printf("Deleted user: %s (%s)", user.Username, user.Email)
	return nil
}

// validateCreateUserRequest validates create user request
func (a *App) validateCreateUserRequest(req CreateUserRequest) error {
	if req.Username == "" {
		return fmt.Errorf("username is required")
	}
	if req.Email == "" {
		return fmt.Errorf("email is required")
	}
	// Basic email validation (can be enhanced with proper regex)
	if !contains(req.Email, "@") || !contains(req.Email, ".") {
		return fmt.Errorf("email format is invalid")
	}
	return nil
}

// validateUpdateUserRequest validates update user request
func (a *App) validateUpdateUserRequest(req UpdateUserRequest) error {
	if req.Username == "" {
		return fmt.Errorf("username cannot be empty")
	}
	if req.Email == "" {
		return fmt.Errorf("email cannot be empty")
	}
	// Basic email validation (can be enhanced with proper regex)
	if !contains(req.Email, "@") || !contains(req.Email, ".") {
		return fmt.Errorf("email format is invalid")
	}
	return nil
}

// contains is a simple helper to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (len(substr) == 0 || func() bool {
		for i := 0; i <= len(s)-len(substr); i++ {
			if s[i:i+len(substr)] == substr {
				return true
			}
		}
		return false
	}())
}
