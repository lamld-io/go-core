package mapper

import (
	"github.com/base-go/base/services/auth/internal/domain"
	"github.com/base-go/base/services/auth/internal/repository/postgres/model"
)

// TokenToModel chuyển domain entity → GORM model.
func TokenToModel(t *domain.RefreshToken) *model.TokenModel {
	return &model.TokenModel{
		ID:        t.ID,
		UserID:    t.UserID,
		TokenHash: t.TokenHash,
		ExpiresAt: t.ExpiresAt,
		Revoked:   t.Revoked,
		CreatedAt: t.CreatedAt,
		IP:        t.IP,
		UserAgent: t.UserAgent,
		DeviceID:  t.DeviceID,
	}
}

// TokenToDomain chuyển GORM model → domain entity.
func TokenToDomain(m *model.TokenModel) *domain.RefreshToken {
	return &domain.RefreshToken{
		ID:        m.ID,
		UserID:    m.UserID,
		TokenHash: m.TokenHash,
		ExpiresAt: m.ExpiresAt,
		Revoked:   m.Revoked,
		CreatedAt: m.CreatedAt,
		IP:        m.IP,
		UserAgent: m.UserAgent,
		DeviceID:  m.DeviceID,
	}
}
