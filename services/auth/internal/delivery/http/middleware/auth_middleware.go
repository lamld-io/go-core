package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/base-go/base/pkg/apperror"
	pkgjwt "github.com/base-go/base/pkg/jwt"
	"github.com/base-go/base/pkg/response"
	"github.com/base-go/base/services/auth/internal/domain"
)

const (
	// AuthorizationHeader là header chứa JWT token.
	AuthorizationHeader = "Authorization"
	// BearerPrefix là prefix của Bearer token.
	BearerPrefix = "Bearer "

	// ContextKeyUserID là key lưu user ID trong gin.Context.
	ContextKeyUserID = "user_id"
	// ContextKeyEmail là key lưu email trong gin.Context.
	ContextKeyEmail = "user_email"
	// ContextKeyRole là key lưu role trong gin.Context.
	ContextKeyRole = "user_role"
)

// AuthMiddleware xác thực JWT access token và inject user info vào context.
func AuthMiddleware(jwtManager *pkgjwt.Manager, blacklist domain.TokenBlacklist) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Lấy token từ Authorization header.
		authHeader := c.GetHeader(AuthorizationHeader)
		if authHeader == "" {
			response.AbortWithError(c, apperror.Unauthorized("authorization header is required"))
			return
		}

		if !strings.HasPrefix(authHeader, BearerPrefix) {
			response.AbortWithError(c, apperror.Unauthorized("invalid authorization header format, expected 'Bearer <token>'"))
			return
		}

		tokenString := strings.TrimPrefix(authHeader, BearerPrefix)
		if tokenString == "" {
			response.AbortWithError(c, apperror.Unauthorized("token is required"))
			return
		}

		// Validate JWT.
		claims, err := jwtManager.ValidateToken(tokenString)
		if err != nil {
			response.AbortWithError(c, apperror.TokenInvalid())
			return
		}

		// Kiểm tra danh sách đen.
		if isBlacklisted, _ := blacklist.IsBlacklisted(c.Request.Context(), claims.ID); isBlacklisted {
			response.AbortWithError(c, apperror.Unauthorized("token has been revoked"))
			return
		}

		// Kiểm tra phải là access token.
		if claims.TokenType != pkgjwt.AccessToken {
			response.AbortWithError(c, apperror.TokenInvalid())
			return
		}

		// Inject user info vào context để handler sử dụng.
		c.Set(ContextKeyUserID, claims.UserID)
		c.Set(ContextKeyEmail, claims.Email)
		c.Set(ContextKeyRole, claims.Role)

		c.Next()
	}
}

// GetUserIDFromContext lấy user ID từ gin.Context.
func GetUserIDFromContext(c *gin.Context) (string, bool) {
	id, exists := c.Get(ContextKeyUserID)
	if !exists {
		return "", false
	}
	return id.(string), true
}

// GetRoleFromContext lấy role từ gin.Context.
func GetRoleFromContext(c *gin.Context) (string, bool) {
	role, exists := c.Get(ContextKeyRole)
	if !exists {
		return "", false
	}
	return role.(string), true
}
