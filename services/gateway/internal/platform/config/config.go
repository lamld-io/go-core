package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config chứa toàn bộ cấu hình cho Gateway Service.
type Config struct {
	Server    ServerConfig
	JWT       JWTConfig
	Routes    []RouteConfig
	RateLimit RateLimitConfig
	Proxy     ProxyConfig
}

// ServerConfig cấu hình HTTP server.
type ServerConfig struct {
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

// JWTConfig cấu hình JWT — Gateway chỉ cần public key để verify.
type JWTConfig struct {
	PublicKeyPath string
	Issuer        string
}

// RouteConfig cấu hình routing từ file YAML.
type RouteConfig struct {
	Prefix       string   `yaml:"prefix"`
	Target       string   `yaml:"target"`
	StripPrefix  bool     `yaml:"strip_prefix"`
	RequiresAuth bool     `yaml:"requires_auth"`
	Methods      []string `yaml:"methods,omitempty"`
}

// RoutesFileConfig cấu trúc file routes.yaml.
type RoutesFileConfig struct {
	Routes []RouteConfig `yaml:"routes"`
}

// RateLimitConfig cấu hình rate limiting.
type RateLimitConfig struct {
	Enabled        bool
	RequestsPerSec float64
	BurstSize      int
}

// ProxyConfig cấu hình proxy behavior.
type ProxyConfig struct {
	Timeout         time.Duration
	MaxIdleConns    int
	IdleConnTimeout time.Duration
}

// Load đọc cấu hình từ biến môi trường.
func Load() (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Port:         getEnv("SERVER_PORT", "8080"),
			ReadTimeout:  getDurationEnv("SERVER_READ_TIMEOUT", 15*time.Second),
			WriteTimeout: getDurationEnv("SERVER_WRITE_TIMEOUT", 30*time.Second),
		},
		JWT: JWTConfig{
			PublicKeyPath: getEnv("JWT_PUBLIC_KEY_PATH", "configs/keys/public.pem"),
			Issuer:        getEnv("JWT_ISSUER", "auth-service"),
		},
		RateLimit: RateLimitConfig{
			Enabled:        getBoolEnv("RATE_LIMIT_ENABLED", true),
			RequestsPerSec: getFloatEnv("RATE_LIMIT_RPS", 100),
			BurstSize:      getIntEnv("RATE_LIMIT_BURST", 200),
		},
		Proxy: ProxyConfig{
			Timeout:         getDurationEnv("PROXY_TIMEOUT", 30*time.Second),
			MaxIdleConns:    getIntEnv("PROXY_MAX_IDLE_CONNS", 100),
			IdleConnTimeout: getDurationEnv("PROXY_IDLE_CONN_TIMEOUT", 90*time.Second),
		},
	}

	// Load routes từ file YAML.
	routesPath := getEnv("ROUTES_CONFIG_PATH", "configs/routes.yaml")
	routes, err := loadRoutesFromFile(routesPath)
	if err != nil {
		return nil, fmt.Errorf("load routes config: %w", err)
	}
	cfg.Routes = routes

	return cfg, nil
}

// loadRoutesFromFile đọc cấu hình route từ file YAML.
func loadRoutesFromFile(path string) ([]RouteConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read routes file %s: %w", path, err)
	}

	var cfg RoutesFileConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse routes yaml: %w", err)
	}

	if len(cfg.Routes) == 0 {
		return nil, fmt.Errorf("no routes defined in %s", path)
	}

	return cfg.Routes, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getIntEnv(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	var n int
	fmt.Sscanf(v, "%d", &n)
	if n == 0 {
		return fallback
	}
	return n
}

func getFloatEnv(key string, fallback float64) float64 {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	var f float64
	fmt.Sscanf(v, "%f", &f)
	if f == 0 {
		return fallback
	}
	return f
}

func getBoolEnv(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v == "true" || v == "1" || v == "yes"
}

func getDurationEnv(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}
