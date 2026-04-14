package domain

import (
	"time"

	"github.com/google/uuid"
)

// ActionTokenType phân biệt loại token nghiệp vụ.
type ActionTokenType string

const (
	ActionTokenTypeEmailVerification ActionTokenType = "email_verification"
	ActionTokenTypePasswordReset     ActionTokenType = "password_reset"
)

// ActionToken đại diện cho token dùng cho verify email hoặc reset password.
type ActionToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Type      ActionTokenType
	TokenHash string
	ExpiresAt time.Time
	UsedAt    *time.Time
	Revoked   bool
	CreatedAt time.Time
}

// IsExpired kiểm tra token đã hết hạn chưa.
func (t *ActionToken) IsExpired(now time.Time) bool {
	return now.After(t.ExpiresAt)
}

// IsUsable kiểm tra token còn dùng được không.
func (t *ActionToken) IsUsable(now time.Time) bool {
	return !t.Revoked && t.UsedAt == nil && !t.IsExpired(now)
}
