<!-- Generated: 2026-04-15 | Files scanned: 60+ | Token estimate: ~500 -->

# Dependencies

## Go Module

```
module github.com/base-go/base
go 1.25.0
```

## Direct Dependencies

| Package | Version | Purpose |
|---------|---------|---------|
| `github.com/gin-gonic/gin` | v1.12.0 | HTTP web framework (both services) |
| `github.com/go-redis/redis_rate/v10` | v10.0.1 | Redis-based rate limiter (token bucket) |
| `github.com/golang-jwt/jwt/v5` | v5.3.1 | JWT issue & validation (RS256) |
| `github.com/google/uuid` | v1.6.0 | UUID v4 generation |
| `github.com/redis/go-redis/v9` | v9.18.0 | Redis client (token blacklist, rate limiting) |
| `github.com/stretchr/testify` | v1.11.1 | Test assertions |
| `golang.org/x/crypto` | v0.49.0 | bcrypt password hashing |
| `golang.org/x/time` | v0.15.0 | Rate limiter — Gateway in-memory (token bucket) |
| `gopkg.in/yaml.v3` | v3.0.1 | routes.yaml config parsing |
| `gorm.io/driver/postgres` | v1.6.0 | PostgreSQL driver (pgx) |
| `gorm.io/gorm` | v1.31.1 | ORM |

## Indirect Dependencies (notable)

| Package | Purpose |
|---------|---------|
| `github.com/pquerna/otp` | TOTP 2FA (RFC 6238) |
| `github.com/boombuler/barcode` | QR code generation (used by otp) |
| `github.com/jackc/pgx/v5` | Low-level PostgreSQL driver |
| `github.com/bytedance/sonic` | Fast JSON (Gin) |
| `github.com/cespare/xxhash/v2` | Hash function (go-redis) |

## Infrastructure

| Service | Version | Role |
|---------|---------|------|
| PostgreSQL | 16-alpine | Primary database (auth_db) |
| Redis | 7+ | Token blacklist, per-IP rate limiting (tuỳ chọn, có fallback) |

## Runtime Configuration

### Auth Service (env vars)

| Variable | Default | Description |
|----------|---------|-------------|
| `SERVER_PORT` | 8081 | HTTP listen port |
| `DB_HOST/PORT/USER/PASSWORD/NAME/SSLMODE` | — | PostgreSQL connection |
| `REDIS_HOST` | localhost | Redis server host |
| `REDIS_PORT` | 6379 | Redis server port |
| `REDIS_PASSWORD` | _(empty)_ | Redis password |
| `REDIS_DB` | 0 | Redis DB index |
| `RATE_LIMIT_LOGIN` | 5 | Login attempts per IP per minute |
| `RATE_LIMIT_GENERAL` | 100 | General requests per IP per minute |
| `JWT_PRIVATE_KEY_PATH` | `configs/keys/private.pem` | RS256 private key |
| `JWT_PUBLIC_KEY_PATH` | `configs/keys/public.pem` | RS256 public key |
| `JWT_ACCESS_TOKEN_TTL` | 15m | Access token lifetime |
| `JWT_REFRESH_TOKEN_TTL` | 168h (7d) | Refresh token lifetime |
| `JWT_ISSUER` | auth-service | JWT iss claim |
| `PASSWORD_MIN_LENGTH` | 8 | Minimum password length |

### Gateway Service (env vars)

| Variable | Default | Description |
|----------|---------|-------------|
| `SERVER_PORT` | 8080 | HTTP listen port |
| `JWT_PUBLIC_KEY_PATH` | — | RS256 public key (shared) |
| `ROUTES_CONFIG_PATH` | `configs/routes.yaml` | Route definitions |
| `RATE_LIMIT_ENABLED` | true | Enable per-IP rate limiting |
| `RATE_LIMIT_RPS` | 100 | Requests per second |
| `RATE_LIMIT_BURST` | 200 | Burst size |
| `PROXY_TIMEOUT` | 30s | Upstream request timeout |

## External Integrations

| Integration | Where Used | Notes |
|-------------|-----------|-------|
| SMTP Email | `services/auth/internal/platform/email/sender.go` | Email verification, password reset |
| Redis | `services/auth/internal/repository/redisrepo/` | Token blacklist; pkg/middleware for IP rate limiting |

## Docker

- **docker-compose.yml** — 3 services: `postgres`, `auth-service`, `gateway`
- Redis is optional (run separately or add to compose)
- Keys (`configs/keys/`) mounted read-only into containers
- Gateway reads `routes.docker.yaml` (separate from local `routes.yaml`)
