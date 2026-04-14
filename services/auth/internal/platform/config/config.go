package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config chứa toàn bộ cấu hình cho Auth Service.
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	JWT      JWTConfig
	Password PasswordPolicy
	Security SecurityConfig
	Email    EmailConfig
}

// ServerConfig cấu hình HTTP server.
type ServerConfig struct {
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

// DatabaseConfig cấu hình kết nối PostgreSQL.
type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// DSN trả về connection string cho PostgreSQL.
func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		d.Host, d.Port, d.User, d.Password, d.DBName, d.SSLMode,
	)
}

// JWTConfig cấu hình JWT.
type JWTConfig struct {
	PrivateKeyPath  string
	PublicKeyPath   string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
	Issuer          string
}

// PasswordPolicy cấu hình yêu cầu mật khẩu — có thể tuỳ chỉnh qua env.
type PasswordPolicy struct {
	MinLength        int
	RequireUppercase bool
	RequireLowercase bool
	RequireDigit     bool
	RequireSpecial   bool
}

// SecurityConfig cấu hình các flow bảo mật nghiệp vụ.
type SecurityConfig struct {
	EmailVerificationTokenTTL time.Duration
	PasswordResetTokenTTL     time.Duration
	LockoutPolicy             LockoutPolicyConfig
}

// LockoutPolicyConfig là default policy khi DB chưa có policy override.
type LockoutPolicyConfig struct {
	MaxFailedAttempts int
	LockDuration      time.Duration
}

// EmailConfig cấu hình SMTP gửi email.
type EmailConfig struct {
	SMTPHost  string
	SMTPPort  int
	Username  string
	Password  string
	FromEmail string
	FromName  string
}

// Load đọc cấu hình từ biến môi trường.
func Load() (*Config, error) {
	cfg := &Config{
		Server: ServerConfig{
			Port:         getEnv("SERVER_PORT", "8081"),
			ReadTimeout:  getDurationEnv("SERVER_READ_TIMEOUT", 15*time.Second),
			WriteTimeout: getDurationEnv("SERVER_WRITE_TIMEOUT", 15*time.Second),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "postgres"),
			DBName:   getEnv("DB_NAME", "auth_db"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
		JWT: JWTConfig{
			PrivateKeyPath:  getEnv("JWT_PRIVATE_KEY_PATH", "configs/keys/private.pem"),
			PublicKeyPath:   getEnv("JWT_PUBLIC_KEY_PATH", "configs/keys/public.pem"),
			AccessTokenTTL:  getDurationEnv("JWT_ACCESS_TOKEN_TTL", 15*time.Minute),
			RefreshTokenTTL: getDurationEnv("JWT_REFRESH_TOKEN_TTL", 7*24*time.Hour),
			Issuer:          getEnv("JWT_ISSUER", "auth-service"),
		},
		Password: PasswordPolicy{
			MinLength:        getIntEnv("PASSWORD_MIN_LENGTH", 8),
			RequireUppercase: getBoolEnv("PASSWORD_REQUIRE_UPPERCASE", false),
			RequireLowercase: getBoolEnv("PASSWORD_REQUIRE_LOWERCASE", false),
			RequireDigit:     getBoolEnv("PASSWORD_REQUIRE_DIGIT", false),
			RequireSpecial:   getBoolEnv("PASSWORD_REQUIRE_SPECIAL", false),
		},
		Security: SecurityConfig{
			EmailVerificationTokenTTL: getDurationEnv("EMAIL_VERIFICATION_TOKEN_TTL", 24*time.Hour),
			PasswordResetTokenTTL:     getDurationEnv("PASSWORD_RESET_TOKEN_TTL", 30*time.Minute),
			LockoutPolicy: LockoutPolicyConfig{
				MaxFailedAttempts: getIntEnv("LOGIN_LOCKOUT_MAX_FAILED_ATTEMPTS", 5),
				LockDuration:      getDurationEnv("LOGIN_LOCKOUT_DURATION", 15*time.Minute),
			},
		},
		Email: EmailConfig{
			SMTPHost:  getEnv("SMTP_HOST", ""),
			SMTPPort:  getIntEnv("SMTP_PORT", 587),
			Username:  getEnv("SMTP_USERNAME", ""),
			Password:  getEnv("SMTP_PASSWORD", ""),
			FromEmail: getEnv("SMTP_FROM_EMAIL", "no-reply@example.com"),
			FromName:  getEnv("SMTP_FROM_NAME", "Base Auth"),
		},
	}

	return cfg, nil
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
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func getBoolEnv(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
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
