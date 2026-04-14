package domain

import "time"

// LoginLockoutPolicy là policy khoá tài khoản khi đăng nhập sai nhiều lần.
type LoginLockoutPolicy struct {
	MaxFailedAttempts int
	LockDuration      time.Duration
}

// Disabled cho biết policy đang tắt.
func (p LoginLockoutPolicy) Disabled() bool {
	return p.MaxFailedAttempts <= 0 || p.LockDuration <= 0
}
