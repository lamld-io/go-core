package domain

import (
	"time"

	"github.com/google/uuid"
)

// RefreshToken đại diện cho refresh token được lưu trong database.
type RefreshToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TokenHash string
	ExpiresAt time.Time
	Revoked   bool
	CreatedAt time.Time
}

// IsExpired kiểm tra token đã hết hạn chưa.
func (t *RefreshToken) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}

// IsUsable kiểm tra token còn sử dụng được không (chưa bị thu hồi và chưa hết hạn).
func (t *RefreshToken) IsUsable() bool {
	return !t.Revoked && !t.IsExpired()
}
