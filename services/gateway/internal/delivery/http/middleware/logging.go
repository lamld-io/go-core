package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

// Logging tạo middleware ghi log structured cho mỗi request.
// Log bao gồm: method, path, status, latency, client IP, request ID.
func Logging() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		// Xử lý request.
		c.Next()

		// Tính latency.
		latency := time.Since(start)
		status := c.Writer.Status()

		// Lấy request ID (đã set bởi RequestID middleware).
		requestID, _ := c.Get(HeaderRequestID)

		// Structured log fields.
		attrs := []any{
			"status", status,
			"method", c.Request.Method,
			"path", path,
			"latency", latency.String(),
			"latency_ms", latency.Milliseconds(),
			"client_ip", c.ClientIP(),
			"user_agent", c.Request.UserAgent(),
		}

		if query != "" {
			attrs = append(attrs, "query", query)
		}

		if requestID != nil {
			attrs = append(attrs, "request_id", requestID)
		}

		// Lấy user ID nếu đã authenticated.
		if userID, exists := c.Get(ContextUserID); exists {
			attrs = append(attrs, "user_id", userID)
		}

		// Log level theo status code.
		switch {
		case status >= 500:
			slog.Error("request completed", attrs...)
		case status >= 400:
			slog.Warn("request completed", attrs...)
		default:
			slog.Info("request completed", attrs...)
		}
	}
}
