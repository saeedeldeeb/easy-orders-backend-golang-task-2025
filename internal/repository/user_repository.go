package repository

import (
	"context"
	"fmt"

	"easy-orders-backend/internal/models"
	"easy-orders-backend/pkg/database"
	"easy-orders-backend/pkg/logger"

	"github.com/google/uuid"
)

// userRepository implements UserRepository interface
type userRepository struct {
	db     *database.DB
	logger *logger.Logger
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *database.DB, logger *logger.Logger) UserRepository {
	return &userRepository{
		db:     db,
		logger: logger,
	}
}

func (r *userRepository) Create(ctx context.Context, user *models.User) error {
	r.logger.Debug("Creating user in database", "email", user.Email)

	// Generate UUID for new user
	user.ID = uuid.New().String()

	// TODO: Replace with actual GORM model and database insert
	// For now, just simulate the operation
	r.logger.Info("User created in database", "id", user.ID, "email", user.Email)
	return nil
}

func (r *userRepository) GetByID(ctx context.Context, id string) (*models.User, error) {
	r.logger.Debug("Getting user by ID", "id", id)

	// TODO: Replace with actual GORM query
	// For now, return a mock user
	user := &models.User{
		ID:    id,
		Email: "user@example.com",
		Name:  "Mock User",
	}

	r.logger.Debug("User retrieved from database", "id", id)
	return user, nil
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	r.logger.Debug("Getting user by email", "email", email)

	// TODO: Replace with actual GORM query
	// For now, return a mock user
	user := &models.User{
		ID:    uuid.New().String(),
		Email: email,
		Name:  "Mock User",
	}

	r.logger.Debug("User retrieved from database", "email", email)
	return user, nil
}

func (r *userRepository) Update(ctx context.Context, user *models.User) error {
	r.logger.Debug("Updating user in database", "id", user.ID)

	// TODO: Replace with actual GORM update
	// For now, just simulate the operation
	r.logger.Info("User updated in database", "id", user.ID)
	return nil
}

func (r *userRepository) Delete(ctx context.Context, id string) error {
	r.logger.Debug("Deleting user from database", "id", id)

	// TODO: Replace with actual GORM delete
	// For now, just simulate the operation
	r.logger.Info("User deleted from database", "id", id)
	return nil
}

func (r *userRepository) List(ctx context.Context, offset, limit int) ([]*models.User, error) {
	r.logger.Debug("Listing users from database", "offset", offset, "limit", limit)

	// TODO: Replace with actual GORM query
	// For now, return mock users
	users := make([]*models.User, 0)
	for i := 0; i < limit && i < 5; i++ { // Mock up to 5 users
		users = append(users, &models.User{
			ID:    uuid.New().String(),
			Email: fmt.Sprintf("user%d@example.com", i+1),
			Name:  fmt.Sprintf("Mock User %d", i+1),
		})
	}

	r.logger.Debug("Users retrieved from database", "count", len(users))
	return users, nil
}
