package services

import (
	"context"
	"errors"

	"easy-orders-backend/internal/models"
	"easy-orders-backend/internal/repository"
	"easy-orders-backend/pkg/jwt"
	"easy-orders-backend/pkg/logger"

	"golang.org/x/crypto/bcrypt"
)

// userService implements UserService interface
type userService struct {
	userRepo     repository.UserRepository
	tokenManager *jwt.TokenManager
	logger       *logger.Logger
}

// NewUserService creates a new user service
func NewUserService(userRepo repository.UserRepository, tokenManager *jwt.TokenManager, logger *logger.Logger) UserService {
	return &userService{
		userRepo:     userRepo,
		tokenManager: tokenManager,
		logger:       logger,
	}
}

func (s *userService) CreateUser(ctx context.Context, req CreateUserRequest) (*UserResponse, error) {
	s.logger.Info("Creating user", "email", req.Email)

	// Validate request
	if req.Email == "" {
		return nil, errors.New("email is required")
	}
	if req.Name == "" {
		return nil, errors.New("name is required")
	}
	if req.Password == "" {
		return nil, errors.New("password is required")
	}

	// Check if a user already exists
	existingUser, err := s.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		s.logger.Error("Failed to check existing user", "error", err, "email", req.Email)
		return nil, err
	}
	if existingUser != nil {
		s.logger.Warn("User already exists", "email", req.Email)
		return nil, errors.New("user with this email already exists")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		s.logger.Error("Failed to hash password", "error", err)
		return nil, errors.New("failed to process password")
	}

	user := &models.User{
		Email:    req.Email,
		Name:     req.Name,
		Password: string(hashedPassword),
		Role:     models.UserRoleCustomer, // Default role
		IsActive: true,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		s.logger.Error("Failed to create user", "error", err, "email", req.Email)
		return nil, err
	}

	s.logger.Info("User created successfully", "id", user.ID, "email", req.Email)

	return &UserResponse{
		ID:       user.ID,
		Email:    user.Email,
		Name:     user.Name,
		Role:     string(user.Role),
		IsActive: user.IsActive,
	}, nil
}

func (s *userService) GetUser(ctx context.Context, id string) (*UserResponse, error) {
	s.logger.Debug("Getting user", "id", id)

	if id == "" {
		return nil, errors.New("user ID is required")
	}

	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		s.logger.Error("Failed to get user", "error", err, "id", id)
		return nil, err
	}

	if user == nil {
		return nil, errors.New("user not found")
	}

	return &UserResponse{
		ID:       user.ID,
		Email:    user.Email,
		Name:     user.Name,
		Role:     string(user.Role),
		IsActive: user.IsActive,
	}, nil
}

func (s *userService) UpdateUser(ctx context.Context, id string, req UpdateUserRequest) (*UserResponse, error) {
	s.logger.Info("Updating user", "id", id)

	// Get existing user
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		s.logger.Error("Failed to get user for update", "error", err, "id", id)
		return nil, err
	}

	// Update fields if provided
	if req.Name != nil {
		user.Name = *req.Name
	}
	if req.Email != nil {
		user.Email = *req.Email
	}

	if err := s.userRepo.Update(ctx, user); err != nil {
		s.logger.Error("Failed to update user", "error", err, "id", id)
		return nil, err
	}

	s.logger.Info("User updated successfully", "id", id)

	return &UserResponse{
		ID:    user.ID,
		Email: user.Email,
		Name:  user.Name,
	}, nil
}

func (s *userService) DeleteUser(ctx context.Context, id string) error {
	s.logger.Info("Deleting user", "id", id)

	if err := s.userRepo.Delete(ctx, id); err != nil {
		s.logger.Error("Failed to delete user", "error", err, "id", id)
		return err
	}

	s.logger.Info("User deleted successfully", "id", id)
	return nil
}

func (s *userService) ListUsers(ctx context.Context, req ListUsersRequest) (*ListUsersResponse, error) {
	s.logger.Debug("Listing users", "offset", req.Offset, "limit", req.Limit)

	users, err := s.userRepo.List(ctx, req.Offset, req.Limit)
	if err != nil {
		s.logger.Error("Failed to list users", "error", err)
		return nil, err
	}

	userResponses := make([]*UserResponse, len(users))
	for i, user := range users {
		userResponses[i] = &UserResponse{
			ID:       user.ID,
			Email:    user.Email,
			Name:     user.Name,
			Role:     string(user.Role),
			IsActive: user.IsActive,
		}
	}

	return &ListUsersResponse{
		Users:  userResponses,
		Offset: req.Offset,
		Limit:  req.Limit,
		Total:  len(userResponses), // TODO: Get actual total count
	}, nil
}

func (s *userService) AuthenticateUser(ctx context.Context, email, password string) (*AuthResponse, error) {
	s.logger.Debug("Authenticating user", "email", email)

	if email == "" || password == "" {
		return nil, errors.New("email and password are required")
	}

	// Get user by email
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		s.logger.Error("Failed to get user for authentication", "error", err, "email", email)
		return nil, errors.New("invalid credentials")
	}

	if user == nil {
		s.logger.Warn("User not found during authentication", "email", email)
		return nil, errors.New("invalid credentials")
	}

	if !user.IsActive {
		s.logger.Warn("Inactive user attempted login", "email", email)
		return nil, errors.New("account is deactivated")
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		s.logger.Warn("Invalid password for user", "email", email)
		return nil, errors.New("invalid credentials")
	}

	// Generate JWT token
	token, err := s.tokenManager.GenerateToken(user)
	if err != nil {
		s.logger.Error("Failed to generate JWT token", "error", err, "user_id", user.ID)
		return nil, errors.New("failed to generate authentication token")
	}

	s.logger.Info("User authenticated successfully", "user_id", user.ID, "email", user.Email)

	return &AuthResponse{
		Token: token,
		User: UserResponse{
			ID:       user.ID,
			Email:    user.Email,
			Name:     user.Name,
			Role:     string(user.Role),
			IsActive: user.IsActive,
		},
	}, nil
}
