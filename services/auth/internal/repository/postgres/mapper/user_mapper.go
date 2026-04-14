package mapper

import (
	"github.com/base-go/base/services/auth/internal/domain"
	"github.com/base-go/base/services/auth/internal/repository/postgres/model"
)

// UserToModel chuyển domain entity → GORM model.
func UserToModel(u *domain.User) *model.UserModel {
	return &model.UserModel{
		ID:                  u.ID,
		Email:               u.Email,
		PasswordHash:        u.PasswordHash,
		FullName:            u.FullName,
		Role:                u.Role,
		EmailVerified:       u.EmailVerified,
		IsActive:            u.IsActive,
		FailedLoginAttempts: u.FailedLoginAttempts,
		LockedUntil:         u.LockedUntil,
		CreatedAt:           u.CreatedAt,
		UpdatedAt:           u.UpdatedAt,
	}
}

// UserToDomain chuyển GORM model → domain entity.
func UserToDomain(m *model.UserModel) *domain.User {
	return &domain.User{
		ID:                  m.ID,
		Email:               m.Email,
		PasswordHash:        m.PasswordHash,
		FullName:            m.FullName,
		Role:                m.Role,
		EmailVerified:       m.EmailVerified,
		IsActive:            m.IsActive,
		FailedLoginAttempts: m.FailedLoginAttempts,
		LockedUntil:         m.LockedUntil,
		CreatedAt:           m.CreatedAt,
		UpdatedAt:           m.UpdatedAt,
	}
}
