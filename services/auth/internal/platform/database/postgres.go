package database

import (
	"fmt"
	"log/slog"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Config chứa thông số kết nối database.
type Config struct {
	DSN             string
	MaxIdleConns    int
	MaxOpenConns    int
	ConnMaxLifetime time.Duration
}

// NewPostgresDB tạo kết nối GORM tới PostgreSQL.
func NewPostgresDB(cfg Config) (*gorm.DB, error) {
	gormLogger := logger.Default.LogMode(logger.Warn)

	db, err := gorm.Open(postgres.Open(cfg.DSN), &gorm.Config{
		Logger:                 gormLogger,
		SkipDefaultTransaction: true, // Tăng performance, dùng explicit transaction khi cần.
	})
	if err != nil {
		return nil, fmt.Errorf("connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get underlying sql.DB: %w", err)
	}

	// Connection pool settings.
	maxIdle := cfg.MaxIdleConns
	if maxIdle == 0 {
		maxIdle = 10
	}
	maxOpen := cfg.MaxOpenConns
	if maxOpen == 0 {
		maxOpen = 100
	}
	connMaxLife := cfg.ConnMaxLifetime
	if connMaxLife == 0 {
		connMaxLife = time.Hour
	}

	sqlDB.SetMaxIdleConns(maxIdle)
	sqlDB.SetMaxOpenConns(maxOpen)
	sqlDB.SetConnMaxLifetime(connMaxLife)

	// Ping để đảm bảo kết nối hoạt động.
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	slog.Info("connected to PostgreSQL", "dsn_host", maskDSN(cfg.DSN))
	return db, nil
}

// Close đóng kết nối database.
func Close(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// maskDSN ẩn password trong DSN để log an toàn.
func maskDSN(dsn string) string {
	// Chỉ log host, không log password.
	return "[masked]"
}
