package postgres

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/base-go/base/services/auth/internal/domain"
	"github.com/base-go/base/services/auth/internal/repository/postgres/mapper"
	"github.com/base-go/base/services/auth/internal/repository/postgres/model"
)

// userRepository implement domain.UserRepository bằng GORM.
type userRepository struct {
	db *gorm.DB
}

// NewUserRepository tạo UserRepository mới.
func NewUserRepository(db *gorm.DB) domain.UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, user *domain.User) error {
	m := mapper.UserToModel(user)

	result := r.db.WithContext(ctx).Create(m)
	if result.Error != nil {
		// Kiểm tra duplicate email (unique constraint violation).
		if strings.Contains(result.Error.Error(), "duplicate") ||
			strings.Contains(result.Error.Error(), "unique") {
			return domain.ErrUserAlreadyExists
		}
		slog.ErrorContext(ctx, "failed to create user", "error", result.Error)
		return result.Error
	}

	// Cập nhật ID sinh bởi DB vào domain entity.
	user.ID = m.ID
	user.CreatedAt = m.CreatedAt
	user.UpdatedAt = m.UpdatedAt
	return nil
}

func (r *userRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	var m model.UserModel

	result := r.db.WithContext(ctx).Where("id = ?", id).First(&m)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, domain.ErrUserNotFound
		}
		slog.ErrorContext(ctx, "failed to get user by id", "error", result.Error, "user_id", id)
		return nil, result.Error
	}

	return mapper.UserToDomain(&m), nil
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	var m model.UserModel

	result := r.db.WithContext(ctx).Where("email = ?", email).First(&m)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, domain.ErrUserNotFound
		}
		slog.ErrorContext(ctx, "failed to get user by email", "error", result.Error)
		return nil, result.Error
	}

	return mapper.UserToDomain(&m), nil
}

func (r *userRepository) Update(ctx context.Context, user *domain.User) error {
	m := mapper.UserToModel(user)

	result := r.db.WithContext(ctx).Save(m)
	if result.Error != nil {
		slog.ErrorContext(ctx, "failed to update user", "error", result.Error, "user_id", user.ID)
		return result.Error
	}

	user.UpdatedAt = m.UpdatedAt
	return nil
}
