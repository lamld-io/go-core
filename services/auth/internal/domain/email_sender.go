package domain

import (
	"context"
	"time"
)

// EmailSender là boundary gửi email ra ngoài hệ thống.
type EmailSender interface {
	SendVerificationEmail(ctx context.Context, user *User, token string, expiresAt time.Time) error
	SendPasswordResetEmail(ctx context.Context, user *User, token string, expiresAt time.Time) error
}
