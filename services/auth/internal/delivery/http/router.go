package http

import (
	"github.com/gin-gonic/gin"

	pkgjwt "github.com/base-go/base/pkg/jwt"
	pkgmiddleware "github.com/base-go/base/pkg/middleware"
	"github.com/base-go/base/services/auth/internal/delivery/http/handler"
	"github.com/base-go/base/services/auth/internal/delivery/http/middleware"
	"github.com/base-go/base/services/auth/internal/platform/config"
	"github.com/redis/go-redis/v9"
)

// NewRouter tạo Gin engine với tất cả route cho Auth Service.
func NewRouter(authHandler *handler.AuthHandler, jwtManager *pkgjwt.Manager, redisClient *redis.Client, cfg *config.Config) *gin.Engine {
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
			public := auth.Group("")
			public.Use(pkgmiddleware.IPRateLimiter(redisClient, cfg.RateLimit.LoginLimit))
			{
				public.POST("/register", authHandler.Register)
				public.POST("/login", authHandler.Login)
				public.POST("/verify-email", authHandler.VerifyEmail)
				public.POST("/resend-verification-email", authHandler.ResendVerificationEmail)
				public.POST("/forgot-password", authHandler.ForgotPassword)
				public.POST("/reset-password", authHandler.ResetPassword)
				public.POST("/refresh", authHandler.RefreshToken)
			}

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
