package model

import (
	"time"

	"github.com/google/uuid"
)

// ActionTokenModel là persistence model cho token verify/reset password.
type ActionTokenModel struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID    uuid.UUID  `gorm:"type:uuid;not null;index"`
	Type      string     `gorm:"type:varchar(64);not null;index"`
	TokenHash string     `gorm:"type:varchar(255);uniqueIndex;not null"`
	ExpiresAt time.Time  `gorm:"not null"`
	UsedAt    *time.Time `gorm:"type:timestamptz"`
	Revoked   bool       `gorm:"not null;default:false"`
	CreatedAt time.Time  `gorm:"not null;autoCreateTime"`

	User *UserModel `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
}

// TableName trả về tên bảng trong database.
func (ActionTokenModel) TableName() string {
	return "action_tokens"
}
