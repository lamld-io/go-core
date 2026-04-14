package jwt

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Manager quản lý việc phát hành và xác thực JWT bằng RSA.
type Manager struct {
	privateKey      *rsa.PrivateKey
	publicKey       *rsa.PublicKey
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
	issuer          string
}

// Config chứa cấu hình cho JWT Manager.
type Config struct {
	PrivateKeyPath  string
	PublicKeyPath   string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
	Issuer          string
}

// NewManager tạo JWT Manager mới từ RSA key files.
func NewManager(cfg Config) (*Manager, error) {
	m := &Manager{
		accessTokenTTL:  cfg.AccessTokenTTL,
		refreshTokenTTL: cfg.RefreshTokenTTL,
		issuer:          cfg.Issuer,
	}

	// Load private key nếu có (Auth Service cần, Gateway không cần).
	if cfg.PrivateKeyPath != "" {
		privKey, err := loadPrivateKey(cfg.PrivateKeyPath)
		if err != nil {
			return nil, fmt.Errorf("load private key: %w", err)
		}
		m.privateKey = privKey
		m.publicKey = &privKey.PublicKey
	}

	// Load public key nếu có (Gateway chỉ cần public key).
	if cfg.PublicKeyPath != "" {
		pubKey, err := loadPublicKey(cfg.PublicKeyPath)
		if err != nil {
			return nil, fmt.Errorf("load public key: %w", err)
		}
		m.publicKey = pubKey
	}

	if m.publicKey == nil {
		return nil, fmt.Errorf("at least one of private key or public key must be provided")
	}

	return m, nil
}

// GenerateAccessToken phát hành access token cho user.
func (m *Manager) GenerateAccessToken(userID, email, role string) (string, error) {
	if m.privateKey == nil {
		return "", fmt.Errorf("private key not available, cannot sign tokens")
	}

	now := time.Now()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(),
			Issuer:    m.issuer,
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.accessTokenTTL)),
		},
		UserID:    userID,
		Email:     email,
		Role:      role,
		TokenType: AccessToken,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(m.privateKey)
}

// GenerateRefreshToken phát hành refresh token cho user.
func (m *Manager) GenerateRefreshToken(userID, email, role string) (string, error) {
	if m.privateKey == nil {
		return "", fmt.Errorf("private key not available, cannot sign tokens")
	}

	now := time.Now()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(),
			Issuer:    m.issuer,
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.refreshTokenTTL)),
		},
		UserID:    userID,
		Email:     email,
		Role:      role,
		TokenType: RefreshToken,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(m.privateKey)
}

// GenerateTemp2FAToken phát hành token tạm thời cho bước xác thực 2FA.
func (m *Manager) GenerateTemp2FAToken(userID, email string) (string, error) {
	if m.privateKey == nil {
		return "", fmt.Errorf("private key not available, cannot sign tokens")
	}

	now := time.Now()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(),
			Issuer:    m.issuer,
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(2 * time.Minute)), // Hạn mặc định 2 phút cho 2FA
		},
		UserID:    userID,
		Email:     email,
		Role:      "",
		TokenType: Temp2FAToken,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(m.privateKey)
}

// ValidateToken xác thực JWT và trả về claims.
func (m *Manager) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return m.publicKey, nil
	})
	if err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}

// GetRefreshTokenTTL trả về thời gian sống của refresh token.
func (m *Manager) GetRefreshTokenTTL() time.Duration {
	return m.refreshTokenTTL
}

func loadPrivateKey(path string) (*rsa.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file %s: %w", path, err)
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block from %s", path)
	}

	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		// Thử PKCS1 nếu PKCS8 không thành công
		rsaKey, rsaErr := x509.ParsePKCS1PrivateKey(block.Bytes)
		if rsaErr != nil {
			return nil, fmt.Errorf("parse private key: pkcs8=%w, pkcs1=%w", err, rsaErr)
		}
		return rsaKey, nil
	}

	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("key is not RSA private key")
	}
	return rsaKey, nil
}

func loadPublicKey(path string) (*rsa.PublicKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file %s: %w", path, err)
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block from %s", path)
	}

	key, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse public key: %w", err)
	}

	rsaKey, ok := key.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("key is not RSA public key")
	}
	return rsaKey, nil
}
