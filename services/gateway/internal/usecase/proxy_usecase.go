package usecase

import (
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/base-go/base/services/gateway/internal/domain"
	"github.com/base-go/base/services/gateway/internal/platform/config"
)

// proxyUsecase implement domain.ProxyService.
type proxyUsecase struct {
	routes []*domain.Route // Sorted by prefix length (longest first).
}

// NewProxyUsecase tạo ProxyService từ route config.
// Routes được sort theo prefix dài nhất trước → longest prefix match.
func NewProxyUsecase(routeConfigs []config.RouteConfig) domain.ProxyService {
	routes := make([]*domain.Route, 0, len(routeConfigs))
	for _, rc := range routeConfigs {
		routes = append(routes, &domain.Route{
			Prefix:       rc.Prefix,
			Target:       rc.Target,
			StripPrefix:  rc.StripPrefix,
			RequiresAuth: rc.RequiresAuth,
			Methods:      rc.Methods,
		})
	}

	// Sort theo prefix dài nhất trước để longest prefix match.
	sort.Slice(routes, func(i, j int) bool {
		return len(routes[i].Prefix) > len(routes[j].Prefix)
	})

	return &proxyUsecase{routes: routes}
}

// FindRoute tìm route phù hợp nhất cho request path (longest prefix match).
// Trả về route, target path (sau khi strip prefix nếu cần), và error.
func (uc *proxyUsecase) FindRoute(path string) (*domain.Route, string, error) {
	for _, route := range uc.routes {
		if strings.HasPrefix(path, route.Prefix) {
			targetPath := path
			if route.StripPrefix {
				targetPath = strings.TrimPrefix(path, route.Prefix)
				if targetPath == "" {
					targetPath = "/"
				}
			}
			return route, targetPath, nil
		}
	}
	return nil, "", domain.ErrRouteNotFound
}

// BuildUpstreamRequest tạo HTTP request tới upstream service.
// Copy headers, query params từ original request.
func (uc *proxyUsecase) BuildUpstreamRequest(original *http.Request, route *domain.Route, targetPath string) (*http.Request, error) {
	// Build upstream URL.
	upstreamURL := route.Target + targetPath
	if original.URL.RawQuery != "" {
		upstreamURL += "?" + original.URL.RawQuery
	}

	// Tạo request mới với cùng method và body.
	req, err := http.NewRequestWithContext(original.Context(), original.Method, upstreamURL, original.Body)
	if err != nil {
		return nil, fmt.Errorf("create upstream request: %w", err)
	}

	// Copy headers từ original request.
	for key, values := range original.Header {
		for _, v := range values {
			req.Header.Add(key, v)
		}
	}

	// Thêm X-Forwarded-* headers cho upstream biết request đến từ gateway.
	req.Header.Set("X-Forwarded-For", original.RemoteAddr)
	req.Header.Set("X-Forwarded-Host", original.Host)
	req.Header.Set("X-Forwarded-Proto", schemeFromRequest(original))

	// Xoá hop-by-hop headers (không nên forward).
	req.Header.Del("Connection")
	req.Header.Del("Keep-Alive")
	req.Header.Del("Transfer-Encoding")
	req.Header.Del("TE")
	req.Header.Del("Trailer")
	req.Header.Del("Upgrade")

	return req, nil
}

func schemeFromRequest(r *http.Request) string {
	if r.TLS != nil {
		return "https"
	}
	if scheme := r.Header.Get("X-Forwarded-Proto"); scheme != "" {
		return scheme
	}
	return "http"
}
