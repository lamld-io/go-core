package middleware

import (
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"

	"github.com/base-go/base/pkg/apperror"
	"github.com/base-go/base/pkg/response"
)

// RateLimitConfig cấu hình rate limiter.
type RateLimitConfig struct {
	// RequestsPerSec số request tối đa mỗi giây (token fill rate).
	RequestsPerSec float64
	// BurstSize số request tối đa trong burst.
	BurstSize int
}

// rateLimiterStore lưu limiter theo IP (in-memory, phù hợp single instance).
type rateLimiterStore struct {
	mu       sync.RWMutex
	limiters map[string]*rate.Limiter
	config   RateLimitConfig
}

func newRateLimiterStore(cfg RateLimitConfig) *rateLimiterStore {
	return &rateLimiterStore{
		limiters: make(map[string]*rate.Limiter),
		config:   cfg,
	}
}

func (s *rateLimiterStore) getLimiter(key string) *rate.Limiter {
	s.mu.RLock()
	limiter, exists := s.limiters[key]
	s.mu.RUnlock()

	if exists {
		return limiter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Double-check sau khi lấy write lock.
	if limiter, exists = s.limiters[key]; exists {
		return limiter
	}

	limiter = rate.NewLimiter(rate.Limit(s.config.RequestsPerSec), s.config.BurstSize)
	s.limiters[key] = limiter
	return limiter
}

// RateLimit tạo middleware rate limiting theo IP.
// Dùng token bucket algorithm (golang.org/x/time/rate).
func RateLimit(cfg RateLimitConfig) gin.HandlerFunc {
	store := newRateLimiterStore(cfg)

	return func(c *gin.Context) {
		// Rate limit theo client IP.
		clientIP := c.ClientIP()
		limiter := store.getLimiter(clientIP)

		if !limiter.Allow() {
			c.Header("Retry-After", "1")
			response.AbortWithError(c, apperror.RateLimited())
			return
		}

		c.Next()
	}
}

// GlobalRateLimit tạo middleware rate limiting global (không phân biệt IP).
// Dùng cho bảo vệ tổng tải trên server.
func GlobalRateLimit(rps float64, burst int) gin.HandlerFunc {
	limiter := rate.NewLimiter(rate.Limit(rps), burst)

	return func(c *gin.Context) {
		if !limiter.Allow() {
			c.Header("Retry-After", "1")
			c.AbortWithStatus(http.StatusTooManyRequests)
			return
		}
		c.Next()
	}
}
