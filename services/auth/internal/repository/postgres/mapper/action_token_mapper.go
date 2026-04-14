package mapper

import (
	"github.com/base-go/base/services/auth/internal/domain"
	"github.com/base-go/base/services/auth/internal/repository/postgres/model"
)

// ActionTokenToModel chuyển domain token sang persistence model.
func ActionTokenToModel(t *domain.ActionToken) *model.ActionTokenModel {
	return &model.ActionTokenModel{
		ID:        t.ID,
		UserID:    t.UserID,
		Type:      string(t.Type),
		TokenHash: t.TokenHash,
		ExpiresAt: t.ExpiresAt,
		UsedAt:    t.UsedAt,
		Revoked:   t.Revoked,
		CreatedAt: t.CreatedAt,
	}
}

// ActionTokenToDomain chuyển persistence model sang domain token.
func ActionTokenToDomain(m *model.ActionTokenModel) *domain.ActionToken {
	return &domain.ActionToken{
		ID:        m.ID,
		UserID:    m.UserID,
		Type:      domain.ActionTokenType(m.Type),
		TokenHash: m.TokenHash,
		ExpiresAt: m.ExpiresAt,
		UsedAt:    m.UsedAt,
		Revoked:   m.Revoked,
		CreatedAt: m.CreatedAt,
	}
}
