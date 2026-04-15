# Tài Liệu ENV — Biến Môi Trường

> Sinh từ `.env.example` và `config.go` của từng service. Cập nhật lần cuối: 2026-04-15.

---

## Auth Service

**File nguồn:** `services/auth/configs/.env.example` + `services/auth/internal/platform/config/config.go`

<!-- AUTO-GENERATED -->
### App & Server

| Biến | Bắt buộc | Ví dụ | Mô tả |
|------|----------|-------|-------|
| `APP_ENV` | Không | `production` | Môi trường (`development` / `production`). Ảnh hưởng log format (JSON vs text). |
| `SERVER_PORT` | Không | `8081` | Port HTTP của Auth Service. |
| `SERVER_READ_TIMEOUT` | Không | `15s` | Timeout đọc request. Dùng format Go duration: `15s`, `1m`. |
| `SERVER_WRITE_TIMEOUT` | Không | `15s` | Timeout ghi response. |

### Database (PostgreSQL)

| Biến | Bắt buộc | Ví dụ | Mô tả |
|------|----------|-------|-------|
| `DB_HOST` | Có | `localhost` hoặc `postgres` (Docker) | Host PostgreSQL. |
| `DB_PORT` | Không | `5432` | Port PostgreSQL. |
| `DB_USER` | Có | `postgres` | User PostgreSQL. |
| `DB_PASSWORD` | **Có** | _(strong password)_ | Password PostgreSQL. **KHÔNG dùng default trong production.** |
| `DB_NAME` | Không | `auth_db` | Tên database. |
| `DB_SSLMODE` | Không | `disable` / `require` | SSL mode. Dùng `require` khi DB ở server riêng. |

### Redis

Auth Service sử dụng Redis cho **token blacklist** (logout an toàn) và **rate limiting** per-IP trên endpoint nhạy cảm (login, register). Nếu Redis không khả dụng, service tự động fallback sang in-memory — chấp nhận mất trạng thái khi restart.

| Biến | Bắt buộc | Ví dụ | Mô tả |
|------|----------|-------|-------|
| `REDIS_HOST` | Không | `localhost` hoặc `redis` (Docker) | Host Redis server. Default: `localhost`. |
| `REDIS_PORT` | Không | `6379` | Port Redis. Default: `6379`. |
| `REDIS_PASSWORD` | Không | _(trống)_ | Password Redis. Để trống nếu không yêu cầu auth. |
| `REDIS_DB` | Không | `0` | Redis DB index (0–15). Default: `0`. |

### Rate Limiting

Rate limiting per-IP sử dụng Redis (hoặc in-memory fallback). Áp dụng cho các endpoint public để chống brute-force.

| Biến | Bắt buộc | Ví dụ | Mô tả |
|------|----------|-------|-------|
| `RATE_LIMIT_LOGIN` | Không | `5` | Số request login tối đa per IP per phút. Default: `5`. |
| `RATE_LIMIT_GENERAL` | Không | `100` | Số request chung tối đa per IP per phút. Default: `100`. |

### JWT (RSA Keys)

Auth Service dùng **RSA asymmetric keys** (không phải HMAC shared secret):
- **Private key** (`private.pem`): Dùng để **ký** (sign) access token và refresh token.
- **Public key** (`public.pem`): Dùng để **verify** token — cần share với Gateway và các service khác.

| Biến | Bắt buộc | Ví dụ | Mô tả |
|------|----------|-------|-------|
| `JWT_PRIVATE_KEY_PATH` | **Có** | `configs/keys/private.pem` | Đường dẫn RSA private key (PEM format). |
| `JWT_PUBLIC_KEY_PATH` | **Có** | `configs/keys/public.pem` | Đường dẫn RSA public key (PEM format). |
| `JWT_ACCESS_TOKEN_TTL` | Không | `15m` | Thời hạn access token. Ngắn để giảm rủi ro khi lộ. |
| `JWT_REFRESH_TOKEN_TTL` | Không | `168h` | Thời hạn refresh token (7 ngày). |
| `JWT_ISSUER` | Không | `auth-service` | JWT `iss` claim. Phải khớp với `JWT_ISSUER` của Gateway. |

> **Tạo RSA key pair:**
> ```bash
> openssl genrsa -out configs/keys/private.pem 4096
> openssl rsa -in configs/keys/private.pem -pubout -out configs/keys/public.pem
> ```

### Password Policy

| Biến | Bắt buộc | Ví dụ | Mô tả |
|------|----------|-------|-------|
| `PASSWORD_MIN_LENGTH` | Không | `8` | Độ dài tối thiểu. |
| `PASSWORD_REQUIRE_UPPERCASE` | Không | `true` | Bắt buộc ≥1 chữ hoa. |
| `PASSWORD_REQUIRE_LOWERCASE` | Không | `true` | Bắt buộc ≥1 chữ thường. |
| `PASSWORD_REQUIRE_DIGIT` | Không | `true` | Bắt buộc ≥1 chữ số. |
| `PASSWORD_REQUIRE_SPECIAL` | Không | `true` | Bắt buộc ≥1 ký tự đặc biệt. |

### Security Flows

| Biến | Bắt buộc | Ví dụ | Mô tả |
|------|----------|-------|-------|
| `EMAIL_VERIFICATION_TOKEN_TTL` | Không | `24h` | TTL của link xác thực email. |
| `PASSWORD_RESET_TOKEN_TTL` | Không | `30m` | TTL của link đặt lại mật khẩu. Ngắn vì nhạy cảm. |
| `LOGIN_LOCKOUT_MAX_FAILED_ATTEMPTS` | Không | `5` | Số lần login sai tối đa trước khi khoá tài khoản. |
| `LOGIN_LOCKOUT_DURATION` | Không | `15m` | Thời gian khoá tài khoản sau khi vượt ngưỡng. |

### SMTP (Email)

Nếu `SMTP_HOST` để **trống**, service sẽ log email ra stdout thay vì gửi thật (chế độ dev).

| Biến | Bắt buộc | Ví dụ | Mô tả |
|------|----------|-------|-------|
| `SMTP_HOST` | Không | `smtp.gmail.com` | SMTP server host. Để trống = log-only. |
| `SMTP_PORT` | Không | `587` | SMTP port (587=STARTTLS, 465=SSL). |
| `SMTP_USERNAME` | Không | `no-reply@example.com` | SMTP username. |
| `SMTP_PASSWORD` | Không | _(app password)_ | SMTP password hoặc app-specific password. |
| `SMTP_FROM_EMAIL` | Không | `no-reply@example.com` | Địa chỉ email người gửi. |
| `SMTP_FROM_NAME` | Không | `Base Auth` | Tên hiển thị của người gửi. |
<!-- AUTO-GENERATED -->

---

## Gateway Service

**File nguồn:** `services/gateway/configs/.env.example`

<!-- AUTO-GENERATED -->
### App & Server

| Biến | Bắt buộc | Ví dụ | Mô tả |
|------|----------|-------|-------|
| `APP_ENV` | Không | `production` | Môi trường chạy. |
| `SERVER_PORT` | Không | `8080` | Port HTTP của Gateway — entry point chính. |
| `SERVER_READ_TIMEOUT` | Không | `15s` | Timeout đọc request từ client. |
| `SERVER_WRITE_TIMEOUT` | Không | `30s` | Timeout ghi response về client (lớn hơn vì proxy request). |

### JWT (chỉ Public Key)

Gateway **chỉ cần verify** token, không cần private key.

| Biến | Bắt buộc | Ví dụ | Mô tả |
|------|----------|-------|-------|
| `JWT_PUBLIC_KEY_PATH` | **Có** | `configs/keys/public.pem` | Phải là **cùng public key** với Auth Service. |
| `JWT_ISSUER` | Không | `auth-service` | Phải khớp với `JWT_ISSUER` của Auth Service. |

### Routes Configuration

| Biến | Bắt buộc | Ví dụ | Mô tả |
|------|----------|-------|-------|
| `ROUTES_CONFIG_PATH` | Không | `configs/routes.yaml` | Đường dẫn file YAML cấu hình routes proxy. |

**Cấu trúc `routes.yaml`:**
```yaml
routes:
  - prefix: /api/v1/auth        # Path prefix để match
    target: http://localhost:8081 # URL của downstream service
    strip_prefix: false           # Có xoá prefix trước khi forward không
    requires_auth: false          # Cần JWT không (false = public endpoint)
    methods: []                   # Giới hạn HTTP methods (rỗng = tất cả)
```

### Rate Limiting (In-Memory, Per-IP)

| Biến | Bắt buộc | Ví dụ | Mô tả |
|------|----------|-------|-------|
| `RATE_LIMIT_ENABLED` | Không | `true` | Bật/tắt rate limiting. |
| `RATE_LIMIT_RPS` | Không | `100` | Request/giây tối đa per IP (token bucket algorithm). |
| `RATE_LIMIT_BURST` | Không | `200` | Burst size — số request tức thời được phép vượt RPS. |

> **Lưu ý**: Rate limiter của Gateway dùng in-memory (`golang.org/x/time/rate`). Rate limiter của Auth Service endpoint login dùng Redis (`go-redis/redis_rate`).

### Proxy Settings

| Biến | Bắt buộc | Ví dụ | Mô tả |
|------|----------|-------|-------|
| `PROXY_TIMEOUT` | Không | `30s` | Timeout của từng proxy request tới downstream. |
| `PROXY_MAX_IDLE_CONNS` | Không | `100` | Max idle connections trong HTTP connection pool. |
| `PROXY_IDLE_CONN_TIMEOUT` | Không | `90s` | Thời gian giữ idle connection trước khi đóng. |
<!-- AUTO-GENERATED -->

---

## Ví Dụ Cấu Hình Production

```bash
# === Auth Service (.env) ===
APP_ENV=production
SERVER_PORT=8081
DB_HOST=db.internal
DB_PORT=5432
DB_USER=auth_user
DB_PASSWORD=<strong-random-password>
DB_NAME=auth_db
DB_SSLMODE=require
REDIS_HOST=redis.internal
REDIS_PORT=6379
REDIS_PASSWORD=<redis-password>
REDIS_DB=0
RATE_LIMIT_LOGIN=5
RATE_LIMIT_GENERAL=100
JWT_PRIVATE_KEY_PATH=/run/secrets/jwt_private.pem
JWT_PUBLIC_KEY_PATH=/run/secrets/jwt_public.pem
JWT_ACCESS_TOKEN_TTL=15m
JWT_REFRESH_TOKEN_TTL=168h
JWT_ISSUER=auth-service
PASSWORD_MIN_LENGTH=12
PASSWORD_REQUIRE_UPPERCASE=true
PASSWORD_REQUIRE_DIGIT=true
LOGIN_LOCKOUT_MAX_FAILED_ATTEMPTS=5
LOGIN_LOCKOUT_DURATION=15m
SMTP_HOST=smtp.sendgrid.net
SMTP_PORT=587
SMTP_USERNAME=apikey
SMTP_PASSWORD=<sendgrid-api-key>
SMTP_FROM_EMAIL=noreply@yourdomain.com
SMTP_FROM_NAME=YourApp

# === Gateway Service (.env) ===
APP_ENV=production
SERVER_PORT=8080
JWT_PUBLIC_KEY_PATH=/run/secrets/jwt_public.pem
JWT_ISSUER=auth-service
ROUTES_CONFIG_PATH=/app/configs/routes.yaml
RATE_LIMIT_ENABLED=true
RATE_LIMIT_RPS=100
RATE_LIMIT_BURST=200
PROXY_TIMEOUT=30s
```
