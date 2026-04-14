package domain

import (
	"time"

	"github.com/google/uuid"
)

// User là entity chính của hệ thống xác thực.
type User struct {
	ID                  uuid.UUID
	Email               string
	PasswordHash        string
	FullName            *string // nullable
	Role                string
	EmailVerified       bool
	IsActive            bool
	FailedLoginAttempts int
	LockedUntil         *time.Time
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// IsLocked kiểm tra user có đang bị khoá tạm thời hay không.
func (u *User) IsLocked(now time.Time) bool {
	return u.LockedUntil != nil && now.Before(*u.LockedUntil)
}

// DefaultRole là role mặc định khi đăng ký.
const DefaultRole = "user"

// Các role hợp lệ trong hệ thống.
const (
	RoleAdmin = "admin"
	RoleUser  = "user"
)

// ValidRoles trả về danh sách role hợp lệ.
func ValidRoles() []string {
	return []string{RoleAdmin, RoleUser}
}

// IsValidRole kiểm tra role có hợp lệ không.
func IsValidRole(role string) bool {
	for _, r := range ValidRoles() {
		if r == role {
			return true
		}
	}
	return false
}
