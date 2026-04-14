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

// tokenRepository implement domain.TokenRepository bằng GORM.
type tokenRepository struct {
	db *gorm.DB
}

// NewTokenRepository tạo TokenRepository mới.
func NewTokenRepository(db *gorm.DB) domain.TokenRepository {
	return &tokenRepository{db: db}
}

func (r *tokenRepository) Create(ctx context.Context, token *domain.RefreshToken) error {
	m := mapper.TokenToModel(token)

	result := r.db.WithContext(ctx).Create(m)
	if result.Error != nil {
		slog.ErrorContext(ctx, "failed to create refresh token", "error", result.Error, "user_id", token.UserID)
		return result.Error
	}

	token.ID = m.ID
	token.CreatedAt = m.CreatedAt
	return nil
}

func (r *tokenRepository) GetByTokenHash(ctx context.Context, tokenHash string) (*domain.RefreshToken, error) {
	var m model.TokenModel

	result := r.db.WithContext(ctx).Where("token_hash = ?", tokenHash).First(&m)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, domain.ErrTokenNotFound
		}
		slog.ErrorContext(ctx, "failed to get token by hash", "error", result.Error)
		return nil, result.Error
	}

	return mapper.TokenToDomain(&m), nil
}

func (r *tokenRepository) RevokeByUserID(ctx context.Context, userID uuid.UUID) error {
	result := r.db.WithContext(ctx).
		Model(&model.TokenModel{}).
		Where("user_id = ? AND revoked = false", userID).
		Update("revoked", true)

	if result.Error != nil {
		slog.ErrorContext(ctx, "failed to revoke tokens by user_id", "error", result.Error, "user_id", userID)
		return result.Error
	}
	return nil
}

func (r *tokenRepository) RevokeByID(ctx context.Context, id uuid.UUID) error {
	result := r.db.WithContext(ctx).
		Model(&model.TokenModel{}).
		Where("id = ?", id).
		Update("revoked", true)

	if result.Error != nil {
		slog.ErrorContext(ctx, "failed to revoke token by id", "error", result.Error, "token_id", id)
		return result.Error
	}
	return nil
}

func (r *tokenRepository) DeleteExpired(ctx context.Context) error {
	result := r.db.WithContext(ctx).
		Where("expires_at < ?", time.Now()).
		Delete(&model.TokenModel{})

	if result.Error != nil {
		slog.ErrorContext(ctx, "failed to delete expired tokens", "error", result.Error)
		return result.Error
	}

	if result.RowsAffected > 0 {
		slog.InfoContext(ctx, "cleaned up expired tokens", "count", result.RowsAffected)
	}
	return nil
}
