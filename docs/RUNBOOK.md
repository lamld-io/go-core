# Runbook — Vận Hành Hệ Thống

> Tài liệu vận hành (operations runbook) cho stack `base-go/base`.  
> Sinh từ source code thực tế. Cập nhật lần cuối: 2026-04-15.

---

## Mục Lục

- [Tổng Quan Services](#tổng-quan-services)
- [Quy Trình Triển Khai](#quy-trình-triển-khai)
- [Health Checks & Monitoring](#health-checks--monitoring)
- [API Endpoints](#api-endpoints)
- [Database Migrations](#database-migrations)
- [Redis Operations](#redis-operations)
- [Xử Lý Sự Cố Thường Gặp](#xử-lý-sự-cố-thường-gặp)
- [Quy Trình Rollback](#quy-trình-rollback)
- [Cấu Hình Bảo Mật](#cấu-hình-bảo-mật)

---

## Tổng Quan Services

<!-- AUTO-GENERATED -->
| Service | Container | Port | Phụ thuộc | Mô tả |
|---------|-----------|------|-----------|-------|
| PostgreSQL | `base-postgres` | `5432` | — | Database chính cho Auth Service |
| Redis | _(tuỳ chọn)_ | `6379` | — | Token blacklist, rate limiting per-IP |
| Auth Service | `base-auth` | `8081` | PostgreSQL (healthy), Redis (tuỳ chọn) | Xác thực, user, JWT, 2FA, sessions |
| Gateway | `base-gateway` | `8080` | Auth Service | API Gateway, reverse proxy, rate limit |

### Thứ Tự Khởi Động

```
PostgreSQL  →  Redis (tuỳ chọn)  →  Auth Service  →  Gateway
(healthcheck)                        (waits for DB)   (waits for Auth)
```

### Dependency Graph

```
Gateway (:8080) ──proxy──▶ Auth Service (:8081) ──▶ PostgreSQL (:5432)
                                    │
                                    └──▶ Redis (:6379) [tuỳ chọn]
                                          ├── Token Blacklist (logout)
                                          └── Rate Limiter (login endpoint)
```
<!-- AUTO-GENERATED -->

---

## Quy Trình Triển Khai

### Production Deployment (Docker Compose)

```bash
# 1. Pull image mới nhất hoặc build lại
docker compose build --no-cache

# 2. Dừng service cần update (zero-downtime không khả dụng với compose đơn)
docker compose stop auth-service gateway

# 3. Chạy migration trước khi restart service mới (xem phần Database Migrations)
# ...

# 4. Khởi động lại với image mới
docker compose up -d auth-service gateway

# 5. Kiểm tra health
docker compose ps
curl http://localhost:8081/health
curl http://localhost:8080/health
```

### Cập Nhật Chỉ Một Service

```bash
# Build và restart chỉ auth-service
docker compose build auth-service
docker compose up -d --no-deps auth-service

# Xem log để xác nhận startup thành công
docker compose logs -f auth-service --tail=50
```

### Biến Môi Trường Production

Đảm bảo các biến sau được set trên server production:

```bash
# Auth Service — BẮT BUỘC override trong production
APP_ENV=production
DB_PASSWORD=<strong_password>          # KHÔNG dùng default 'postgres'
JWT_PRIVATE_KEY_PATH=/run/secrets/jwt_private.pem
JWT_PUBLIC_KEY_PATH=/run/secrets/jwt_public.pem
REDIS_HOST=redis.internal
REDIS_PASSWORD=<redis_password>
SMTP_HOST=<smtp_server>
SMTP_PASSWORD=<smtp_password>

# Gateway — BẮT BUỘC override trong production
APP_ENV=production
JWT_PUBLIC_KEY_PATH=/run/secrets/jwt_public.pem
RATE_LIMIT_RPS=100                     # Điều chỉnh theo traffic thực tế
```

---

## Health Checks & Monitoring

### Endpoints

<!-- AUTO-GENERATED -->
| Endpoint | Method | Service | Auth | Mô tả |
|----------|--------|---------|------|-------|
| `GET /health` | GET | Auth Service (`:8081`) | Không | Kiểm tra service + DB alive |
| `GET /health` | GET | Gateway (`:8080`) | Không | Kiểm tra gateway alive |
| `GET /api/v1/auth/validate` | GET | Auth Service (qua Gateway) | Bearer token | Validate JWT token |
<!-- AUTO-GENERATED -->

### Kiểm Tra Nhanh

```bash
# Auth Service health
curl -s http://localhost:8081/health | python3 -m json.tool

# Gateway health (qua proxy)
curl -s http://localhost:8080/health | python3 -m json.tool

# Validate token (thay <TOKEN> bằng access token thật)
curl -s -H "Authorization: Bearer <TOKEN>" http://localhost:8080/api/v1/auth/validate

# Kiểm tra Redis connectivity (nếu chạy riêng)
redis-cli -h localhost ping   # Mong đợi: PONG
```

### Monitoring Checklist

- [ ] Gateway `:8080/health` trả về 200
- [ ] Auth Service `:8081/health` trả về 200 (bao gồm DB ping)
- [ ] PostgreSQL container `base-postgres` ở trạng thái `healthy`
- [ ] Redis container/server trả về `PONG` cho `PING`
- [ ] Log không có lỗi cấp `ERROR` liên tục
- [ ] Rate limiting không trigger oan (log auth-service và gateway)
- [ ] Token blacklist hoạt động (logout rồi dùng lại token → 401)

### Xem Logs

```bash
# Logs tất cả services
docker compose logs -f

# Chỉ auth-service, 100 dòng cuối
docker compose logs --tail=100 auth-service

# Chỉ lỗi (lọc theo pattern)
docker compose logs auth-service 2>&1 | grep -i "error\|fatal"

# Logs theo thời gian
docker compose logs --since=1h auth-service
```

---

## API Endpoints

<!-- AUTO-GENERATED -->
### Auth Service (`/api/v1/auth`)

Tất cả endpoint public được rate-limit per-IP qua Redis (fallback in-memory).

**Public endpoints** (không cần JWT):

| Method | Path | Mô tả |
|--------|------|-------|
| `POST` | `/register` | Đăng ký tài khoản mới |
| `POST` | `/login` | Đăng nhập (trả `TokenPair` hoặc `requires_2fa: true` + `temp_token`) |
| `POST` | `/login/2fa` | Xác thực mã TOTP khi login 2FA (cần `temp_token` + `code`) |
| `POST` | `/verify-email` | Xác thực email bằng token |
| `POST` | `/resend-verification-email` | Gửi lại email xác thực |
| `POST` | `/forgot-password` | Yêu cầu reset password |
| `POST` | `/reset-password` | Đặt lại mật khẩu bằng token |
| `POST` | `/refresh` | Làm mới access token từ refresh token |
| `GET`  | `/validate` | Validate JWT token (dùng nội bộ bởi Gateway) |

**Protected endpoints** (cần Bearer JWT access token):

| Method   | Path | Mô tả |
|----------|------|-------|
| `POST`   | `/logout` | Đăng xuất — blacklist access token + revoke refresh token |
| `GET`    | `/profile` | Lấy thông tin user hiện tại |
| `GET`    | `/sessions` | Danh sách phiên đăng nhập (refresh token) đang active |
| `DELETE` | `/sessions/:id` | Thu hồi phiên đăng nhập cụ thể (IDOR-safe: kiểm tra userID) |
| `POST`   | `/2fa/setup` | Khởi tạo TOTP 2FA (trả `secret` + `secret_url` cho QR code) |
| `POST`   | `/2fa/verify` | Xác thực mã OTP để bật 2FA cho tài khoản |
| `GET`    | `/security/lockout-policy` | Lấy cấu hình lockout policy |
| `PUT`    | `/security/lockout-policy` | Cập nhật lockout policy (max attempts, lock duration) |
<!-- AUTO-GENERATED -->

---

## Database Migrations

Auth service quản lý migrations thủ công bằng các file SQL đánh số trong `services/auth/migrations/`.

### Danh Sách Migrations Hiện Tại

<!-- AUTO-GENERATED -->
| Migration | Mô tả |
|-----------|-------|
| `000001_create_users_table` | Tạo bảng `users` — lưu thông tin tài khoản |
| `000002_create_refresh_tokens_table` | Tạo bảng `refresh_tokens` |
| `000003_add_security_fields_to_users_table` | Thêm fields lockout vào `users` |
| `000004_create_action_tokens_table` | Tạo bảng `action_tokens` (verify email, reset password) |
| `000005_create_login_lockout_policies_table` | Tạo bảng `login_lockout_policies` |
<!-- AUTO-GENERATED -->

> **Lưu ý**: Auth Service cũng dùng GORM `AutoMigrate` trong `bootstrap/app.go` để tạo bảng tự động khi khởi động. SQL migration files dùng cho `golang-migrate` CLI trên production.

### Chạy Migration Thủ Công

> **Yêu cầu**: Cài đặt [`golang-migrate`](https://github.com/golang-migrate/migrate) CLI

```bash
# Up — áp dụng tất cả migration chưa chạy
migrate -path services/auth/migrations \
        -database "postgres://postgres:postgres@localhost:5432/auth_db?sslmode=disable" \
        up

# Down — rollback 1 migration
migrate -path services/auth/migrations \
        -database "postgres://postgres:postgres@localhost:5432/auth_db?sslmode=disable" \
        down 1

# Kiểm tra version hiện tại
migrate -path services/auth/migrations \
        -database "postgres://postgres:postgres@localhost:5432/auth_db?sslmode=disable" \
        version
```

### Tạo Migration Mới

```bash
# Đặt tên mô tả rõ ràng
migrate create -ext sql -dir services/auth/migrations -seq add_user_avatar
# Sinh ra:
# services/auth/migrations/000006_add_user_avatar.up.sql
# services/auth/migrations/000006_add_user_avatar.down.sql
```

> **Quy tắc**: File `.up.sql` PHẢI có file `.down.sql` tương ứng để rollback được.

---

## Redis Operations

### Kiểm Tra Token Blacklist

```bash
# Xem tất cả key blacklist (pattern: bl:*)
redis-cli KEYS "bl:*"

# Kiểm tra một token cụ thể có bị blacklist không
redis-cli EXISTS "bl:<token_id>"

# Xem TTL còn lại
redis-cli TTL "bl:<token_id>"
```

### Xoá Token Blacklist (khẩn cấp)

```bash
# Xoá blacklist của một token cụ thể (cho phép dùng lại)
redis-cli DEL "bl:<token_id>"

# ⚠️ XOÁ TOÀN BỘ blacklist (chỉ dùng khi khẩn cấp)
redis-cli KEYS "bl:*" | xargs redis-cli DEL
```

### Rate Limiter

```bash
# Xem rate limit counter cho một IP
redis-cli KEYS "rl:*"

# Reset rate limit cho một IP cụ thể (unblock)
redis-cli DEL "rl:<ip_address>"
```

### Redis Fallback

Nếu Redis down, Auth Service sẽ:
- **Token Blacklist**: Fallback in-memory → token đã blacklist sẽ bị "quên" khi service restart
- **Rate Limiter**: Fallback in-memory → counter reset khi restart
- **Log**: In ra `WARN` level: `"failed to connect to redis, rate limiter will use in-memory fallback"`

---

## Xử Lý Sự Cố Thường Gặp

### 1. Auth Service không khởi động — lỗi kết nối DB

**Triệu chứng:**
```
failed to connect to database: dial tcp localhost:5432: connect: connection refused
```

**Nguyên nhân & Xử lý:**
```bash
# Kiểm tra PostgreSQL có đang chạy không
docker compose ps postgres

# Nếu không healthy → xem logs
docker compose logs postgres

# Khởi động lại PostgreSQL
docker compose restart postgres

# Sau khi PostgreSQL healthy, restart auth-service
docker compose restart auth-service
```

---

### 2. JWT errors — Token invalid / signature verification failed

**Triệu chứng:** API trả về `401 TOKEN_INVALID` dù token còn hạn.

**Nguyên nhân:** Key pair không khớp giữa Auth Service và Gateway.

**Xử lý:**
```bash
# Kiểm tra path key của Auth Service
docker compose exec auth-service ls -la /app/configs/keys/

# Kiểm tra path key của Gateway
docker compose exec gateway ls -la /app/configs/keys/

# Đảm bảo cùng public.pem được mount vào cả 2 service
# Xem docker-compose.yml — cả 2 đều mount ./configs/keys vào /app/configs/keys
```

---

### 3. SMTP — Email không được gửi

**Triệu chứng:** Register/ForgotPassword thành công nhưng user không nhận được email.

**Xử lý Local Dev:**
```bash
# Log sẽ chứa nội dung email thay vì gửi thật
docker compose logs auth-service | grep -i "email\|smtp\|send"
```

**Xử lý Production:**
```bash
# Kiểm tra SMTP config
docker compose exec auth-service env | grep SMTP

# Test SMTP kết nối
telnet $SMTP_HOST $SMTP_PORT
```

---

### 4. Rate limiting — 429 Too Many Requests

**Triệu chứng:** Client nhận `429 RATE_LIMITED`.

**Nguyên nhân:** Có 2 tầng rate limiting:
1. **Auth Service** (Redis-backed): Giới hạn `/login`, `/register` endpoints — `RATE_LIMIT_LOGIN=5` req/phút/IP
2. **Gateway** (In-memory): Giới hạn toàn cục — `RATE_LIMIT_RPS=100` req/giây/IP

**Xử lý:**
```bash
# Kiểm tra cấu hình rate limit Auth Service
docker compose exec auth-service env | grep RATE_LIMIT

# Kiểm tra cấu hình rate limit Gateway
docker compose exec gateway env | grep RATE_LIMIT

# Reset rate limit cho một IP bị block (Redis)
redis-cli DEL "rl:<blocked_ip>"

# Tăng giới hạn tạm thời (không khuyến nghị production)
# → Sửa RATE_LIMIT_LOGIN, RATE_LIMIT_RPS trong env
# → Restart service
```

---

### 5. Redis down — Token blacklist / rate limiter mất tác dụng

**Triệu chứng:** Sau logout, token cũ vẫn hoạt động. Hoặc rate limiter không block.

**Nguyên nhân:** Redis không khả dụng, service đang dùng in-memory fallback.

**Xử lý:**
```bash
# Kiểm tra Redis connectivity
redis-cli ping

# Xem log auth-service có warning không
docker compose logs auth-service | grep -i "redis\|fallback"

# Nếu Redis container bị stop
docker start base-redis

# Restart auth-service để reconnect Redis
docker compose restart auth-service
```

**Lưu ý quan trọng:** Sau khi Redis restore, in-memory blacklist sẽ bị mất. Các token đã logout trước đó có thể hoạt động trở lại cho đến khi hết hạn tự nhiên.

---

### 6. 2FA — User bị khoá không vào được

**Triệu chứng:** User đã bật 2FA nhưng mất authenticator app, không có mã recovery.

**Xử lý (admin):**
```bash
# Kết nối PostgreSQL
docker compose exec postgres psql -U postgres auth_db

# Kiểm tra user
SELECT id, email, is_2fa_enabled, totp_secret FROM users WHERE email = '<user_email>';

# Tắt 2FA cho user (reset TOTP secret)
UPDATE users SET is_2fa_enabled = false, totp_secret = NULL WHERE email = '<user_email>';
```

---

### 7. Database full / disk space

**Triệu chứng:** `ERROR: could not write to file: No space left on device`

**Xử lý:**
```bash
# Kiểm tra dung lượng
docker system df

# Cleanup images/containers không dùng
docker system prune -f

# Xem kích thước volume PostgreSQL
docker volume inspect base_postgres_data
```

---

## Quy Trình Rollback

### Rollback Service (Docker Image)

```bash
# Tag image hiện tại trước khi deploy
docker tag base-auth:latest base-auth:backup-$(date +%Y%m%d)

# Sau khi deploy thất bại → rollback về bản cũ
docker compose stop auth-service
docker tag base-auth:backup-<date> base-auth:latest
docker compose up -d auth-service
```

### Rollback Database Migration

```bash
# Rollback 1 migration
migrate -path services/auth/migrations \
        -database "postgres://postgres:postgres@localhost:5432/auth_db?sslmode=disable" \
        down 1

# Rollback về version cụ thể (ví dụ: version 3)
migrate -path services/auth/migrations \
        -database "postgres://postgres:postgres@localhost:5432/auth_db?sslmode=disable" \
        goto 3
```

> **CẢNH BÁO**: Rollback migration có thể xoá data. Luôn backup DB trước:
> ```bash
> docker compose exec postgres pg_dump -U postgres auth_db > backup_$(date +%Y%m%d_%H%M%S).sql
> ```

### Rollback Redis Data

```bash
# Redis không cần rollback phức tạp — data là ephemeral (TTL-based)
# Nếu cần flush toàn bộ Redis DB:
redis-cli -n 0 FLUSHDB
```

---

## Cấu Hình Bảo Mật

### Production Checklist

- [ ] `APP_ENV=production` — bật JSON logging, tắt debug mode
- [ ] RSA key pair 4096-bit được mount qua Docker secrets hoặc volume riêng
- [ ] File `.env` và key PEM **không** được commit vào git
- [ ] `DB_PASSWORD` không dùng giá trị default (`postgres`)
- [ ] `DB_SSLMODE=require` khi DB ở server riêng
- [ ] SMTP credentials lưu trong secrets manager
- [ ] Rate limiting hoạt động cả 2 tầng (Auth: Redis, Gateway: in-memory)
- [ ] Redis password được set trong production (`REDIS_PASSWORD`)
- [ ] Token blacklist (Redis) đang hoạt động — test: logout → request → 401
- [ ] HTTPS termination ở phía load balancer (nginx, caddy, v.v.) — service không tự handle TLS
- [ ] 2FA secret (`totp_secret`) được lưu encrypted trong DB (hoặc ít nhất DB encrypton at rest)

### File cần bảo vệ

| File | Rủi ro nếu lộ |
|------|---------------|
| `configs/keys/private.pem` | Hacker có thể ký JWT bất kỳ → **chiếm toàn bộ hệ thống** |
| `configs/keys/public.pem` | Ít nguy hiểm nhưng vẫn không cần public |
| `.env` (production) | Lộ DB password, SMTP credentials, Redis password |
| User.TOTPSecret (DB) | Hacker có thể bypass 2FA nếu biết TOTP secret |

---

## Liên Hệ & Escalation

Nếu sự cố không giải quyết được bằng runbook này:

1. Kiểm tra issue tracker của dự án
2. Xem tài liệu kiến trúc chi tiết: [docs/huong-dan-them-service-moi.md](huong-dan-them-service-moi.md)
3. Ghi lại: thời gian xảy ra, logs liên quan, bước tái hiện
