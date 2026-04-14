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

// ToAuthResponse kết hợp user + token pair → dto.AuthResponse.
func ToAuthResponse(user *domain.User, tokenPair *domain.TokenPair, expiresInSec int64) dto.AuthResponse {
	return dto.AuthResponse{
		User: UserToResponse(user),
		Token: dto.TokenResponse{
			AccessToken:  tokenPair.AccessToken,
			RefreshToken: tokenPair.RefreshToken,
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
