package handler_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pkgjwt "github.com/base-go/base/pkg/jwt"
	"github.com/base-go/base/pkg/response"
	deliveryhttp "github.com/base-go/base/services/auth/internal/delivery/http"
	"github.com/base-go/base/services/auth/internal/delivery/http/handler"
	"github.com/base-go/base/services/auth/internal/domain"
	"github.com/base-go/base/services/auth/internal/platform/config"
	"github.com/base-go/base/services/auth/internal/usecase"
	"github.com/base-go/base/services/auth/internal/usecase/dto"
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

func (r *mockUserRepo) setRole(email, role string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if user, ok := r.users[email]; ok {
		user.Role = role
	}
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

func (r *mockTokenRepo) RevokeByID(_ context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, token := range r.tokens {
		if token.ID == id {
			token.Revoked = true
		}
	}
	return nil
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

func (m *mockEmailSender) verificationToken(email string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.verificationTokens[email]
}

func (m *mockEmailSender) resetToken(email string) string {
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
	return ok, nil
}

func setupTestRouter(t *testing.T) (*httptest.Server, *mockUserRepo, *mockEmailSender) {
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

	authService := usecase.NewAuthUsecase(
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

	authHandler := handler.NewAuthHandler(authService, 900)
	cfg := &config.Config{RateLimit: config.RateLimitConfig{LoginLimit: 1000000}}
	router := deliveryhttp.NewRouter(authHandler, jwtManager, nil, tokenBlacklist, cfg)
	ts := httptest.NewServer(router)
	t.Cleanup(ts.Close)

	return ts, userRepo, emailSender
}

func TestHandler_RegisterVerifyAndLogin(t *testing.T) {
	ts, _, emailSender := setupTestRouter(t)

	registerResp, err := postJSON(ts.URL+"/api/v1/auth/register", dto.RegisterRequest{
		Email:    "handler@test.com",
		Password: "password123",
		FullName: "Handler User",
	})
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, registerResp.StatusCode)
	registerBody := decodeResponse(t, registerResp)
	assert.Equal(t, "SUCCESS", registerBody.Code)

	loginResp, err := postJSON(ts.URL+"/api/v1/auth/login", dto.LoginRequest{
		Email:    "handler@test.com",
		Password: "password123",
	})
	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, loginResp.StatusCode)
	loginResp.Body.Close()

	verifyResp, err := postJSON(ts.URL+"/api/v1/auth/verify-email", dto.VerifyEmailRequest{
		Token: emailSender.verificationToken("handler@test.com"),
	})
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, verifyResp.StatusCode)
	verifyResp.Body.Close()

	loginResp, err = postJSON(ts.URL+"/api/v1/auth/login", dto.LoginRequest{
		Email:    "handler@test.com",
		Password: "password123",
	})
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, loginResp.StatusCode)
	loginResp.Body.Close()
}

func TestHandler_ForgotAndResetPassword(t *testing.T) {
	ts, _, emailSender := setupTestRouter(t)

	_, err := postJSON(ts.URL+"/api/v1/auth/register", dto.RegisterRequest{
		Email:    "forgot@test.com",
		Password: "password123",
		FullName: "Forgot User",
	})
	require.NoError(t, err)

	_, err = postJSON(ts.URL+"/api/v1/auth/verify-email", dto.VerifyEmailRequest{
		Token: emailSender.verificationToken("forgot@test.com"),
	})
	require.NoError(t, err)

	resp, err := postJSON(ts.URL+"/api/v1/auth/forgot-password", dto.ForgotPasswordRequest{
		Email: "forgot@test.com",
	})
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	resp, err = postJSON(ts.URL+"/api/v1/auth/reset-password", dto.ResetPasswordRequest{
		Token:       emailSender.resetToken("forgot@test.com"),
		NewPassword: "newpassword123",
	})
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()

	resp, err = postJSON(ts.URL+"/api/v1/auth/login", dto.LoginRequest{
		Email:    "forgot@test.com",
		Password: "password123",
	})
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	resp.Body.Close()

	resp, err = postJSON(ts.URL+"/api/v1/auth/login", dto.LoginRequest{
		Email:    "forgot@test.com",
		Password: "newpassword123",
	})
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	resp.Body.Close()
}

func TestHandler_AdminLockoutPolicyEndpoints(t *testing.T) {
	ts, userRepo, emailSender := setupTestRouter(t)

	_, err := postJSON(ts.URL+"/api/v1/auth/register", dto.RegisterRequest{
		Email:    "admin@test.com",
		Password: "password123",
		FullName: "Admin User",
	})
	require.NoError(t, err)

	_, err = postJSON(ts.URL+"/api/v1/auth/verify-email", dto.VerifyEmailRequest{
		Token: emailSender.verificationToken("admin@test.com"),
	})
	require.NoError(t, err)

	userRepo.setRole("admin@test.com", domain.RoleAdmin)

	loginResp, err := postJSON(ts.URL+"/api/v1/auth/login", dto.LoginRequest{
		Email:    "admin@test.com",
		Password: "password123",
	})
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, loginResp.StatusCode)
	accessToken := extractAccessToken(t, loginResp)

	getResp, err := getWithAuth(ts.URL+"/api/v1/auth/security/lockout-policy", accessToken)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, getResp.StatusCode)
	getResp.Body.Close()

	updateResp, err := putJSONWithAuth(ts.URL+"/api/v1/auth/security/lockout-policy", accessToken, dto.UpdateLockoutPolicyRequest{
		MaxFailedAttempts:   2,
		LockDurationSeconds: 600,
	})
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, updateResp.StatusCode)
	updateResp.Body.Close()

	_, err = postJSON(ts.URL+"/api/v1/auth/register", dto.RegisterRequest{
		Email:    "victim@test.com",
		Password: "password123",
		FullName: "Victim User",
	})
	require.NoError(t, err)

	_, err = postJSON(ts.URL+"/api/v1/auth/verify-email", dto.VerifyEmailRequest{
		Token: emailSender.verificationToken("victim@test.com"),
	})
	require.NoError(t, err)

	resp, err := postJSON(ts.URL+"/api/v1/auth/login", dto.LoginRequest{
		Email:    "victim@test.com",
		Password: "wrong-password",
	})
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	resp.Body.Close()

	resp, err = postJSON(ts.URL+"/api/v1/auth/login", dto.LoginRequest{
		Email:    "victim@test.com",
		Password: "wrong-password",
	})
	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	resp.Body.Close()
}

func postJSON(url string, body interface{}) (*http.Response, error) {
	payload, _ := json.Marshal(body)
	return http.Post(url, "application/json", bytes.NewReader(payload))
}

func getWithAuth(url, token string) (*http.Response, error) {
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	return http.DefaultClient.Do(req)
}

func putJSONWithAuth(url, token string, body interface{}) (*http.Response, error) {
	payload, _ := json.Marshal(body)
	req, _ := http.NewRequest(http.MethodPut, url, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	return http.DefaultClient.Do(req)
}

func decodeResponse(t *testing.T, resp *http.Response) response.Response {
	t.Helper()
	defer resp.Body.Close()

	var result response.Response
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	return result
}

func extractAccessToken(t *testing.T, resp *http.Response) string {
	t.Helper()

	body := decodeResponse(t, resp)
	dataMap, ok := body.Data.(map[string]interface{})
	require.True(t, ok)
	tokenMap, ok := dataMap["token"].(map[string]interface{})
	require.True(t, ok)
	accessToken, ok := tokenMap["access_token"].(string)
	require.True(t, ok)
	return accessToken
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
