package repository

import (
	"context"
	"errors"

	"easy-orders-backend/internal/models"
	"easy-orders-backend/pkg/database"
	"easy-orders-backend/pkg/logger"

	"gorm.io/gorm"
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

	// Create a user in a database
	if err := r.db.WithContext(ctx).Create(user).Error; err != nil {
		r.logger.Error("Failed to create user", "error", err, "email", user.Email)
		return err
	}

	r.logger.Info("User created in database", "id", user.ID, "email", user.Email)
	return nil
}

func (r *userRepository) GetByID(ctx context.Context, id string) (*models.User, error) {
	r.logger.Debug("Getting user by ID", "id", id)

	var user models.User
	if err := r.db.WithContext(ctx).First(&user, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			r.logger.Debug("User not found", "id", id)
			return nil, nil
		}
		r.logger.Error("Failed to get user by ID", "error", err, "id", id)
		return nil, err
	}

	r.logger.Debug("User retrieved from database", "id", id)
	return &user, nil
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	r.logger.Debug("Getting user by email", "email", email)

	var user models.User
	if err := r.db.WithContext(ctx).First(&user, "email = ?", email).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			r.logger.Debug("User not found", "email", email)
			return nil, nil
		}
		r.logger.Error("Failed to get user by email", "error", err, "email", email)
		return nil, err
	}

	r.logger.Debug("User retrieved from database", "email", email)
	return &user, nil
}

func (r *userRepository) Update(ctx context.Context, user *models.User) error {
	r.logger.Debug("Updating user in database", "id", user.ID)

	if err := r.db.WithContext(ctx).Save(user).Error; err != nil {
		r.logger.Error("Failed to update user", "error", err, "id", user.ID)
		return err
	}

	r.logger.Info("User updated in database", "id", user.ID)
	return nil
}

func (r *userRepository) Delete(ctx context.Context, id string) error {
	r.logger.Debug("Deleting user from database", "id", id)

	// Soft delete the user
	if err := r.db.WithContext(ctx).Delete(&models.User{}, "id = ?", id).Error; err != nil {
		r.logger.Error("Failed to delete user", "error", err, "id", id)
		return err
	}

	r.logger.Info("User deleted from database", "id", id)
	return nil
}

func (r *userRepository) List(ctx context.Context, offset, limit int) ([]*models.User, error) {
	r.logger.Debug("Listing users from database", "offset", offset, "limit", limit)

	var users []*models.User
	if err := r.db.WithContext(ctx).
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&users).Error; err != nil {
		r.logger.Error("Failed to list users", "error", err)
		return nil, err
	}

	r.logger.Debug("Users retrieved from database", "count", len(users))
	return users, nil
}

func (r *userRepository) Count(ctx context.Context) (int64, error) {
	r.logger.Debug("Counting total users")

	var count int64
	if err := r.db.WithContext(ctx).Model(&models.User{}).Count(&count).Error; err != nil {
		r.logger.Error("Failed to count users", "error", err)
		return 0, err
	}

	r.logger.Debug("Total users counted", "count", count)
	return count, nil
}
