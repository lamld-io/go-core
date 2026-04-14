package mapper

import (
	"time"

	"github.com/base-go/base/services/auth/internal/domain"
	"github.com/base-go/base/services/auth/internal/repository/postgres/model"
)

// LoginLockoutPolicyToModel chuyển domain policy sang persistence model.
func LoginLockoutPolicyToModel(policy *domain.LoginLockoutPolicy) *model.LoginLockoutPolicyModel {
	return &model.LoginLockoutPolicyModel{
		ID:                  1,
		MaxFailedAttempts:   policy.MaxFailedAttempts,
		LockDurationSeconds: int64(policy.LockDuration / time.Second),
	}
}

// LoginLockoutPolicyToDomain chuyển persistence model sang domain policy.
func LoginLockoutPolicyToDomain(m *model.LoginLockoutPolicyModel) *domain.LoginLockoutPolicy {
	return &domain.LoginLockoutPolicy{
		MaxFailedAttempts: m.MaxFailedAttempts,
		LockDuration:      time.Duration(m.LockDurationSeconds) * time.Second,
	}
}
