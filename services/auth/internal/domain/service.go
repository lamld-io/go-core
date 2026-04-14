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

// LoginResult chứa kết quả của quá trình đăng nhập (có thể là TokenPair hoặc yêu cầu 2FA).
type LoginResult struct {
	User        *User
	TokenPair   *TokenPair
	Requires2FA bool
	TempToken   string
}

// AuthService định nghĩa interface cho các use case xác thực.
// Tầng delivery gọi interface này, tầng usecase implement.
type AuthService interface {
	// Register đăng ký tài khoản mới.
	Register(ctx context.Context, email, password, fullName string) (*User, error)

	// Login xác thực và trả về LoginResult (có thể bao gồm bước yêu cầu 2FA).
	Login(ctx context.Context, email, password string, meta ClientMetadata) (*LoginResult, error)

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

	// GetSessions lấy danh sách các phiên đăng nhập (refresh token) đang active của user.
	GetSessions(ctx context.Context, userID uuid.UUID) ([]*RefreshToken, error)

	// DeleteSession xóa/thu hồi một phiên đăng nhập cụ thể theo ID.
	DeleteSession(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID) error

	// ValidateToken xác thực access token và trả về thông tin user.
	ValidateToken(ctx context.Context, accessToken string) (*User, error)

	// GetLoginLockoutPolicy lấy policy khoá tài khoản hiện tại.
	GetLoginLockoutPolicy(ctx context.Context) (*LoginLockoutPolicy, error)

	// UpdateLoginLockoutPolicy cập nhật policy khoá tài khoản.
	UpdateLoginLockoutPolicy(ctx context.Context, maxFailedAttempts int, lockDuration time.Duration) (*LoginLockoutPolicy, error)

	// Setup2FA khởi tạo TOTP 2FA.
	Setup2FA(ctx context.Context, userID uuid.UUID) (*Setup2FAResponse, error)

	// Verify2FASetup xác thực mã OTP để bật 2FA.
	Verify2FASetup(ctx context.Context, userID uuid.UUID, code string) error

	// Verify2FALogin xác thực mã OTP trong quá trình đăng nhập.
	Verify2FALogin(ctx context.Context, tempToken, code string, meta ClientMetadata) (*LoginResult, error)
}

// Setup2FAResponse trả về thông tin cấu hình 2FA (Secret, URL)
type Setup2FAResponse struct {
	Secret    string
	SecretURL string
}

// TokenBlacklist lưu trữ danh sách các access token bị vô hiệu hóa trước hạn.
type TokenBlacklist interface {
	BlacklistToken(ctx context.Context, tokenID string, expiresAt time.Time) error
	IsBlacklisted(ctx context.Context, tokenID string) (bool, error)
}
