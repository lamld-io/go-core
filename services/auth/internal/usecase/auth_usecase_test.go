package usecase_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pkgjwt "github.com/base-go/base/pkg/jwt"
	"github.com/base-go/base/services/auth/internal/domain"
	"github.com/base-go/base/services/auth/internal/platform/config"
	"github.com/base-go/base/services/auth/internal/usecase"
)

type mockUserRepo struct {
	mu    sync.RWMutex
	users map[string]*domain.User
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{users: make(map[string]*domain.User)}
}

func (r *mockUserRepo) Create(_ context.Context, user *domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.users[user.Email]; exists {
		return domain.ErrUserAlreadyExists
	}

	cloned := cloneUser(user)
	cloned.CreatedAt = time.Now()
	cloned.UpdatedAt = cloned.CreatedAt
	r.users[user.Email] = cloned
	user.CreatedAt = cloned.CreatedAt
	user.UpdatedAt = cloned.UpdatedAt
	return nil
}

func (r *mockUserRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, user := range r.users {
		if user.ID == id {
			return cloneUser(user), nil
		}
	}

	return nil, domain.ErrUserNotFound
}

func (r *mockUserRepo) GetByEmail(_ context.Context, email string) (*domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	user, ok := r.users[email]
	if !ok {
		return nil, domain.ErrUserNotFound
	}

	return cloneUser(user), nil
}

func (r *mockUserRepo) Update(_ context.Context, user *domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for email, existing := range r.users {
		if existing.ID == user.ID {
			cloned := cloneUser(user)
			cloned.CreatedAt = existing.CreatedAt
			cloned.UpdatedAt = time.Now()
			r.users[email] = cloned
			user.CreatedAt = cloned.CreatedAt
			user.UpdatedAt = cloned.UpdatedAt
			return nil
		}
	}

	return domain.ErrUserNotFound
}

type mockTokenRepo struct {
	mu     sync.RWMutex
	tokens map[string]*domain.RefreshToken
}

func newMockTokenRepo() *mockTokenRepo {
	return &mockTokenRepo{tokens: make(map[string]*domain.RefreshToken)}
}

func (r *mockTokenRepo) Create(_ context.Context, token *domain.RefreshToken) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	cloned := *token
	cloned.CreatedAt = time.Now()
	r.tokens[token.TokenHash] = &cloned
	token.CreatedAt = cloned.CreatedAt
	return nil
}

func (r *mockTokenRepo) GetByTokenHash(_ context.Context, hash string) (*domain.RefreshToken, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	token, ok := r.tokens[hash]
	if !ok {
		return nil, domain.ErrTokenNotFound
	}

	cloned := *token
	return &cloned, nil
}

func (r *mockTokenRepo) RevokeByUserID(_ context.Context, userID uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, token := range r.tokens {
		if token.UserID == userID {
			token.Revoked = true
		}
	}

	return nil
}

func (r *mockTokenRepo) RevokeByIDAndUserID(_ context.Context, id, userID uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, token := range r.tokens {
		if token.ID == id && token.UserID == userID {
			token.Revoked = true
		}
	}

	return nil
}

func (r *mockTokenRepo) ListByUserID(_ context.Context, userID uuid.UUID) ([]*domain.RefreshToken, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*domain.RefreshToken
	for _, token := range r.tokens {
		if token.UserID == userID && !token.Revoked {
			cloned := *token
			result = append(result, &cloned)
		}
	}
	return result, nil
}

func (r *mockTokenRepo) DeleteExpired(_ context.Context) error {
	return nil
}

type mockActionTokenRepo struct {
	mu     sync.RWMutex
	tokens map[string]*domain.ActionToken
}

func newMockActionTokenRepo() *mockActionTokenRepo {
	return &mockActionTokenRepo{tokens: make(map[string]*domain.ActionToken)}
}

func (r *mockActionTokenRepo) Create(_ context.Context, token *domain.ActionToken) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	cloned := *token
	cloned.CreatedAt = time.Now()
	r.tokens[token.TokenHash] = &cloned
	token.CreatedAt = cloned.CreatedAt
	return nil
}

func (r *mockActionTokenRepo) GetUsableByTokenHash(_ context.Context, tokenHash string, tokenType domain.ActionTokenType) (*domain.ActionToken, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	token, ok := r.tokens[tokenHash]
	if !ok || token.Type != tokenType || token.Revoked || token.UsedAt != nil {
		return nil, domain.ErrActionTokenNotFound
	}

	cloned := *token
	return &cloned, nil
}

func (r *mockActionTokenRepo) MarkUsed(_ context.Context, id uuid.UUID, usedAt time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, token := range r.tokens {
		if token.ID == id {
			ts := usedAt
			token.UsedAt = &ts
			return nil
		}
	}

	return domain.ErrActionTokenNotFound
}

func (r *mockActionTokenRepo) RevokeByUserAndType(_ context.Context, userID uuid.UUID, tokenType domain.ActionTokenType) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, token := range r.tokens {
		if token.UserID == userID && token.Type == tokenType {
			token.Revoked = true
		}
	}

	return nil
}

func (r *mockActionTokenRepo) DeleteExpired(_ context.Context) error {
	return nil
}

type mockLockoutPolicyRepo struct {
	mu     sync.RWMutex
	policy *domain.LoginLockoutPolicy
}

func newMockLockoutPolicyRepo() *mockLockoutPolicyRepo {
	return &mockLockoutPolicyRepo{}
}

func (r *mockLockoutPolicyRepo) Get(_ context.Context) (*domain.LoginLockoutPolicy, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.policy == nil {
		return nil, domain.ErrLoginLockoutPolicyNotFound
	}

	cloned := *r.policy
	return &cloned, nil
}

func (r *mockLockoutPolicyRepo) Upsert(_ context.Context, policy *domain.LoginLockoutPolicy) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	cloned := *policy
	r.policy = &cloned
	return nil
}

type mockEmailSender struct {
	mu                 sync.RWMutex
	verificationTokens map[string]string
	resetTokens        map[string]string
}

func newMockEmailSender() *mockEmailSender {
	return &mockEmailSender{
		verificationTokens: make(map[string]string),
		resetTokens:        make(map[string]string),
	}
}

func (m *mockEmailSender) SendVerificationEmail(_ context.Context, user *domain.User, token string, _ time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.verificationTokens[user.Email] = token
	return nil
}

func (m *mockEmailSender) SendPasswordResetEmail(_ context.Context, user *domain.User, token string, _ time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.resetTokens[user.Email] = token
	return nil
}

func (m *mockEmailSender) VerificationToken(email string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.verificationTokens[email]
}

func (m *mockEmailSender) ResetToken(email string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.resetTokens[email]
}

type mockTokenBlacklist struct {
	mu        sync.RWMutex
	blacklist map[string]time.Time
}

func newMockTokenBlacklist() *mockTokenBlacklist {
	return &mockTokenBlacklist{blacklist: make(map[string]time.Time)}
}

func (m *mockTokenBlacklist) BlacklistToken(_ context.Context, tokenID string, expiresAt time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.blacklist[tokenID] = expiresAt
	return nil
}

func (m *mockTokenBlacklist) IsBlacklisted(_ context.Context, tokenID string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.blacklist[tokenID]
	// Có thể thêm check time.Now().After(expiresAt)
	return ok, nil
}

func setupTestUsecase(t *testing.T) (domain.AuthService, *mockUserRepo, *mockTokenRepo, *mockActionTokenRepo, *mockLockoutPolicyRepo, *mockEmailSender) {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	dir := t.TempDir()
	privPath := filepath.Join(dir, "private.pem")
	pubPath := filepath.Join(dir, "public.pem")

	privBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(privPath, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privBytes}), 0600))

	pubBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(pubPath, pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubBytes}), 0600))

	jwtManager, err := pkgjwt.NewManager(pkgjwt.Config{
		PrivateKeyPath:  privPath,
		PublicKeyPath:   pubPath,
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: time.Hour,
		Issuer:          "test",
	})
	require.NoError(t, err)

	userRepo := newMockUserRepo()
	tokenRepo := newMockTokenRepo()
	actionTokenRepo := newMockActionTokenRepo()
	lockoutPolicyRepo := newMockLockoutPolicyRepo()
	emailSender := newMockEmailSender()
	tokenBlacklist := newMockTokenBlacklist()

	svc := usecase.NewAuthUsecase(
		userRepo,
		tokenRepo,
		actionTokenRepo,
		lockoutPolicyRepo,
		emailSender,
		jwtManager,
		tokenBlacklist,
		config.PasswordPolicy{MinLength: 8},
		config.SecurityConfig{
			EmailVerificationTokenTTL: time.Hour,
			PasswordResetTokenTTL:     time.Hour,
			LockoutPolicy: config.LockoutPolicyConfig{
				MaxFailedAttempts: 3,
				LockDuration:      15 * time.Minute,
			},
		},
	)

	return svc, userRepo, tokenRepo, actionTokenRepo, lockoutPolicyRepo, emailSender
}

func TestRegisterRequiresEmailVerification(t *testing.T) {
	svc, _, _, _, _, emailSender := setupTestUsecase(t)
	ctx := context.Background()

	user, err := svc.Register(ctx, "user@example.com", "password123", "Test User")
	require.NoError(t, err)
	assert.False(t, user.EmailVerified)
	assert.NotEmpty(t, emailSender.VerificationToken("user@example.com"))

	_, err = svc.Login(ctx, "user@example.com", "password123", domain.ClientMetadata{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not verified")
}

func TestVerifyEmailThenLogin(t *testing.T) {
	svc, _, _, _, _, emailSender := setupTestUsecase(t)
	ctx := context.Background()

	_, err := svc.Register(ctx, "verify@example.com", "password123", "Verify User")
	require.NoError(t, err)

	token := emailSender.VerificationToken("verify@example.com")
	require.NotEmpty(t, token)
	require.NoError(t, svc.VerifyEmail(ctx, token))

	res, err := svc.Login(ctx, "verify@example.com", "password123", domain.ClientMetadata{})
	require.NoError(t, err)
	user := res.User
	tokenPair := res.TokenPair
	assert.Equal(t, "verify@example.com", user.Email)
	assert.True(t, user.EmailVerified)
	assert.NotEmpty(t, tokenPair.AccessToken)
	assert.NotEmpty(t, tokenPair.RefreshToken)
}

func TestForgotAndResetPassword(t *testing.T) {
	svc, _, _, _, _, emailSender := setupTestUsecase(t)
	ctx := context.Background()

	_, err := svc.Register(ctx, "reset@example.com", "password123", "Reset User")
	require.NoError(t, err)
	require.NoError(t, svc.VerifyEmail(ctx, emailSender.VerificationToken("reset@example.com")))

	res, err := svc.Login(ctx, "reset@example.com", "password123", domain.ClientMetadata{})
	oldTokenPair := res.TokenPair
	require.NoError(t, err)

	require.NoError(t, svc.ForgotPassword(ctx, "reset@example.com"))
	resetToken := emailSender.ResetToken("reset@example.com")
	require.NotEmpty(t, resetToken)
	require.NoError(t, svc.ResetPassword(ctx, resetToken, "newpassword123"))

	_, err = svc.Login(ctx, "reset@example.com", "password123", domain.ClientMetadata{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid email or password")

	_, err = svc.Login(ctx, "reset@example.com", "newpassword123", domain.ClientMetadata{})
	require.NoError(t, err)

	_, err = svc.RefreshToken(ctx, oldTokenPair.RefreshToken, domain.ClientMetadata{})
	require.Error(t, err)
}

func TestLoginLockoutPolicyIsRuntimeConfigurable(t *testing.T) {
	svc, userRepo, _, _, _, emailSender := setupTestUsecase(t)
	ctx := context.Background()

	registered, err := svc.Register(ctx, "lock@example.com", "password123", "Lock User")
	require.NoError(t, err)
	require.NoError(t, svc.VerifyEmail(ctx, emailSender.VerificationToken("lock@example.com")))

	_, err = svc.UpdateLoginLockoutPolicy(ctx, 2, 10*time.Minute)
	require.NoError(t, err)

	_, err = svc.Login(ctx, "lock@example.com", "wrong-password", domain.ClientMetadata{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid email or password")

	_, err = svc.Login(ctx, "lock@example.com", "wrong-password", domain.ClientMetadata{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "account is locked until")

	user, err := userRepo.GetByID(ctx, registered.ID)
	require.NoError(t, err)
	require.NotNil(t, user.LockedUntil)

	_, err = svc.Login(ctx, "lock@example.com", "password123", domain.ClientMetadata{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "account is locked until")

	policy, err := svc.GetLoginLockoutPolicy(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, policy.MaxFailedAttempts)
	assert.Equal(t, 10*time.Minute, policy.LockDuration)
}

func cloneUser(user *domain.User) *domain.User {
	cloned := *user
	if user.FullName != nil {
		fullName := *user.FullName
		cloned.FullName = &fullName
	}
	if user.LockedUntil != nil {
		lockedUntil := *user.LockedUntil
		cloned.LockedUntil = &lockedUntil
	}
	return &cloned
}
