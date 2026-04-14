package handler

import (
	"io"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/base-go/base/pkg/response"
	"github.com/base-go/base/services/gateway/internal/domain"
	"github.com/base-go/base/services/gateway/internal/delivery/http/middleware"
	pkgjwt "github.com/base-go/base/pkg/jwt"
)

// ProxyHandler xử lý proxy request tới downstream service.
type ProxyHandler struct {
	proxyService domain.ProxyService
	httpClient   *http.Client
	jwtManager   *pkgjwt.Manager
}

// NewProxyHandler tạo ProxyHandler mới.
func NewProxyHandler(proxyService domain.ProxyService, httpClient *http.Client, jwtManager *pkgjwt.Manager) *ProxyHandler {
	return &ProxyHandler{
		proxyService: proxyService,
		httpClient:   httpClient,
		jwtManager:   jwtManager,
	}
}

// Handle xử lý tất cả incoming request, tìm route, authenticate nếu cần, rồi proxy.
func (h *ProxyHandler) Handle(c *gin.Context) {
	requestPath := c.Request.URL.Path

	// 1. Tìm route phù hợp.
	route, targetPath, err := h.proxyService.FindRoute(requestPath)
	if err != nil {
		response.Error(c, err)
		return
	}

	// 2. Kiểm tra method.
	if !route.MatchMethod(c.Request.Method) {
		response.Error(c, domain.ErrMethodNotAllowed)
		return
	}

	// 3. Authenticate nếu route yêu cầu.
	if route.RequiresAuth {
		middleware.Auth(h.jwtManager)(c)
		if c.IsAborted() {
			return // Auth middleware đã trả lỗi.
		}
	}

	// 4. Build upstream request.
	upstreamReq, err := h.proxyService.BuildUpstreamRequest(c.Request, route, targetPath)
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "failed to build upstream request",
			"error", err, "path", requestPath)
		response.Error(c, domain.ErrUpstreamUnavail)
		return
	}

	// 5. Forward request tới upstream.
	upstreamResp, err := h.httpClient.Do(upstreamReq)
	if err != nil {
		slog.ErrorContext(c.Request.Context(), "upstream request failed",
			"error", err, "target", route.Target, "path", targetPath)
		response.Error(c, domain.ErrUpstreamUnavail)
		return
	}
	defer upstreamResp.Body.Close()

	// 6. Copy response headers từ upstream → client.
	for key, values := range upstreamResp.Header {
		for _, v := range values {
			c.Header(key, v)
		}
	}

	// 7. Stream response body từ upstream → client.
	c.Status(upstreamResp.StatusCode)
	if _, err := io.Copy(c.Writer, upstreamResp.Body); err != nil {
		slog.ErrorContext(c.Request.Context(), "failed to stream upstream response",
			"error", err, "path", requestPath)
	}
}

// HealthCheck trả về trạng thái gateway.
func (h *ProxyHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"service": "gateway",
	})
}
