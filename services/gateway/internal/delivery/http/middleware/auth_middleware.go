package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/base-go/base/pkg/apperror"
	pkgjwt "github.com/base-go/base/pkg/jwt"
	"github.com/base-go/base/pkg/response"
)

const (
	// HeaderAuthorization là header chứa JWT token.
	HeaderAuthorization = "Authorization"
	// BearerPrefix là prefix của Bearer token.
	BearerPrefix = "Bearer "

	// ContextUserID lưu user ID trong gin.Context sau khi verify JWT.
	ContextUserID = "gateway_user_id"
	// ContextUserEmail lưu email trong gin.Context.
	ContextUserEmail = "gateway_user_email"
	// ContextUserRole lưu role trong gin.Context.
	ContextUserRole = "gateway_user_role"
)

// Auth tạo middleware xác thực JWT bằng RSA public key.
// Gateway chỉ cần public key để verify — không cần private key.
func Auth(jwtManager *pkgjwt.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader(HeaderAuthorization)
		if authHeader == "" {
			response.AbortWithError(c, apperror.Unauthorized("authorization header is required"))
			return
		}

		if !strings.HasPrefix(authHeader, BearerPrefix) {
			response.AbortWithError(c, apperror.Unauthorized("invalid authorization format, expected 'Bearer <token>'"))
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, BearerPrefix)
		if tokenStr == "" {
			response.AbortWithError(c, apperror.Unauthorized("token is required"))
			return
		}

		claims, err := jwtManager.ValidateToken(tokenStr)
		if err != nil {
			response.AbortWithError(c, apperror.TokenInvalid())
			return
		}

		if claims.TokenType != pkgjwt.AccessToken {
			response.AbortWithError(c, apperror.TokenInvalid())
			return
		}

		// Inject user info vào context → downstream service nhận qua X-User-* headers.
		c.Set(ContextUserID, claims.UserID)
		c.Set(ContextUserEmail, claims.Email)
		c.Set(ContextUserRole, claims.Role)

		// Thêm headers để downstream service biết user info mà không cần parse JWT lại.
		c.Request.Header.Set("X-User-ID", claims.UserID)
		c.Request.Header.Set("X-User-Email", claims.Email)
		c.Request.Header.Set("X-User-Role", claims.Role)

		c.Next()
	}
}
