# Runbook — Vận Hành Hệ Thống

> Tài liệu vận hành (operations runbook) cho stack `base-go/base`.  
> Sinh từ source code thực tế. Cập nhật lần cuối: 2026-04-14.

---

## Mục Lục

- [Tổng Quan Services](#tổng-quan-services)
- [Quy Trình Triển Khai](#quy-trình-triển-khai)
- [Health Checks & Monitoring](#health-checks--monitoring)
- [Database Migrations](#database-migrations)
- [Xử Lý Sự Cố Thường Gặp](#xử-lý-sự-cố-thường-gặp)
- [Quy Trình Rollback](#quy-trình-rollback)
- [Cấu Hình Bảo Mật](#cấu-hình-bảo-mật)

---

## Tổng Quan Services

<!-- AUTO-GENERATED -->
| Service | Container | Port | Phụ thuộc | Mô tả |
|---------|-----------|------|-----------|-------|
| PostgreSQL | `base-postgres` | `5432` | — | Database chính cho Auth Service |
| Auth Service | `base-auth` | `8081` | PostgreSQL (healthy) | Xác thực, quản lý user, JWT |
| Gateway | `base-gateway` | `8080` | Auth Service | API Gateway, reverse proxy, rate limit |

### Thứ Tự Khởi Động

```
PostgreSQL  →  Auth Service  →  Gateway
(healthcheck)   (waits for DB)   (waits for Auth)
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
JWT_ACCESS_TOKEN_TTL=15m
JWT_REFRESH_TOKEN_TTL=168h
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
```

### Monitoring Checklist

- [ ] Gateway `:8080/health` trả về 200
- [ ] Auth Service `:8081/health` trả về 200 (bao gồm DB ping)
- [ ] PostgreSQL container `base-postgres` ở trạng thái `healthy`
- [ ] Log không có lỗi cấp `ERROR` liên tục
- [ ] Rate limiting không trigger oan (log gateway)

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

**Xử lý:**
```bash
# Xem cấu hình rate limit hiện tại
docker compose exec gateway env | grep RATE_LIMIT

# Tăng giới hạn tạm thời (không khuyến nghị production)
# → Sửa RATE_LIMIT_RPS và RATE_LIMIT_BURST trong docker-compose.yml
# → docker compose up -d gateway
```

---

### 5. Database full / disk space

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

---

## Cấu Hình Bảo Mật

### Production Checklist

- [ ] `APP_ENV=production` — bật JSON logging, tắt debug mode
- [ ] RSA key pair 4096-bit được mount qua Docker secrets hoặc volume riêng
- [ ] File `.env` và key PEM **không** được commit vào git
- [ ] `DB_PASSWORD` không dùng giá trị default (`postgres`)
- [ ] `DB_SSLMODE=require` khi DB ở server riêng
- [ ] SMTP credentials lưu trong secrets manager
- [ ] Rate limiting (`RATE_LIMIT_ENABLED=true`) đang hoạt động
- [ ] HTTPS termination ở phía load balancer (nginx, caddy, v.v.) — service không tự handle TLS

### File cần bảo vệ

| File | Rủi ro nếu lộ |
|------|---------------|
| `configs/keys/private.pem` | Hacker có thể ký JWT bất kỳ → **chiếm toàn bộ hệ thống** |
| `configs/keys/public.pem` | Ít nguy hiểm nhưng vẫn không cần public |
| `.env` (production) | Lộ DB password, SMTP credentials |

---

## Liên Hệ & Escalation

Nếu sự cố không giải quyết được bằng runbook này:

1. Kiểm tra issue tracker của dự án
2. Xem tài liệu kiến trúc chi tiết: [docs/huong-dan-them-service-moi.md](huong-dan-them-service-moi.md)
3. Ghi lại: thời gian xảy ra, logs liên quan, bước tái hiện
