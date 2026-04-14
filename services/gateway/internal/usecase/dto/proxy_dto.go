package dto

// ProxyError chứa thông tin lỗi từ upstream service.
type ProxyError struct {
	StatusCode int
	Body       []byte
	Headers    map[string][]string
}
