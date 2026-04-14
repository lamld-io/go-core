package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UserModel là GORM persistence model cho bảng users.
// Tách biệt khỏi domain entity — không dùng trực tiếp trong business logic.
type UserModel struct {
	ID                  uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Email               string         `gorm:"type:varchar(255);uniqueIndex;not null"`
	PasswordHash        string         `gorm:"type:varchar(255);not null"`
	FullName            *string        `gorm:"type:varchar(255)"`
	Role                string         `gorm:"type:varchar(50);not null;default:'user'"`
	EmailVerified       bool           `gorm:"not null;default:false"`
	IsActive            bool           `gorm:"not null;default:true"`
	FailedLoginAttempts int            `gorm:"not null;default:0"`
	LockedUntil         *time.Time     `gorm:"type:timestamptz"`
	CreatedAt           time.Time      `gorm:"not null;autoCreateTime"`
	UpdatedAt           time.Time      `gorm:"not null;autoUpdateTime"`
	DeletedAt           gorm.DeletedAt `gorm:"index"`
}

// TableName trả về tên bảng trong database.
func (UserModel) TableName() string {
	return "users"
}
