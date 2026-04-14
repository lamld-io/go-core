package postgres

import (
	"context"
	"errors"
	"log/slog"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/base-go/base/services/auth/internal/domain"
	"github.com/base-go/base/services/auth/internal/repository/postgres/mapper"
	"github.com/base-go/base/services/auth/internal/repository/postgres/model"
)

// loginLockoutPolicyRepository implement domain.LoginLockoutPolicyRepository bằng GORM.
type loginLockoutPolicyRepository struct {
	db *gorm.DB
}

// NewLoginLockoutPolicyRepository tạo repository policy lockout.
func NewLoginLockoutPolicyRepository(db *gorm.DB) domain.LoginLockoutPolicyRepository {
	return &loginLockoutPolicyRepository{db: db}
}

func (r *loginLockoutPolicyRepository) Get(ctx context.Context) (*domain.LoginLockoutPolicy, error) {
	var m model.LoginLockoutPolicyModel

	result := r.db.WithContext(ctx).First(&m, 1)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, domain.ErrLoginLockoutPolicyNotFound
		}
		slog.ErrorContext(ctx, "failed to get login lockout policy", "error", result.Error)
		return nil, result.Error
	}

	return mapper.LoginLockoutPolicyToDomain(&m), nil
}

func (r *loginLockoutPolicyRepository) Upsert(ctx context.Context, policy *domain.LoginLockoutPolicy) error {
	m := mapper.LoginLockoutPolicyToModel(policy)

	result := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "id"}},
			DoUpdates: clause.AssignmentColumns([]string{"max_failed_attempts", "lock_duration_seconds", "updated_at"}),
		}).
		Create(m)
	if result.Error != nil {
		slog.ErrorContext(ctx, "failed to upsert login lockout policy", "error", result.Error)
		return result.Error
	}

	return nil
}
