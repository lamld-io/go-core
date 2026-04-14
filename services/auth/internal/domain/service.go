package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// TokenPair chứa cặp access token và refresh token.
type TokenPair struct {
	AccessToken  string
	RefreshToken string
}

// AuthService định nghĩa interface cho các use case xác thực.
// Tầng delivery gọi interface này, tầng usecase implement.
type AuthService interface {
	// Register đăng ký tài khoản mới.
	Register(ctx context.Context, email, password, fullName string) (*User, error)

	// Login xác thực và trả về token pair.
	Login(ctx context.Context, email, password string, meta ClientMetadata) (*User, *TokenPair, error)

	// VerifyEmail xác thực email bằng token một lần.
	VerifyEmail(ctx context.Context, token string) error

	// ResendVerificationEmail gửi lại email xác thực.
	ResendVerificationEmail(ctx context.Context, email string) error

	// ForgotPassword phát hành token reset password.
	ForgotPassword(ctx context.Context, email string) error

	// ResetPassword cập nhật mật khẩu mới bằng token reset.
	ResetPassword(ctx context.Context, token, newPassword string) error

	// RefreshToken làm mới access token từ refresh token.
	RefreshToken(ctx context.Context, refreshToken string, meta ClientMetadata) (*TokenPair, error)

	// Logout thu hồi refresh token và đưa access token vào danh sách đen.
	Logout(ctx context.Context, userID uuid.UUID, accessTokenStr string) error

	// GetProfile lấy thông tin user hiện tại.
	GetProfile(ctx context.Context, userID uuid.UUID) (*User, error)

	// ValidateToken xác thực access token và trả về thông tin user.
	ValidateToken(ctx context.Context, accessToken string) (*User, error)

	// GetLoginLockoutPolicy lấy policy khoá tài khoản hiện tại.
	GetLoginLockoutPolicy(ctx context.Context) (*LoginLockoutPolicy, error)

	// UpdateLoginLockoutPolicy cập nhật policy khoá tài khoản.
	UpdateLoginLockoutPolicy(ctx context.Context, maxFailedAttempts int, lockDuration time.Duration) (*LoginLockoutPolicy, error)
}

// TokenBlacklist lưu trữ danh sách các access token bị vô hiệu hóa trước hạn.
type TokenBlacklist interface {
	BlacklistToken(ctx context.Context, tokenID string, expiresAt time.Time) error
	IsBlacklisted(ctx context.Context, tokenID string) (bool, error)
}
