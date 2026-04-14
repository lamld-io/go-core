package jwt_test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pkgjwt "github.com/base-go/base/pkg/jwt"
)

// generateTestKeys tạo RSA key pair tạm thời cho test.
func generateTestKeys(t *testing.T) (privPath, pubPath string) {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	dir := t.TempDir()

	// Lưu private key.
	privPath = filepath.Join(dir, "private.pem")
	privBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	require.NoError(t, err)
	privPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privBytes})
	require.NoError(t, os.WriteFile(privPath, privPEM, 0600))

	// Lưu public key.
	pubPath = filepath.Join(dir, "public.pem")
	pubBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	require.NoError(t, err)
	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubBytes})
	require.NoError(t, os.WriteFile(pubPath, pubPEM, 0600))

	return privPath, pubPath
}

// newTestManager tạo JWT Manager cho test.
func newTestManager(t *testing.T) *pkgjwt.Manager {
	t.Helper()
	privPath, pubPath := generateTestKeys(t)

	mgr, err := pkgjwt.NewManager(pkgjwt.Config{
		PrivateKeyPath:  privPath,
		PublicKeyPath:   pubPath,
		AccessTokenTTL:  5 * time.Minute,
		RefreshTokenTTL: 1 * time.Hour,
		Issuer:          "test-issuer",
	})
	require.NoError(t, err)
	return mgr
}

func TestGenerateAndValidateAccessToken(t *testing.T) {
	mgr := newTestManager(t)

	tests := []struct {
		name   string
		userID string
		email  string
		role   string
	}{
		{"admin user", "user-001", "admin@test.com", "admin"},
		{"regular user", "user-002", "user@test.com", "user"},
		{"empty full name", "user-003", "noname@test.com", "user"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Generate access token.
			token, err := mgr.GenerateAccessToken(tc.userID, tc.email, tc.role)
			require.NoError(t, err)
			assert.NotEmpty(t, token)

			// Validate token.
			claims, err := mgr.ValidateToken(token)
			require.NoError(t, err)

			assert.Equal(t, tc.userID, claims.UserID)
			assert.Equal(t, tc.email, claims.Email)
			assert.Equal(t, tc.role, claims.Role)
			assert.Equal(t, pkgjwt.AccessToken, claims.TokenType)
			assert.Equal(t, "test-issuer", claims.Issuer)
			assert.Equal(t, tc.userID, claims.Subject)
		})
	}
}

func TestGenerateAndValidateRefreshToken(t *testing.T) {
	mgr := newTestManager(t)

	token, err := mgr.GenerateRefreshToken("user-001", "user@test.com", "user")
	require.NoError(t, err)
	assert.NotEmpty(t, token)

	claims, err := mgr.ValidateToken(token)
	require.NoError(t, err)

	assert.Equal(t, "user-001", claims.UserID)
	assert.Equal(t, pkgjwt.RefreshToken, claims.TokenType)
}

func TestValidateToken_InvalidToken(t *testing.T) {
	mgr := newTestManager(t)

	tests := []struct {
		name  string
		token string
	}{
		{"empty token", ""},
		{"garbage token", "not-a-jwt"},
		{"malformed jwt", "eyJ.eyJ.sig"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			claims, err := mgr.ValidateToken(tc.token)
			assert.Error(t, err)
			assert.Nil(t, claims)
		})
	}
}

func TestValidateToken_WrongKey(t *testing.T) {
	mgr1 := newTestManager(t)
	mgr2 := newTestManager(t) // Different key pair.

	// Token signed by mgr1 should NOT be valid with mgr2.
	token, err := mgr1.GenerateAccessToken("user-001", "user@test.com", "user")
	require.NoError(t, err)

	claims, err := mgr2.ValidateToken(token)
	assert.Error(t, err)
	assert.Nil(t, claims)
}

func TestNewManager_PublicKeyOnly(t *testing.T) {
	_, pubPath := generateTestKeys(t)

	// Gateway mode: chỉ có public key, không có private key.
	mgr, err := pkgjwt.NewManager(pkgjwt.Config{
		PublicKeyPath:   pubPath,
		AccessTokenTTL:  5 * time.Minute,
		RefreshTokenTTL: 1 * time.Hour,
		Issuer:          "test-issuer",
	})
	require.NoError(t, err)

	// Không thể sign token.
	_, err = mgr.GenerateAccessToken("user-001", "user@test.com", "user")
	assert.Error(t, err)
}

func TestNewManager_NoKeys(t *testing.T) {
	_, err := pkgjwt.NewManager(pkgjwt.Config{
		AccessTokenTTL:  5 * time.Minute,
		RefreshTokenTTL: 1 * time.Hour,
		Issuer:          "test-issuer",
	})
	assert.Error(t, err)
}
