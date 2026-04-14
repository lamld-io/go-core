package postgres

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/base-go/base/services/auth/internal/domain"
	"github.com/base-go/base/services/auth/internal/repository/postgres/mapper"
	"github.com/base-go/base/services/auth/internal/repository/postgres/model"
)

// actionTokenRepository implement domain.ActionTokenRepository bằng GORM.
type actionTokenRepository struct {
	db *gorm.DB
}

// NewActionTokenRepository tạo repository token nghiệp vụ mới.
func NewActionTokenRepository(db *gorm.DB) domain.ActionTokenRepository {
	return &actionTokenRepository{db: db}
}

func (r *actionTokenRepository) Create(ctx context.Context, token *domain.ActionToken) error {
	m := mapper.ActionTokenToModel(token)

	result := r.db.WithContext(ctx).Create(m)
	if result.Error != nil {
		slog.ErrorContext(ctx, "failed to create action token", "error", result.Error, "user_id", token.UserID, "type", token.Type)
		return result.Error
	}

	token.ID = m.ID
	token.CreatedAt = m.CreatedAt
	return nil
}

func (r *actionTokenRepository) GetUsableByTokenHash(ctx context.Context, tokenHash string, tokenType domain.ActionTokenType) (*domain.ActionToken, error) {
	var m model.ActionTokenModel

	result := r.db.WithContext(ctx).
		Where("token_hash = ? AND type = ? AND revoked = false AND used_at IS NULL", tokenHash, string(tokenType)).
		First(&m)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, domain.ErrActionTokenNotFound
		}
		slog.ErrorContext(ctx, "failed to get action token", "error", result.Error, "type", tokenType)
		return nil, result.Error
	}

	return mapper.ActionTokenToDomain(&m), nil
}

func (r *actionTokenRepository) MarkUsed(ctx context.Context, id uuid.UUID, usedAt time.Time) error {
	result := r.db.WithContext(ctx).
		Model(&model.ActionTokenModel{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"used_at": usedAt,
		})
	if result.Error != nil {
		slog.ErrorContext(ctx, "failed to mark action token as used", "error", result.Error, "token_id", id)
		return result.Error
	}
	return nil
}

func (r *actionTokenRepository) RevokeByUserAndType(ctx context.Context, userID uuid.UUID, tokenType domain.ActionTokenType) error {
	result := r.db.WithContext(ctx).
		Model(&model.ActionTokenModel{}).
		Where("user_id = ? AND type = ? AND revoked = false", userID, string(tokenType)).
		Update("revoked", true)
	if result.Error != nil {
		slog.ErrorContext(ctx, "failed to revoke action tokens", "error", result.Error, "user_id", userID, "type", tokenType)
		return result.Error
	}
	return nil
}

func (r *actionTokenRepository) DeleteExpired(ctx context.Context) error {
	result := r.db.WithContext(ctx).
		Where("expires_at < ?", time.Now()).
		Delete(&model.ActionTokenModel{})
	if result.Error != nil {
		slog.ErrorContext(ctx, "failed to delete expired action tokens", "error", result.Error)
		return result.Error
	}
	return nil
}
