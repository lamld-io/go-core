package jwt

import (
	"github.com/golang-jwt/jwt/v5"
)

// TokenType phân biệt access token và refresh token.
type TokenType string

const (
	AccessToken  TokenType = "access"
	RefreshToken TokenType = "refresh"
)

// Claims chứa thông tin xác thực được nhúng vào JWT.
type Claims struct {
	jwt.RegisteredClaims
	UserID   string    `json:"user_id"`
	Email    string    `json:"email"`
	Role     string    `json:"role"`
	TokenType TokenType `json:"token_type"`
}
