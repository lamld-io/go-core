package http

import (
	"github.com/gin-gonic/gin"

	pkgjwt "github.com/base-go/base/pkg/jwt"
	pkgmiddleware "github.com/base-go/base/pkg/middleware"
	"github.com/base-go/base/services/auth/internal/delivery/http/handler"
	"github.com/base-go/base/services/auth/internal/delivery/http/middleware"
)

// NewRouter tạo Gin engine với tất cả route cho Auth Service.
func NewRouter(authHandler *handler.AuthHandler, jwtManager *pkgjwt.Manager) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	r := gin.New()

	// Global middleware
	r.Use(gin.Recovery())
	r.Use(pkgmiddleware.CORS(pkgmiddleware.DefaultCORSConfig()))

	// Health check — không cần auth
	r.GET("/health", authHandler.HealthCheck)

	// API v1 group
	v1 := r.Group("/api/v1")
	{
		auth := v1.Group("/auth")
		{
			// Public endpoints — không cần JWT
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.POST("/verify-email", authHandler.VerifyEmail)
			auth.POST("/resend-verification-email", authHandler.ResendVerificationEmail)
			auth.POST("/forgot-password", authHandler.ForgotPassword)
			auth.POST("/reset-password", authHandler.ResetPassword)
			auth.POST("/refresh", authHandler.RefreshToken)

			// Internal endpoint — dùng cho Gateway hoặc service khác validate token
			auth.GET("/validate", authHandler.ValidateToken)

			// Protected endpoints — cần JWT access token
			protected := auth.Group("")
			protected.Use(middleware.AuthMiddleware(jwtManager))
			{
				protected.POST("/logout", authHandler.Logout)
				protected.GET("/profile", authHandler.GetProfile)
				protected.GET("/security/lockout-policy", authHandler.GetLoginLockoutPolicy)
				protected.PUT("/security/lockout-policy", authHandler.UpdateLoginLockoutPolicy)
			}
		}
	}

	return r
}
