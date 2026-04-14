package handler

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/base-go/base/pkg/apperror"
	"github.com/base-go/base/pkg/response"
	"github.com/base-go/base/services/auth/internal/delivery/http/middleware"
	"github.com/base-go/base/services/auth/internal/delivery/http/presenter"
	"github.com/base-go/base/services/auth/internal/domain"
	"github.com/base-go/base/services/auth/internal/usecase/dto"
)

// AuthHandler xử lý HTTP request cho các endpoint xác thực.
type AuthHandler struct {
	authService    domain.AuthService
	accessTokenTTL int64
}

// NewAuthHandler tạo AuthHandler mới.
func NewAuthHandler(authService domain.AuthService, accessTokenTTLSec int64) *AuthHandler {
	return &AuthHandler{
		authService:    authService,
		accessTokenTTL: accessTokenTTLSec,
	}
}

// Register xử lý POST /api/v1/auth/register.
func (h *AuthHandler) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.ValidationError(err.Error()))
		return
	}

	user, err := h.authService.Register(c.Request.Context(), req.Email, req.Password, req.FullName)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Created(c, presenter.ToRegisterResponse(user))
}

// Login xử lý POST /api/v1/auth/login.
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.ValidationError(err.Error()))
		return
	}

	user, tokenPair, err := h.authService.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, presenter.ToAuthResponse(user, tokenPair, h.accessTokenTTL))
}

// VerifyEmail xử lý POST /api/v1/auth/verify-email.
func (h *AuthHandler) VerifyEmail(c *gin.Context) {
	var req dto.VerifyEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.ValidationError(err.Error()))
		return
	}

	if err := h.authService.VerifyEmail(c.Request.Context(), req.Token); err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, dto.MessageResponse{Message: "email verified successfully"})
}

// ResendVerificationEmail xử lý POST /api/v1/auth/resend-verification-email.
func (h *AuthHandler) ResendVerificationEmail(c *gin.Context) {
	var req dto.ResendVerificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.ValidationError(err.Error()))
		return
	}

	if err := h.authService.ResendVerificationEmail(c.Request.Context(), req.Email); err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, dto.MessageResponse{Message: "if the account exists and is pending verification, a verification email will be sent"})
}

// ForgotPassword xử lý POST /api/v1/auth/forgot-password.
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req dto.ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.ValidationError(err.Error()))
		return
	}

	if err := h.authService.ForgotPassword(c.Request.Context(), req.Email); err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, dto.MessageResponse{Message: "if the account exists, password reset instructions will be sent"})
}

// ResetPassword xử lý POST /api/v1/auth/reset-password.
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req dto.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.ValidationError(err.Error()))
		return
	}

	if err := h.authService.ResetPassword(c.Request.Context(), req.Token, req.NewPassword); err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, dto.MessageResponse{Message: "password reset successfully"})
}

// RefreshToken xử lý POST /api/v1/auth/refresh.
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req dto.RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.ValidationError(err.Error()))
		return
	}

	tokenPair, err := h.authService.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, dto.TokenResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    h.accessTokenTTL,
	})
}

// Logout xử lý POST /api/v1/auth/logout.
func (h *AuthHandler) Logout(c *gin.Context) {
	userIDStr, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		response.Error(c, apperror.Unauthorized("user not authenticated"))
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		response.Error(c, apperror.Unauthorized("invalid user id in token"))
		return
	}

	if err := h.authService.Logout(c.Request.Context(), userID); err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, dto.MessageResponse{Message: "logged out successfully"})
}

// GetProfile xử lý GET /api/v1/auth/profile.
func (h *AuthHandler) GetProfile(c *gin.Context) {
	userIDStr, exists := middleware.GetUserIDFromContext(c)
	if !exists {
		response.Error(c, apperror.Unauthorized("user not authenticated"))
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		response.Error(c, apperror.Unauthorized("invalid user id in token"))
		return
	}

	user, err := h.authService.GetProfile(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, dto.ProfileResponse{User: presenter.UserToResponse(user)})
}

// ValidateToken xử lý GET /api/v1/auth/validate.
func (h *AuthHandler) ValidateToken(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		token = c.GetHeader("Authorization")
		if len(token) > 7 && token[:7] == "Bearer " {
			token = token[7:]
		}
	}

	if token == "" {
		response.OK(c, dto.ValidateResponse{Valid: false})
		return
	}

	user, err := h.authService.ValidateToken(c.Request.Context(), token)
	if err != nil {
		slog.DebugContext(c.Request.Context(), "token validation failed", "error", err)
		response.OK(c, dto.ValidateResponse{Valid: false})
		return
	}

	userResp := presenter.UserToResponse(user)
	response.OK(c, dto.ValidateResponse{
		Valid: true,
		User:  &userResp,
	})
}

// GetLoginLockoutPolicy xử lý GET /api/v1/auth/security/lockout-policy.
func (h *AuthHandler) GetLoginLockoutPolicy(c *gin.Context) {
	if !h.isAdmin(c) {
		response.Error(c, domain.ErrForbidden)
		return
	}

	policy, err := h.authService.GetLoginLockoutPolicy(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, dto.LockoutPolicyResponse{
		MaxFailedAttempts:   policy.MaxFailedAttempts,
		LockDurationSeconds: int64(policy.LockDuration / time.Second),
	})
}

// UpdateLoginLockoutPolicy xử lý PUT /api/v1/auth/security/lockout-policy.
func (h *AuthHandler) UpdateLoginLockoutPolicy(c *gin.Context) {
	if !h.isAdmin(c) {
		response.Error(c, domain.ErrForbidden)
		return
	}

	var req dto.UpdateLockoutPolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, apperror.ValidationError(err.Error()))
		return
	}

	policy, err := h.authService.UpdateLoginLockoutPolicy(
		c.Request.Context(),
		req.MaxFailedAttempts,
		time.Duration(req.LockDurationSeconds)*time.Second,
	)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, dto.LockoutPolicyResponse{
		MaxFailedAttempts:   policy.MaxFailedAttempts,
		LockDurationSeconds: int64(policy.LockDuration / time.Second),
	})
}

// HealthCheck xử lý GET /health.
func (h *AuthHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"service": "auth-service",
	})
}

func (h *AuthHandler) isAdmin(c *gin.Context) bool {
	role, exists := middleware.GetRoleFromContext(c)
	return exists && role == domain.RoleAdmin
}
