package presenter

import (
	"github.com/base-go/base/services/auth/internal/domain"
	"github.com/base-go/base/services/auth/internal/usecase/dto"
)

// UserToResponse chuyển domain.User → dto.UserResponse.
// Đảm bảo không bao giờ trả password_hash ra ngoài.
func UserToResponse(user *domain.User) dto.UserResponse {
	return dto.UserResponse{
		ID:            user.ID.String(),
		Email:         user.Email,
		FullName:      user.FullName,
		Role:          user.Role,
		EmailVerified: user.EmailVerified,
		IsActive:      user.IsActive,
		CreatedAt:     user.CreatedAt,
		UpdatedAt:     user.UpdatedAt,
	}
}

// ToLoginResponse chuyển đổi LoginResult thành dto.LoginResponse.
func ToLoginResponse(result *domain.LoginResult, expiresInSec int64) dto.LoginResponse {
	if result.Requires2FA {
		return dto.LoginResponse{
			Requires2FA: true,
			TempToken:   result.TempToken,
			Message:     "2FA verification required",
		}
	}

	userResp := UserToResponse(result.User)
	return dto.LoginResponse{
		Requires2FA: false,
		User:        &userResp,
		Token: &dto.TokenResponse{
			AccessToken:  result.TokenPair.AccessToken,
			RefreshToken: result.TokenPair.RefreshToken,
			TokenType:    "Bearer",
			ExpiresIn:    expiresInSec,
		},
	}
}

// ToRegisterResponse map user sang response đăng ký.
func ToRegisterResponse(user *domain.User) dto.RegisterResponse {
	return dto.RegisterResponse{
		User:                      UserToResponse(user),
		RequiresEmailVerification: true,
		Message:                   "registration successful, please verify your email before logging in",
	}
}

// ToSessionResponse chuyển đổi RefreshToken thành DTO công khai (loại bỏ token hash).
func ToSessionResponse(session *domain.RefreshToken) dto.SessionResponse {
	return dto.SessionResponse{
		ID:        session.ID.String(),
		IP:        session.IP,
		UserAgent: session.UserAgent,
		DeviceID:  session.DeviceID,
		CreatedAt: session.CreatedAt,
		ExpiresAt: session.ExpiresAt,
	}
}
