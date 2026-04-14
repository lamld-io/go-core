package http

import (
	"github.com/gin-gonic/gin"

	pkgmiddleware "github.com/base-go/base/pkg/middleware"
	"github.com/base-go/base/services/gateway/internal/delivery/http/handler"
	"github.com/base-go/base/services/gateway/internal/delivery/http/middleware"
)

// RouterConfig chứa cấu hình cho router.
type RouterConfig struct {
	RateLimitEnabled bool
	RateLimitRPS     float64
	RateLimitBurst   int
}

// NewRouter tạo Gin engine cho Gateway Service.
// Middleware pipeline: Recovery → CORS → RequestID → Logging → RateLimit → Proxy
func NewRouter(proxyHandler *handler.ProxyHandler, cfg RouterConfig) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	r := gin.New()

	// 1. Recovery — bắt panic, trả 500 thay vì crash.
	r.Use(gin.Recovery())

	// 2. CORS.
	r.Use(pkgmiddleware.CORS(pkgmiddleware.DefaultCORSConfig()))

	// 3. Request ID — inject X-Request-ID cho tracing.
	r.Use(middleware.RequestID())

	// 4. Logging — structured log cho mỗi request.
	r.Use(middleware.Logging())

	// 5. Rate Limit — per-IP token bucket (nếu enabled).
	if cfg.RateLimitEnabled {
		r.Use(middleware.RateLimit(middleware.RateLimitConfig{
			RequestsPerSec: cfg.RateLimitRPS,
			BurstSize:      cfg.RateLimitBurst,
		}))
	}

	// Health check — bypass middleware pipeline (ngoại trừ recovery).
	r.GET("/health", proxyHandler.HealthCheck)

	// Catch-all: proxy tất cả request tới downstream service.
	// Handler tự xử lý route matching và authentication.
	r.NoRoute(proxyHandler.Handle)

	return r
}
