package middleware

import (
	"log/slog"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis_rate/v10"
	"github.com/redis/go-redis/v9"
	"golang.org/x/time/rate"

	"github.com/base-go/base/pkg/apperror"
	"github.com/base-go/base/pkg/response"
)

var (
	localLimiters = make(map[string]*rate.Limiter)
	localMu       sync.Mutex
)

// getLocalLimiter trả về limiter cho IP, tạo mới nếu chưa tồn tại.
func getLocalLimiter(ip string, reqPerMin int) *rate.Limiter {
	localMu.Lock()
	defer localMu.Unlock()

	limiter, exists := localLimiters[ip]
	if !exists {
		// rate.Limit is events per second.
		r := rate.Limit(float64(reqPerMin) / 60.0)
		limiter = rate.NewLimiter(r, reqPerMin)
		localLimiters[ip] = limiter
	}
	return limiter
}

// IPRateLimiter giới hạn tốc độ request theo IP bằng Redis, fallback bộ nhớ nếu Redis lỗi.
func IPRateLimiter(redisClient *redis.Client, requestsPerMin int) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()

		// 1. Dùng Redis Rate Limiter nếu khả dụng
		if redisClient != nil {
			limiter := redis_rate.NewLimiter(redisClient)
			// Key theo IP
			key := "rate_limit:ip:" + ip
			res, err := limiter.Allow(c.Request.Context(), key, redis_rate.PerMinute(requestsPerMin))
			if err == nil {
				if res.Allowed == 0 {
					response.AbortWithError(c, apperror.RateLimited())
					return
				}
				c.Next()
				return
			}
			slog.WarnContext(c.Request.Context(), "redis rate limiter failed, falling back to memory", "error", err, "ip", ip)
		}

		// 2. Fallback sang In-Memory Rate Limiter nếu Redis lỗi
		fallbackLimiter := getLocalLimiter(ip, requestsPerMin)
		if !fallbackLimiter.Allow() {
			response.AbortWithError(c, apperror.RateLimited())
			return
		}

		c.Next()
	}
}
