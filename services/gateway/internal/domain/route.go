package domain

// Route định nghĩa một rule routing từ Gateway tới downstream service.
type Route struct {
	// Prefix là path prefix để match request, ví dụ "/api/v1/auth".
	Prefix string

	// Target là URL của downstream service, ví dụ "http://auth-service:8081".
	Target string

	// StripPrefix nếu true sẽ xoá prefix trước khi forward request.
	// Ví dụ: request /api/v1/users/123 → forward /123 (nếu prefix = /api/v1/users).
	StripPrefix bool

	// RequiresAuth nếu true, Gateway sẽ yêu cầu JWT access token hợp lệ.
	RequiresAuth bool

	// Methods giới hạn HTTP methods được phép. Rỗng = tất cả methods.
	Methods []string
}

// MatchMethod kiểm tra method có được phép cho route này không.
func (r *Route) MatchMethod(method string) bool {
	if len(r.Methods) == 0 {
		return true // Không giới hạn → cho phép tất cả.
	}
	for _, m := range r.Methods {
		if m == method {
			return true
		}
	}
	return false
}
