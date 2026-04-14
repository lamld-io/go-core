package domain

import (
	"net/http"
)

// ProxyService định nghĩa interface cho proxy logic.
type ProxyService interface {
	// FindRoute tìm route phù hợp cho request path.
	// Trả về Route và phần path còn lại sau khi strip prefix (nếu có).
	FindRoute(path string) (*Route, string, error)

	// BuildUpstreamRequest tạo HTTP request tới upstream service.
	BuildUpstreamRequest(original *http.Request, route *Route, targetPath string) (*http.Request, error)
}
