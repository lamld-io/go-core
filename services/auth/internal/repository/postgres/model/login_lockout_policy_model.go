package model

import "time"

// LoginLockoutPolicyModel lưu policy khoá tài khoản có thể cập nhật runtime.
type LoginLockoutPolicyModel struct {
	ID                  uint      `gorm:"primaryKey"`
	MaxFailedAttempts   int       `gorm:"not null"`
	LockDurationSeconds int64     `gorm:"not null"`
	CreatedAt           time.Time `gorm:"not null;autoCreateTime"`
	UpdatedAt           time.Time `gorm:"not null;autoUpdateTime"`
}

// TableName trả về tên bảng trong database.
func (LoginLockoutPolicyModel) TableName() string {
	return "login_lockout_policies"
}
