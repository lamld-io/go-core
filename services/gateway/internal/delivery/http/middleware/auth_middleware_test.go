package middleware_test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pkgjwt "github.com/base-go/base/pkg/jwt"
	"github.com/base-go/base/services/gateway/internal/delivery/http/middleware"
)

func newTestJWTManager(t *testing.T) (*pkgjwt.Manager, *pkgjwt.Manager) {
	t.Helper()

	privateKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	dir := t.TempDir()

	privPath := filepath.Join(dir, "private.pem")
	pubPath := filepath.Join(dir, "public.pem")

	privBytes, _ := x509.MarshalPKCS8PrivateKey(privateKey)
	os.WriteFile(privPath, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privBytes}), 0600)
	pubBytes, _ := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	os.WriteFile(pubPath, pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubBytes}), 0600)

	// Signer (Auth Service) — có private key.
	signer, _ := pkgjwt.NewManager(pkgjwt.Config{
		PrivateKeyPath:  privPath,
		PublicKeyPath:   pubPath,
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 1 * time.Hour,
		Issuer:          "test",
	})

	// Verifier (Gateway) — chỉ public key.
	verifier, _ := pkgjwt.NewManager(pkgjwt.Config{
		PublicKeyPath:   pubPath,
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 1 * time.Hour,
		Issuer:          "test",
	})

	return signer, verifier
}

func setupGatewayRouter(verifier *pkgjwt.Manager) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	protected := r.Group("/protected")
	protected.Use(middleware.Auth(verifier))
	protected.GET("/resource", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"user_id": c.GetString(middleware.ContextUserID),
			"email":   c.GetString(middleware.ContextUserEmail),
			"role":    c.GetString(middleware.ContextUserRole),
		})
	})

	return r
}

func TestAuthMiddleware(t *testing.T) {
	signer, verifier := newTestJWTManager(t)
	router := setupGatewayRouter(verifier)

	// Tạo valid access token.
	validToken, err := signer.GenerateAccessToken("user-001", "test@test.com", "admin")
	require.NoError(t, err)

	// Tạo refresh token (không hợp lệ cho access).
	refreshToken, err := signer.GenerateRefreshToken("user-001", "test@test.com", "admin")
	require.NoError(t, err)

	// Tạo token từ key khác.
	otherKey, _ := rsa.GenerateKey(rand.Reader, 2048)
	otherDir := t.TempDir()
	otherPrivPath := filepath.Join(otherDir, "priv.pem")
	otherPubPath := filepath.Join(otherDir, "pub.pem")
	otherPrivBytes, _ := x509.MarshalPKCS8PrivateKey(otherKey)
	os.WriteFile(otherPrivPath, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: otherPrivBytes}), 0600)
	otherPubBytes, _ := x509.MarshalPKIXPublicKey(&otherKey.PublicKey)
	os.WriteFile(otherPubPath, pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: otherPubBytes}), 0600)
	otherSigner, _ := pkgjwt.NewManager(pkgjwt.Config{
		PrivateKeyPath: otherPrivPath, PublicKeyPath: otherPubPath,
		AccessTokenTTL: 15 * time.Minute, RefreshTokenTTL: 1 * time.Hour, Issuer: "other",
	})
	wrongKeyToken, _ := otherSigner.GenerateAccessToken("user-999", "hacker@evil.com", "admin")

	tests := []struct {
		name       string
		authHeader string
		wantStatus int
		wantUserID string
	}{
		{
			name:       "success — valid access token",
			authHeader: "Bearer " + validToken,
			wantStatus: http.StatusOK,
			wantUserID: "user-001",
		},
		{
			name:       "fail — no authorization header",
			authHeader: "",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "fail — missing Bearer prefix",
			authHeader: validToken,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "fail — empty token after Bearer",
			authHeader: "Bearer ",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "fail — garbage token",
			authHeader: "Bearer not-a-jwt",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "fail — refresh token (wrong type)",
			authHeader: "Bearer " + refreshToken,
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "fail — wrong key",
			authHeader: "Bearer " + wrongKeyToken,
			wantStatus: http.StatusUnauthorized,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/protected/resource", nil)
			if tc.authHeader != "" {
				req.Header.Set("Authorization", tc.authHeader)
			}
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)
			assert.Equal(t, tc.wantStatus, w.Code)

			if tc.wantStatus == http.StatusOK {
				var body map[string]interface{}
				json.Unmarshal(w.Body.Bytes(), &body)
				assert.Equal(t, tc.wantUserID, body["user_id"])
			}
		})
	}
}

func TestAuthMiddleware_InjectsHeaders(t *testing.T) {
	signer, verifier := newTestJWTManager(t)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.Auth(verifier))
	r.GET("/check", func(c *gin.Context) {
		// Gateway inject X-User-* headers vào request → downstream nhận.
		c.JSON(http.StatusOK, gin.H{
			"x_user_id":    c.Request.Header.Get("X-User-ID"),
			"x_user_email": c.Request.Header.Get("X-User-Email"),
			"x_user_role":  c.Request.Header.Get("X-User-Role"),
		})
	})

	token, _ := signer.GenerateAccessToken("user-123", "me@test.com", "admin")

	req := httptest.NewRequest("GET", "/check", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var body map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &body)
	assert.Equal(t, "user-123", body["x_user_id"])
	assert.Equal(t, "me@test.com", body["x_user_email"])
	assert.Equal(t, "admin", body["x_user_role"])
}
