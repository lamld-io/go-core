package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	pkgjwt "github.com/base-go/base/pkg/jwt"
	deliveryhttp "github.com/base-go/base/services/auth/internal/delivery/http"
	"github.com/base-go/base/services/auth/internal/delivery/http/handler"
	"github.com/base-go/base/services/auth/internal/platform/config"
	"github.com/base-go/base/services/auth/internal/platform/database"
	platformemail "github.com/base-go/base/services/auth/internal/platform/email"
	"github.com/base-go/base/services/auth/internal/platform/logger"
	"github.com/base-go/base/services/auth/internal/repository/postgres"
	"github.com/base-go/base/services/auth/internal/repository/postgres/model"
	"github.com/base-go/base/services/auth/internal/usecase"
	"github.com/redis/go-redis/v9"
)

// App chứa tất cả dependency đã wire và server HTTP.
type App struct {
	cfg    *config.Config
	server *http.Server
}

// NewApp khởi tạo toàn bộ dependency theo thứ tự:
// config → logger → database → jwt → repository → usecase → handler → router → server
func NewApp() (*App, error) {
	// 1. Load config từ env.
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	// 2. Setup logger.
	env := os.Getenv("APP_ENV")
	logger.Setup(env)
	slog.Info("starting auth service", "port", cfg.Server.Port, "env", env)

	// 3. Kết nối database.
	db, err := database.NewPostgresDB(database.Config{
		DSN: cfg.Database.DSN(),
	})
	if err != nil {
		return nil, fmt.Errorf("connect database: %w", err)
	}

	// Auto-migrate: tạo bảng nếu chưa tồn tại.
	if err := db.AutoMigrate(
		&model.UserModel{},
		&model.TokenModel{},
		&model.ActionTokenModel{},
		&model.LoginLockoutPolicyModel{},
	); err != nil {
		return nil, fmt.Errorf("auto-migrate: %w", err)
	}
	slog.Info("database migration completed")

	// 4. Kết nối Redis.
	redisClient := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		slog.Warn("failed to connect to redis, rate limiter will use in-memory fallback", "error", err)
		redisClient = nil
	} else {
		slog.Info("connected to redis successfully")
	}

	// 5. Khởi tạo JWT Manager (Auth Service cần private key để sign).
	jwtManager, err := pkgjwt.NewManager(pkgjwt.Config{
		PrivateKeyPath:  cfg.JWT.PrivateKeyPath,
		PublicKeyPath:   cfg.JWT.PublicKeyPath,
		AccessTokenTTL:  cfg.JWT.AccessTokenTTL,
		RefreshTokenTTL: cfg.JWT.RefreshTokenTTL,
		Issuer:          cfg.JWT.Issuer,
	})
	if err != nil {
		return nil, fmt.Errorf("init jwt manager: %w", err)
	}

	// 5. Khởi tạo repositories.
	userRepo := postgres.NewUserRepository(db)
	tokenRepo := postgres.NewTokenRepository(db)
	actionTokenRepo := postgres.NewActionTokenRepository(db)
	lockoutPolicyRepo := postgres.NewLoginLockoutPolicyRepository(db)

	// 6. Khởi tạo adapter hạ tầng.
	emailSender := platformemail.NewSender(cfg.Email)

	// 7. Khởi tạo usecase.
	authService := usecase.NewAuthUsecase(
		userRepo,
		tokenRepo,
		actionTokenRepo,
		lockoutPolicyRepo,
		emailSender,
		jwtManager,
		cfg.Password,
		cfg.Security,
	)

	// 8. Khởi tạo handler.
	accessTokenTTLSec := int64(cfg.JWT.AccessTokenTTL.Seconds())
	authHandler := handler.NewAuthHandler(authService, accessTokenTTLSec)

	// 9. Tạo router.
	router := deliveryhttp.NewRouter(authHandler, jwtManager, redisClient, cfg)

	// 10. Tạo HTTP server.
	server := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	return &App{
		cfg:    cfg,
		server: server,
	}, nil
}

// Run khởi chạy HTTP server và xử lý graceful shutdown.
func (a *App) Run() error {
	// Channel nhận signal để graceful shutdown.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Chạy server trong goroutine riêng.
	errCh := make(chan error, 1)
	go func() {
		slog.Info("auth service is running", "addr", a.server.Addr)
		if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("server error: %w", err)
		}
	}()

	// Chờ signal hoặc lỗi server.
	select {
	case err := <-errCh:
		return err
	case sig := <-quit:
		slog.Info("received shutdown signal", "signal", sig.String())
	}

	// Graceful shutdown với timeout 10 giây.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := a.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown: %w", err)
	}

	slog.Info("auth service stopped gracefully")
	return nil
}
