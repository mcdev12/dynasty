package users

// CreateUserRequest represents the data needed to create a new user
type CreateUserRequest struct {
	Username string `json:"username" validate:"required"`
	Email    string `json:"email" validate:"required,email"`
}

// UpdateUserRequest represents the data that can be updated for a user
type UpdateUserRequest struct {
	Username string `json:"username" validate:"required"`
	Email    string `json:"email" validate:"required,email"`
}
