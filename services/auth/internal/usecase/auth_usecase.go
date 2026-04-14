package usecase

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"

	"github.com/base-go/base/pkg/apperror"
	pkgjwt "github.com/base-go/base/pkg/jwt"
	"github.com/base-go/base/services/auth/internal/domain"
	"github.com/base-go/base/services/auth/internal/platform/config"
)

// authUsecase implement domain.AuthService.
type authUsecase struct {
	userRepo          domain.UserRepository
	tokenRepo         domain.TokenRepository
	actionTokenRepo   domain.ActionTokenRepository
	lockoutPolicyRepo domain.LoginLockoutPolicyRepository
	emailSender       domain.EmailSender
	jwtManager        *pkgjwt.Manager
	tokenBlacklist    domain.TokenBlacklist
	passwordPolicy    config.PasswordPolicy
	securityConfig    config.SecurityConfig
	now               func() time.Time
}

// NewAuthUsecase tạo AuthService mới.
func NewAuthUsecase(
	userRepo domain.UserRepository,
	tokenRepo domain.TokenRepository,
	actionTokenRepo domain.ActionTokenRepository,
	lockoutPolicyRepo domain.LoginLockoutPolicyRepository,
	emailSender domain.EmailSender,
	jwtManager *pkgjwt.Manager,
	tokenBlacklist domain.TokenBlacklist,
	passwordPolicy config.PasswordPolicy,
	securityConfig config.SecurityConfig,
) domain.AuthService {
	return &authUsecase{
		userRepo:          userRepo,
		tokenRepo:         tokenRepo,
		actionTokenRepo:   actionTokenRepo,
		lockoutPolicyRepo: lockoutPolicyRepo,
		emailSender:       emailSender,
		jwtManager:        jwtManager,
		tokenBlacklist:    tokenBlacklist,
		passwordPolicy:    passwordPolicy,
		securityConfig:    securityConfig,
		now:               time.Now,
	}
}

func (uc *authUsecase) Register(ctx context.Context, email, password, fullName string) (*domain.User, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" {
		return nil, apperror.ValidationError("email is required")
	}

	if err := uc.validatePassword(password); err != nil {
		return nil, err
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, apperror.InternalError("failed to hash password", err)
	}

	var namePtr *string
	if fullName = strings.TrimSpace(fullName); fullName != "" {
		namePtr = &fullName
	}

	user := &domain.User{
		ID:                  uuid.New(),
		Email:               email,
		PasswordHash:        string(hashedPassword),
		FullName:            namePtr,
		Role:                domain.DefaultRole,
		EmailVerified:       false,
		IsActive:            true,
		FailedLoginAttempts: 0,
	}

	if err := uc.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}

	if err := uc.sendVerificationEmail(ctx, user); err != nil {
		return nil, apperror.ServiceUnavailable("failed to send verification email")
	}

	slog.InfoContext(ctx, "user registered and pending email verification", "user_id", user.ID, "email", user.Email)
	return user, nil
}

func (uc *authUsecase) Login(ctx context.Context, email, password string, meta domain.ClientMetadata) (*domain.LoginResult, error) {
	email = strings.TrimSpace(strings.ToLower(email))

	user, err := uc.userRepo.GetByEmail(ctx, email)
	if err != nil {
		var appErr *apperror.AppError
		if errors.As(err, &appErr) && appErr.Code == apperror.CodeNotFound {
			return nil, domain.ErrInvalidCredentials
		}
		return nil, err
	}

	if !user.IsActive {
		return nil, domain.ErrUserInactive
	}

	if !user.EmailVerified {
		return nil, domain.ErrEmailNotVerified
	}

	now := uc.now()
	if user.IsLocked(now) {
		return nil, domain.NewAccountLockedError(*user.LockedUntil)
	}

	if user.LockedUntil != nil {
		if err := uc.clearExpiredLockoutState(ctx, user, now); err != nil {
			return nil, err
		}
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, uc.handleFailedLogin(ctx, user)
	}

	if user.FailedLoginAttempts > 0 || user.LockedUntil != nil {
		user.FailedLoginAttempts = 0
		user.LockedUntil = nil
		if err := uc.userRepo.Update(ctx, user); err != nil {
			return nil, apperror.InternalError("failed to reset login failure state", err)
		}
	}

	if user.Is2FAEnabled {
		tempToken, err := uc.jwtManager.GenerateTemp2FAToken(user.ID.String(), user.Email)
		if err != nil {
			return nil, apperror.InternalError("failed to generate 2FA token", err)
		}
		slog.InfoContext(ctx, "user requires 2FA login", "user_id", user.ID)
		return &domain.LoginResult{
			User:        user,
			Requires2FA: true,
			TempToken:   tempToken,
		}, nil
	}

	if err := uc.tokenRepo.RevokeByUserID(ctx, user.ID); err != nil {
		slog.WarnContext(ctx, "failed to revoke old tokens on login", "error", err, "user_id", user.ID)
	}

	tokenPair, err := uc.generateTokenPair(ctx, user, meta)
	if err != nil {
		return nil, err
	}

	slog.InfoContext(ctx, "user logged in", "user_id", user.ID)
	return &domain.LoginResult{
		User:      user,
		TokenPair: tokenPair,
	}, nil
}

func (uc *authUsecase) Verify2FALogin(ctx context.Context, tempToken, code string, meta domain.ClientMetadata) (*domain.LoginResult, error) {
	claims, err := uc.jwtManager.ValidateToken(tempToken)
	if err != nil || claims.TokenType != pkgjwt.Temp2FAToken {
		return nil, apperror.Unauthorized("invalid or expired 2FA token")
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return nil, apperror.Unauthorized("invalid token user info")
	}

	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, apperror.Unauthorized("user not found")
	}

	if !user.IsActive || !user.Is2FAEnabled || user.TOTPSecret == nil {
		return nil, apperror.Unauthorized("invalid 2FA state")
	}

	now := uc.now()
	if user.IsLocked(now) {
		return nil, domain.NewAccountLockedError(*user.LockedUntil)
	}

	valid := totp.Validate(code, *user.TOTPSecret)
	if !valid {
		// Increment failed attempts as it's a security barrier
		user.FailedLoginAttempts++
		if policy, err := uc.GetLoginLockoutPolicy(ctx); err == nil {
			if user.FailedLoginAttempts >= policy.MaxFailedAttempts {
				lockTime := now.Add(policy.LockDuration)
				user.LockedUntil = &lockTime
			}
		}
		_ = uc.userRepo.Update(ctx, user)
		if user.LockedUntil != nil {
			return nil, domain.NewAccountLockedError(*user.LockedUntil)
		}
		return nil, apperror.Unauthorized("invalid 2FA code")
	}

	if user.FailedLoginAttempts > 0 || user.LockedUntil != nil {
		user.FailedLoginAttempts = 0
		user.LockedUntil = nil
		_ = uc.userRepo.Update(ctx, user)
	}

	if err := uc.tokenRepo.RevokeByUserID(ctx, user.ID); err != nil {
		slog.WarnContext(ctx, "failed to revoke old tokens on 2fa login", "error", err, "user_id", user.ID)
	}

	tokenPair, err := uc.generateTokenPair(ctx, user, meta)
	if err != nil {
		return nil, err
	}

	slog.InfoContext(ctx, "user logged in via 2FA", "user_id", user.ID)
	return &domain.LoginResult{
		User:      user,
		TokenPair: tokenPair,
	}, nil
}

func (uc *authUsecase) VerifyEmail(ctx context.Context, token string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return apperror.ValidationError("token is required")
	}

	actionToken, err := uc.actionTokenRepo.GetUsableByTokenHash(ctx, hashToken(token), domain.ActionTokenTypeEmailVerification)
	if err != nil {
		return uc.mapActionTokenError(err)
	}

	now := uc.now()
	if !actionToken.IsUsable(now) {
		return domain.ErrActionTokenInvalid
	}

	user, err := uc.userRepo.GetByID(ctx, actionToken.UserID)
	if err != nil {
		return err
	}

	if !user.EmailVerified {
		user.EmailVerified = true
		if err := uc.userRepo.Update(ctx, user); err != nil {
			return apperror.InternalError("failed to update user verification status", err)
		}
	}

	if err := uc.actionTokenRepo.MarkUsed(ctx, actionToken.ID, now); err != nil {
		return apperror.InternalError("failed to consume verification token", err)
	}

	slog.InfoContext(ctx, "user email verified", "user_id", user.ID)
	return nil
}

func (uc *authUsecase) ResendVerificationEmail(ctx context.Context, email string) error {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" {
		return apperror.ValidationError("email is required")
	}

	user, err := uc.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return uc.swallowNotFound(err)
	}

	if user.EmailVerified || !user.IsActive {
		return nil
	}

	if err := uc.sendVerificationEmail(ctx, user); err != nil {
		slog.ErrorContext(ctx, "failed to resend verification email", "error", err, "user_id", user.ID)
	}

	return nil
}

func (uc *authUsecase) ForgotPassword(ctx context.Context, email string) error {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" {
		return apperror.ValidationError("email is required")
	}

	user, err := uc.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return uc.swallowNotFound(err)
	}

	if !user.IsActive || !user.EmailVerified {
		return nil
	}

	if err := uc.sendPasswordResetEmail(ctx, user); err != nil {
		slog.ErrorContext(ctx, "failed to send password reset email", "error", err, "user_id", user.ID)
	}

	return nil
}

func (uc *authUsecase) ResetPassword(ctx context.Context, token, newPassword string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return apperror.ValidationError("token is required")
	}

	if err := uc.validatePassword(newPassword); err != nil {
		return err
	}

	actionToken, err := uc.actionTokenRepo.GetUsableByTokenHash(ctx, hashToken(token), domain.ActionTokenTypePasswordReset)
	if err != nil {
		return uc.mapActionTokenError(err)
	}

	now := uc.now()
	if !actionToken.IsUsable(now) {
		return domain.ErrActionTokenInvalid
	}

	if err := uc.actionTokenRepo.MarkUsed(ctx, actionToken.ID, now); err != nil {
		return apperror.InternalError("failed to consume password reset token", err)
	}

	user, err := uc.userRepo.GetByID(ctx, actionToken.UserID)
	if err != nil {
		return err
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return apperror.InternalError("failed to hash password", err)
	}

	user.PasswordHash = string(hashedPassword)
	user.FailedLoginAttempts = 0
	user.LockedUntil = nil
	if err := uc.userRepo.Update(ctx, user); err != nil {
		return apperror.InternalError("failed to update password", err)
	}

	if err := uc.actionTokenRepo.RevokeByUserAndType(ctx, user.ID, domain.ActionTokenTypePasswordReset); err != nil {
		slog.WarnContext(ctx, "failed to revoke password reset tokens", "error", err, "user_id", user.ID)
	}

	if err := uc.tokenRepo.RevokeByUserID(ctx, user.ID); err != nil {
		slog.WarnContext(ctx, "failed to revoke refresh tokens after password reset", "error", err, "user_id", user.ID)
	}

	slog.InfoContext(ctx, "password reset completed", "user_id", user.ID)
	return nil
}

func (uc *authUsecase) RefreshToken(ctx context.Context, refreshTokenStr string, meta domain.ClientMetadata) (*domain.TokenPair, error) {
	claims, err := uc.jwtManager.ValidateToken(refreshTokenStr)
	if err != nil {
		return nil, domain.ErrTokenInvalid
	}

	if claims.TokenType != pkgjwt.RefreshToken {
		return nil, domain.ErrTokenInvalid
	}

	tokenHash := hashToken(refreshTokenStr)
	storedToken, err := uc.tokenRepo.GetByTokenHash(ctx, tokenHash)
	if err != nil {
		return nil, domain.ErrTokenInvalid
	}

	if !storedToken.IsUsable() {
		return nil, domain.ErrTokenRevoked
	}

	if err := uc.tokenRepo.RevokeByIDAndUserID(ctx, storedToken.ID, storedToken.UserID); err != nil {
		slog.WarnContext(ctx, "failed to revoke old refresh token", "error", err)
	}

	user, err := uc.userRepo.GetByID(ctx, storedToken.UserID)
	if err != nil {
		return nil, err
	}

	if !user.IsActive {
		return nil, domain.ErrUserInactive
	}

	if !user.EmailVerified {
		return nil, domain.ErrEmailNotVerified
	}

	if user.IsLocked(uc.now()) {
		return nil, domain.NewAccountLockedError(*user.LockedUntil)
	}

	tokenPair, err := uc.generateTokenPair(ctx, user, meta)
	if err != nil {
		return nil, err
	}

	slog.InfoContext(ctx, "token refreshed", "user_id", user.ID)
	return tokenPair, nil
}

func (uc *authUsecase) Logout(ctx context.Context, userID uuid.UUID, accessTokenStr string) error {
	if accessTokenStr != "" {
		claims, err := uc.jwtManager.ValidateToken(accessTokenStr)
		if err == nil && claims.TokenType == pkgjwt.AccessToken {
			if err := uc.tokenBlacklist.BlacklistToken(ctx, claims.ID, claims.ExpiresAt.Time); err != nil {
				slog.WarnContext(ctx, "failed to blacklist access token", "error", err, "user_id", userID)
			}
		}
	}

	if err := uc.tokenRepo.RevokeByUserID(ctx, userID); err != nil {
		return apperror.InternalError("failed to revoke tokens", err)
	}

	slog.InfoContext(ctx, "user logged out", "user_id", userID)
	return nil
}

func (uc *authUsecase) GetProfile(ctx context.Context, userID uuid.UUID) (*domain.User, error) {
	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (uc *authUsecase) GetSessions(ctx context.Context, userID uuid.UUID) ([]*domain.RefreshToken, error) {
	sessions, err := uc.tokenRepo.ListByUserID(ctx, userID)
	if err != nil {
		return nil, apperror.InternalError("failed to get active sessions", err)
	}
	return sessions, nil
}

func (uc *authUsecase) DeleteSession(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID) error {
	if err := uc.tokenRepo.RevokeByIDAndUserID(ctx, sessionID, userID); err != nil {
		return apperror.InternalError("failed to delete session", err)
	}
	slog.InfoContext(ctx, "user session deleted", "user_id", userID, "session_id", sessionID)
	return nil
}

func (uc *authUsecase) ValidateToken(ctx context.Context, accessToken string) (*domain.User, error) {
	claims, err := uc.jwtManager.ValidateToken(accessToken)
	if err != nil {
		return nil, domain.ErrTokenInvalid
	}

	if claims.TokenType != pkgjwt.AccessToken {
		return nil, domain.ErrTokenInvalid
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return nil, domain.ErrTokenInvalid
	}

	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if !user.IsActive {
		return nil, domain.ErrUserInactive
	}

	if !user.EmailVerified {
		return nil, domain.ErrEmailNotVerified
	}

	if user.IsLocked(uc.now()) {
		return nil, domain.NewAccountLockedError(*user.LockedUntil)
	}

	return user, nil
}

func (uc *authUsecase) GetLoginLockoutPolicy(ctx context.Context) (*domain.LoginLockoutPolicy, error) {
	return uc.getEffectiveLockoutPolicy(ctx)
}

func (uc *authUsecase) UpdateLoginLockoutPolicy(ctx context.Context, maxFailedAttempts int, lockDuration time.Duration) (*domain.LoginLockoutPolicy, error) {
	if maxFailedAttempts <= 0 {
		return nil, apperror.ValidationError("max_failed_attempts must be greater than 0")
	}
	if lockDuration <= 0 {
		return nil, apperror.ValidationError("lock_duration must be greater than 0")
	}

	policy := &domain.LoginLockoutPolicy{
		MaxFailedAttempts: maxFailedAttempts,
		LockDuration:      lockDuration,
	}

	if err := uc.lockoutPolicyRepo.Upsert(ctx, policy); err != nil {
		return nil, apperror.InternalError("failed to update login lockout policy", err)
	}

	slog.InfoContext(ctx, "login lockout policy updated", "max_failed_attempts", maxFailedAttempts, "lock_duration", lockDuration.String())
	return policy, nil
}

// generateTokenPair phát hành cặp access + refresh token và lưu refresh token vào DB.
func (uc *authUsecase) generateTokenPair(ctx context.Context, user *domain.User, meta domain.ClientMetadata) (*domain.TokenPair, error) {
	accessToken, err := uc.jwtManager.GenerateAccessToken(user.ID.String(), user.Email, user.Role)
	if err != nil {
		return nil, apperror.InternalError("failed to generate access token", err)
	}

	refreshToken, err := uc.jwtManager.GenerateRefreshToken(user.ID.String(), user.Email, user.Role)
	if err != nil {
		return nil, apperror.InternalError("failed to generate refresh token", err)
	}

	tokenHash := hashToken(refreshToken)
	storedToken := &domain.RefreshToken{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: tokenHash,
		ExpiresAt: uc.now().Add(uc.jwtManager.GetRefreshTokenTTL()),
		Revoked:   false,
		IP:        meta.IP,
		UserAgent: meta.UserAgent,
		DeviceID:  meta.DeviceID,
	}

	if err := uc.tokenRepo.Create(ctx, storedToken); err != nil {
		return nil, apperror.InternalError("failed to store refresh token", err)
	}

	return &domain.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (uc *authUsecase) sendVerificationEmail(ctx context.Context, user *domain.User) error {
	rawToken, expiresAt, err := uc.issueActionToken(ctx, user.ID, domain.ActionTokenTypeEmailVerification, uc.securityConfig.EmailVerificationTokenTTL)
	if err != nil {
		return err
	}

	if err := uc.emailSender.SendVerificationEmail(ctx, user, rawToken, expiresAt); err != nil {
		return err
	}

	return nil
}

func (uc *authUsecase) sendPasswordResetEmail(ctx context.Context, user *domain.User) error {
	rawToken, expiresAt, err := uc.issueActionToken(ctx, user.ID, domain.ActionTokenTypePasswordReset, uc.securityConfig.PasswordResetTokenTTL)
	if err != nil {
		return err
	}

	if err := uc.emailSender.SendPasswordResetEmail(ctx, user, rawToken, expiresAt); err != nil {
		return err
	}

	return nil
}

func (uc *authUsecase) issueActionToken(ctx context.Context, userID uuid.UUID, tokenType domain.ActionTokenType, ttl time.Duration) (string, time.Time, error) {
	if err := uc.actionTokenRepo.RevokeByUserAndType(ctx, userID, tokenType); err != nil {
		return "", time.Time{}, apperror.InternalError("failed to revoke old action tokens", err)
	}

	rawToken, err := generateSecureToken()
	if err != nil {
		return "", time.Time{}, apperror.InternalError("failed to generate secure token", err)
	}

	expiresAt := uc.now().Add(ttl)
	actionToken := &domain.ActionToken{
		ID:        uuid.New(),
		UserID:    userID,
		Type:      tokenType,
		TokenHash: hashToken(rawToken),
		ExpiresAt: expiresAt,
		Revoked:   false,
	}

	if err := uc.actionTokenRepo.Create(ctx, actionToken); err != nil {
		return "", time.Time{}, apperror.InternalError("failed to store action token", err)
	}

	return rawToken, expiresAt, nil
}

func (uc *authUsecase) getEffectiveLockoutPolicy(ctx context.Context) (*domain.LoginLockoutPolicy, error) {
	policy, err := uc.lockoutPolicyRepo.Get(ctx)
	if err == nil {
		return policy, nil
	}

	var appErr *apperror.AppError
	if errors.As(err, &appErr) && appErr.Code == apperror.CodeNotFound {
		return &domain.LoginLockoutPolicy{
			MaxFailedAttempts: uc.securityConfig.LockoutPolicy.MaxFailedAttempts,
			LockDuration:      uc.securityConfig.LockoutPolicy.LockDuration,
		}, nil
	}

	return nil, apperror.InternalError("failed to load login lockout policy", err)
}

func (uc *authUsecase) handleFailedLogin(ctx context.Context, user *domain.User) error {
	policy, err := uc.getEffectiveLockoutPolicy(ctx)
	if err != nil {
		return err
	}

	if policy.Disabled() {
		return domain.ErrInvalidCredentials
	}

	user.FailedLoginAttempts++
	if user.FailedLoginAttempts >= policy.MaxFailedAttempts {
		lockUntil := uc.now().Add(policy.LockDuration)
		user.FailedLoginAttempts = 0
		user.LockedUntil = &lockUntil
		if err := uc.userRepo.Update(ctx, user); err != nil {
			return apperror.InternalError("failed to persist account lock state", err)
		}
		return domain.NewAccountLockedError(lockUntil)
	}

	if err := uc.userRepo.Update(ctx, user); err != nil {
		return apperror.InternalError("failed to persist login failure state", err)
	}

	return domain.ErrInvalidCredentials
}

func (uc *authUsecase) clearExpiredLockoutState(ctx context.Context, user *domain.User, now time.Time) error {
	if user.LockedUntil == nil || now.Before(*user.LockedUntil) {
		return nil
	}

	user.LockedUntil = nil
	user.FailedLoginAttempts = 0
	if err := uc.userRepo.Update(ctx, user); err != nil {
		return apperror.InternalError("failed to clear expired lockout state", err)
	}

	return nil
}

func (uc *authUsecase) mapActionTokenError(err error) error {
	var appErr *apperror.AppError
	if errors.As(err, &appErr) && appErr.Code == apperror.CodeNotFound {
		return domain.ErrActionTokenInvalid
	}
	return err
}

func (uc *authUsecase) swallowNotFound(err error) error {
	var appErr *apperror.AppError
	if errors.As(err, &appErr) && appErr.Code == apperror.CodeNotFound {
		return nil
	}
	return err
}

// validatePassword kiểm tra password theo policy configurable.
func (uc *authUsecase) validatePassword(password string) error {
	p := uc.passwordPolicy

	if len(password) < p.MinLength {
		return apperror.ValidationError(
			fmt.Sprintf("password must be at least %d characters", p.MinLength),
		)
	}

	if p.RequireUppercase && !containsFunc(password, unicode.IsUpper) {
		return apperror.ValidationError("password must contain at least one uppercase letter")
	}

	if p.RequireLowercase && !containsFunc(password, unicode.IsLower) {
		return apperror.ValidationError("password must contain at least one lowercase letter")
	}

	if p.RequireDigit && !containsFunc(password, unicode.IsDigit) {
		return apperror.ValidationError("password must contain at least one digit")
	}

	if p.RequireSpecial && !containsFunc(password, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	}) {
		return apperror.ValidationError("password must contain at least one special character")
	}

	return nil
}

// hashToken tạo SHA-256 hash của token để lưu vào DB.
func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

func generateSecureToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// containsFunc kiểm tra string có chứa ít nhất 1 rune thoả điều kiện.
func containsFunc(s string, f func(rune) bool) bool {
	for _, r := range s {
		if f(r) {
			return true
		}
	}
	return false
}

// Setup2FA tạo TOTP secret cho user nếu chưa bật 2FA.
func (uc *authUsecase) Setup2FA(ctx context.Context, userID uuid.UUID) (*domain.Setup2FAResponse, error) {
	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("userRepo.GetByID: %w", err)
	}

	if user.Is2FAEnabled {
		return nil, apperror.ValidationError("2FA is already enabled")
	}

	// Tạo TOTP secret
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "lamld-io",
		AccountName: user.Email,
	})
	if err != nil {
		return nil, fmt.Errorf("totp.Generate: %w", err)
	}

	secret := key.Secret()
	user.TOTPSecret = &secret

	// Cập nhật lại user (lưu secret nhưng chưa enable 2FA)
	if err := uc.userRepo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("userRepo.Update: %w", err)
	}

	return &domain.Setup2FAResponse{
		Secret:    key.Secret(),
		SecretURL: key.URL(),
	}, nil
}

// Verify2FASetup kiểm tra OTP hợp lệ và bặt cờ 2FA cho user.
func (uc *authUsecase) Verify2FASetup(ctx context.Context, userID uuid.UUID, code string) error {
	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("userRepo.GetByID: %w", err)
	}

	if user.Is2FAEnabled {
		return apperror.ValidationError("2FA is already enabled")
	}

	if user.TOTPSecret == nil {
		return apperror.ValidationError("2FA setup has not been initialized")
	}

	// Xác thực mã OTP
	valid := totp.Validate(code, *user.TOTPSecret)
	if !valid {
		return apperror.ValidationError("invalid OTP code")
	}

	// Kích hoạt 2FA
	user.Is2FAEnabled = true

	if err := uc.userRepo.Update(ctx, user); err != nil {
		return fmt.Errorf("userRepo.Update: %w", err)
	}

	return nil
}
