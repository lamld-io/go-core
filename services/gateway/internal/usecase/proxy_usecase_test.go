package usecase_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/base-go/base/services/gateway/internal/platform/config"
	"github.com/base-go/base/services/gateway/internal/usecase"
)

func setupRoutes() []config.RouteConfig {
	return []config.RouteConfig{
		{Prefix: "/api/v1/auth", Target: "http://auth:8081", StripPrefix: false, RequiresAuth: false},
		{Prefix: "/api/v1/auth/profile", Target: "http://auth:8081", StripPrefix: false, RequiresAuth: true},
		{Prefix: "/api/v1/users", Target: "http://users:8082", StripPrefix: true, RequiresAuth: true},
		{Prefix: "/api/v1/products", Target: "http://products:8083", StripPrefix: false, RequiresAuth: true, Methods: []string{"GET", "POST"}},
	}
}

func TestFindRoute(t *testing.T) {
	svc := usecase.NewProxyUsecase(setupRoutes())

	tests := []struct {
		name         string
		path         string
		wantPrefix   string
		wantTarget   string
		wantPath     string
		wantAuth     bool
		wantErr      bool
	}{
		{
			name:       "match auth — exact prefix",
			path:       "/api/v1/auth/login",
			wantPrefix: "/api/v1/auth",
			wantTarget: "http://auth:8081",
			wantPath:   "/api/v1/auth/login",
			wantAuth:   false,
		},
		{
			name:       "match auth profile — longest prefix wins",
			path:       "/api/v1/auth/profile",
			wantPrefix: "/api/v1/auth/profile",
			wantTarget: "http://auth:8081",
			wantPath:   "/api/v1/auth/profile",
			wantAuth:   true,
		},
		{
			name:       "match users — strip prefix",
			path:       "/api/v1/users/123",
			wantPrefix: "/api/v1/users",
			wantTarget: "http://users:8082",
			wantPath:   "/123",
			wantAuth:   true,
		},
		{
			name:       "match users — strip prefix root",
			path:       "/api/v1/users",
			wantPrefix: "/api/v1/users",
			wantTarget: "http://users:8082",
			wantPath:   "/",
			wantAuth:   true,
		},
		{
			name:       "match products",
			path:       "/api/v1/products/456",
			wantPrefix: "/api/v1/products",
			wantTarget: "http://products:8083",
			wantPath:   "/api/v1/products/456",
			wantAuth:   true,
		},
		{
			name:    "no match — unknown path",
			path:    "/api/v2/unknown",
			wantErr: true,
		},
		{
			name:    "no match — root",
			path:    "/",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			route, targetPath, err := svc.FindRoute(tc.path)

			if tc.wantErr {
				assert.Error(t, err)
				assert.Nil(t, route)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.wantPrefix, route.Prefix)
				assert.Equal(t, tc.wantTarget, route.Target)
				assert.Equal(t, tc.wantPath, targetPath)
				assert.Equal(t, tc.wantAuth, route.RequiresAuth)
			}
		})
	}
}

func TestBuildUpstreamRequest(t *testing.T) {
	svc := usecase.NewProxyUsecase(setupRoutes())

	tests := []struct {
		name           string
		method         string
		path           string
		query          string
		headers        map[string]string
		routePrefix    string
		wantUpstream   string
	}{
		{
			name:         "GET with query params",
			method:       "GET",
			path:         "/api/v1/auth/login",
			query:        "redirect=home",
			routePrefix:  "/api/v1/auth",
			wantUpstream: "http://auth:8081/api/v1/auth/login?redirect=home",
		},
		{
			name:         "POST without query",
			method:       "POST",
			path:         "/api/v1/auth/register",
			routePrefix:  "/api/v1/auth",
			wantUpstream: "http://auth:8081/api/v1/auth/register",
		},
		{
			name:         "strip prefix path",
			method:       "GET",
			path:         "/api/v1/users/123",
			routePrefix:  "/api/v1/users",
			wantUpstream: "http://users:8082/123",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			url := "http://gateway:8080" + tc.path
			if tc.query != "" {
				url += "?" + tc.query
			}

			original := httptest.NewRequest(tc.method, url, nil)
			for k, v := range tc.headers {
				original.Header.Set(k, v)
			}

			route, targetPath, err := svc.FindRoute(tc.path)
			require.NoError(t, err)

			upstream, err := svc.BuildUpstreamRequest(original, route, targetPath)
			require.NoError(t, err)

			assert.Equal(t, tc.method, upstream.Method)
			assert.Equal(t, tc.wantUpstream, upstream.URL.String())

			// X-Forwarded headers phải có.
			assert.NotEmpty(t, upstream.Header.Get("X-Forwarded-For"))
			assert.NotEmpty(t, upstream.Header.Get("X-Forwarded-Host"))
			assert.NotEmpty(t, upstream.Header.Get("X-Forwarded-Proto"))

			// Hop-by-hop headers phải bị xoá.
			assert.Empty(t, upstream.Header.Get("Connection"))
			assert.Empty(t, upstream.Header.Get("Keep-Alive"))
		})
	}
}

func TestBuildUpstreamRequest_ForwardHeaders(t *testing.T) {
	svc := usecase.NewProxyUsecase(setupRoutes())

	original := httptest.NewRequest("POST", "http://gateway:8080/api/v1/auth/register", strings.NewReader(`{"email":"test@test.com"}`))
	original.Header.Set("Content-Type", "application/json")
	original.Header.Set("X-Request-ID", "req-123")
	original.Header.Set("X-User-ID", "user-456")

	route, targetPath, _ := svc.FindRoute("/api/v1/auth/register")
	upstream, err := svc.BuildUpstreamRequest(original, route, targetPath)
	require.NoError(t, err)

	// Custom headers phải được forward.
	assert.Equal(t, "application/json", upstream.Header.Get("Content-Type"))
	assert.Equal(t, "req-123", upstream.Header.Get("X-Request-ID"))
	assert.Equal(t, "user-456", upstream.Header.Get("X-User-ID"))
}

func TestRouteMatchMethod(t *testing.T) {
	svc := usecase.NewProxyUsecase(setupRoutes())

	route, _, err := svc.FindRoute("/api/v1/products/1")
	require.NoError(t, err)

	tests := []struct {
		method string
		want   bool
	}{
		{"GET", true},
		{"POST", true},
		{"PUT", false},
		{"DELETE", false},
		{"PATCH", false},
	}

	for _, tc := range tests {
		t.Run(tc.method, func(t *testing.T) {
			assert.Equal(t, tc.want, route.MatchMethod(tc.method))
		})
	}
}

func TestRouteMatchMethod_AllMethods(t *testing.T) {
	svc := usecase.NewProxyUsecase(setupRoutes())

	// Auth route — no method restriction.
	route, _, err := svc.FindRoute("/api/v1/auth/login")
	require.NoError(t, err)

	for _, method := range []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"} {
		assert.True(t, route.MatchMethod(method), "method %s should be allowed", method)
	}
}

func TestProxyIntegration(t *testing.T) {
	// Tạo fake upstream server.
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Upstream", "true")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"from_upstream","path":"` + r.URL.Path + `"}`))
	}))
	defer upstream.Close()

	// Config routes trỏ vào fake upstream.
	routes := []config.RouteConfig{
		{Prefix: "/api/v1/test", Target: upstream.URL, StripPrefix: false, RequiresAuth: false},
	}
	svc := usecase.NewProxyUsecase(routes)

	// Build và thực thi request.
	original := httptest.NewRequest("GET", "http://gateway/api/v1/test/hello", nil)
	route, targetPath, err := svc.FindRoute("/api/v1/test/hello")
	require.NoError(t, err)

	upstreamReq, err := svc.BuildUpstreamRequest(original, route, targetPath)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(upstreamReq)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "true", resp.Header.Get("X-Upstream"))
}
