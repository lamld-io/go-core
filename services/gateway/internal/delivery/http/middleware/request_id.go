package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	// HeaderRequestID là header chứa request ID.
	HeaderRequestID = "X-Request-ID"
)

// RequestID tạo middleware inject unique request ID vào mỗi request.
// Nếu client đã gửi X-Request-ID, sẽ sử dụng lại (tracing support).
// Nếu không, tạo UUID mới.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader(HeaderRequestID)
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// Set vào context và response header.
		c.Set(HeaderRequestID, requestID)
		c.Header(HeaderRequestID, requestID)

		// Set vào request header để forward tới downstream service.
		c.Request.Header.Set(HeaderRequestID, requestID)

		c.Next()
	}
}
