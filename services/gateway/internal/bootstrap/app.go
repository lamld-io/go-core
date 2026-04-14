package bootstrap

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	pkgjwt "github.com/base-go/base/pkg/jwt"
	deliveryhttp "github.com/base-go/base/services/gateway/internal/delivery/http"
	"github.com/base-go/base/services/gateway/internal/delivery/http/handler"
	"github.com/base-go/base/services/gateway/internal/platform/config"
	"github.com/base-go/base/services/gateway/internal/platform/logger"
	"github.com/base-go/base/services/gateway/internal/usecase"
)

// App chứa tất cả dependency đã wire và server HTTP.
type App struct {
	cfg    *config.Config
	server *http.Server
}

// NewApp khởi tạo toàn bộ dependency theo thứ tự:
// config → logger → jwt (public key only) → proxy usecase → http client → handler → router → server
func NewApp() (*App, error) {
	// 1. Load config.
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	// 2. Setup logger.
	env := os.Getenv("APP_ENV")
	logger.Setup(env)
	slog.Info("starting gateway service", "port", cfg.Server.Port, "env", env, "routes", len(cfg.Routes))

	// 3. Khởi tạo JWT Manager — Gateway chỉ cần public key để verify token.
	jwtManager, err := pkgjwt.NewManager(pkgjwt.Config{
		PublicKeyPath: cfg.JWT.PublicKeyPath,
		Issuer:        cfg.JWT.Issuer,
		// AccessTokenTTL và RefreshTokenTTL không cần vì Gateway không sign token.
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 7 * 24 * time.Hour,
	})
	if err != nil {
		return nil, fmt.Errorf("init jwt manager: %w", err)
	}

	// 4. Khởi tạo proxy usecase.
	proxyService := usecase.NewProxyUsecase(cfg.Routes)

	// 5. Tạo HTTP client cho proxy — tuned cho reverse proxy workload.
	httpClient := &http.Client{
		Timeout: cfg.Proxy.Timeout,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   10 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			MaxIdleConns:        cfg.Proxy.MaxIdleConns,
			MaxIdleConnsPerHost: 20,
			IdleConnTimeout:     cfg.Proxy.IdleConnTimeout,
			TLSClientConfig:     &tls.Config{MinVersion: tls.VersionTLS12},
			DisableCompression:  false,
		},
		// Không follow redirects — trả redirect response cho client tự xử lý.
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	// 6. Khởi tạo handler.
	proxyHandler := handler.NewProxyHandler(proxyService, httpClient, jwtManager)

	// 7. Tạo router.
	router := deliveryhttp.NewRouter(proxyHandler, deliveryhttp.RouterConfig{
		RateLimitEnabled: cfg.RateLimit.Enabled,
		RateLimitRPS:     cfg.RateLimit.RequestsPerSec,
		RateLimitBurst:   cfg.RateLimit.BurstSize,
	})

	// 8. Tạo HTTP server.
	server := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	// Log loaded routes.
	for _, r := range cfg.Routes {
		slog.Info("route registered",
			"prefix", r.Prefix,
			"target", r.Target,
			"auth", r.RequiresAuth,
			"strip_prefix", r.StripPrefix,
		)
	}

	return &App{
		cfg:    cfg,
		server: server,
	}, nil
}

// Run khởi chạy HTTP server và xử lý graceful shutdown.
func (a *App) Run() error {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	errCh := make(chan error, 1)
	go func() {
		slog.Info("gateway service is running", "addr", a.server.Addr)
		if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("server error: %w", err)
		}
	}()

	select {
	case err := <-errCh:
		return err
	case sig := <-quit:
		slog.Info("received shutdown signal", "signal", sig.String())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := a.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown: %w", err)
	}

	slog.Info("gateway service stopped gracefully")
	return nil
}
