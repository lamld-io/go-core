package model

import (
	"time"

	"github.com/google/uuid"
)

// TokenModel là GORM persistence model cho bảng refresh_tokens.
type TokenModel struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index"`
	TokenHash string    `gorm:"type:varchar(255);uniqueIndex;not null"`
	ExpiresAt time.Time `gorm:"not null"`
	Revoked   bool      `gorm:"not null;default:false"`
	CreatedAt time.Time `gorm:"not null;autoCreateTime"`

	// Foreign key relationship (chỉ dùng ở persistence layer).
	User *UserModel `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
}

// TableName trả về tên bảng trong database.
func (TokenModel) TableName() string {
	return "refresh_tokens"
}
