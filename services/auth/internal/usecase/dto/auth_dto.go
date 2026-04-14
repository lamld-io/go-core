package dto

import "time"

// --- Request DTOs ---

// RegisterRequest chứa dữ liệu đăng ký tài khoản.
type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
	FullName string `json:"full_name"`
}

// LoginRequest chứa dữ liệu đăng nhập.
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// RefreshRequest chứa refresh token để làm mới access token.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// VerifyEmailRequest chứa token xác thực email.
type VerifyEmailRequest struct {
	Token string `json:"token" binding:"required"`
}

// ResendVerificationRequest chứa email cần gửi lại mail xác thực.
type ResendVerificationRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// ForgotPasswordRequest chứa email yêu cầu reset password.
type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// ResetPasswordRequest chứa token reset và mật khẩu mới.
type ResetPasswordRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required"`
}

// UpdateLockoutPolicyRequest chứa policy lockout mới.
type UpdateLockoutPolicyRequest struct {
	MaxFailedAttempts   int   `json:"max_failed_attempts" binding:"required,min=1"`
	LockDurationSeconds int64 `json:"lock_duration_seconds" binding:"required,min=1"`
}

// --- Response DTOs ---

// UserResponse chứa thông tin user trả về cho client.
// Không bao giờ trả về password_hash.
type UserResponse struct {
	ID            string    `json:"id"`
	Email         string    `json:"email"`
	FullName      *string   `json:"full_name,omitempty"`
	Role          string    `json:"role"`
	EmailVerified bool      `json:"email_verified"`
	IsActive      bool      `json:"is_active"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// TokenResponse chứa cặp access/refresh token.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"` // seconds
}

// AuthResponse kết hợp thông tin user và token — dùng cho register & login.
type AuthResponse struct {
	User  UserResponse  `json:"user"`
	Token TokenResponse `json:"token"`
}

// RegisterResponse trả về trạng thái đăng ký chờ xác thực email.
type RegisterResponse struct {
	User                      UserResponse `json:"user"`
	RequiresEmailVerification bool         `json:"requires_email_verification"`
	Message                   string       `json:"message"`
}

// ProfileResponse trả về thông tin profile user hiện tại.
type ProfileResponse struct {
	User UserResponse `json:"user"`
}

// ValidateResponse trả về kết quả validate token (cho internal call).
type ValidateResponse struct {
	Valid bool          `json:"valid"`
	User  *UserResponse `json:"user,omitempty"`
}

// MessageResponse dùng cho các endpoint chỉ trả thông điệp.
type MessageResponse struct {
	Message string `json:"message"`
}

// LockoutPolicyResponse biểu diễn policy khoá tài khoản.
type LockoutPolicyResponse struct {
	MaxFailedAttempts   int   `json:"max_failed_attempts"`
	LockDurationSeconds int64 `json:"lock_duration_seconds"`
}

// SessionResponse biểu diễn thông tin thiết bị / phiên đăng nhập đang hoạt động.
type SessionResponse struct {
	ID        string    `json:"id"`
	IP        string    `json:"ip"`
	UserAgent string    `json:"user_agent"`
	DeviceID  string    `json:"device_id"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}
