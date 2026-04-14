# Hướng Dẫn Đóng Góp (Contributing Guide)

> Sinh từ source code thực tế. Cập nhật lần cuối: 2026-04-14.

## Mục Lục

- [Yêu Cầu Môi Trường](#yêu-cầu-môi-trường)
- [Thiết Lập Môi Trường Dev](#thiết-lập-môi-trường-dev)
- [Scripts Có Sẵn](#scripts-có-sẵn)
- [Biến Môi Trường](#biến-môi-trường)
- [Chạy Test](#chạy-test)
- [Kiến Trúc Dự Án](#kiến-trúc-dự-án)
- [Quy Tắc Code Style](#quy-tắc-code-style)
- [Quy Trình PR](#quy-trình-pr)

---

## Yêu Cầu Môi Trường

| Công cụ      | Phiên bản tối thiểu | Ghi chú                                      |
|-------------|---------------------|----------------------------------------------|
| Go          | 1.25.0              | Xem `go.mod`                                 |
| Docker      | 24+                 | Cần cho PostgreSQL và chạy toàn bộ stack     |
| Docker Compose | v2+              | Tích hợp sẵn trong Docker Desktop            |
| `psql` / `migrate` | Tuỳ chọn   | Nếu muốn chạy migration thủ công             |

---

## Thiết Lập Môi Trường Dev

### 1. Clone repo và tạo RSA keys (cần cho JWT)

```powershell
git clone <repo-url>
cd base

# Tạo RSA key pair cho JWT (cần OpenSSL)
mkdir configs\keys
openssl genrsa -out configs\keys\private.pem 4096
openssl rsa -in configs\keys\private.pem -pubout -out configs\keys\public.pem
```

### 2. Cấu hình biến môi trường cho từng service

```powershell
# Auth Service
Copy-Item services\auth\configs\.env.example services\auth\configs\.env
# Chỉnh sửa .env nếu cần (DB password, SMTP, v.v.)

# Gateway Service
Copy-Item services\gateway\configs\.env.example services\gateway\configs\.env
```

### 3. Khởi động toàn bộ stack bằng Docker Compose

```powershell
docker compose up -d
```

Services sẽ khởi động theo thứ tự:
1. **PostgreSQL** (port `5432`) — healthcheck tự động
2. **Auth Service** (port `8081`) — chờ PostgreSQL healthy
3. **Gateway** (port `8080`) — chờ Auth Service

### 4. Kiểm tra health

```powershell
# Gateway (entry point)
Invoke-WebRequest http://localhost:8080/health
# Auth Service trực tiếp
Invoke-WebRequest http://localhost:8081/health
```

---

## Scripts Có Sẵn

> Sinh từ `go.mod`, `Makefile` (nếu có), và `docker-compose.yml`.

<!-- AUTO-GENERATED -->
### Docker Compose

| Lệnh | Mô tả |
|------|-------|
| `docker compose up -d` | Khởi động toàn bộ stack (PostgreSQL, Auth, Gateway) ở background |
| `docker compose up -d postgres` | Chỉ khởi động PostgreSQL |
| `docker compose logs -f auth-service` | Xem log realtime của Auth Service |
| `docker compose logs -f gateway` | Xem log realtime của Gateway |
| `docker compose down` | Dừng và xoá container (giữ volume) |
| `docker compose down -v` | Dừng và xoá cả volume (xoá data DB) |
| `docker compose build` | Build lại tất cả image |
| `docker compose ps` | Xem trạng thái các service |

### Go Commands

| Lệnh | Mô tả |
|------|-------|
| `go test ./...` | Chạy tất cả test trong mono-repo |
| `go test ./services/auth/...` | Chạy test của Auth Service |
| `go test -v -run TestName ./...` | Chạy test cụ thể theo tên |
| `go test -cover ./...` | Chạy test + báo cáo coverage |
| `go build ./services/auth/cmd/auth` | Build Auth Service binary |
| `go build ./services/gateway/cmd/gateway` | Build Gateway binary |
| `go vet ./...` | Kiểm tra lỗi tĩnh |
| `go mod tidy` | Dọn dẹp dependencies |

### Chạy Service Trực Tiếp (không Docker)

```powershell
# Auth Service (cần PostgreSQL đang chạy)
$env:SERVER_PORT="8081"; $env:DB_HOST="localhost"; go run ./services/auth/cmd/auth

# Gateway
$env:SERVER_PORT="8080"; go run ./services/gateway/cmd/gateway
```
<!-- AUTO-GENERATED -->

---

## Biến Môi Trường

<!-- AUTO-GENERATED -->
### Auth Service (`services/auth/configs/.env.example`)

| Biến | Bắt buộc | Mô tả | Giá trị mặc định |
|------|----------|-------|-----------------|
| `APP_ENV` | Không | Môi trường chạy | `development` |
| `SERVER_PORT` | Không | Port HTTP của Auth Service | `8081` |
| `SERVER_READ_TIMEOUT` | Không | Timeout đọc request | `15s` |
| `SERVER_WRITE_TIMEOUT` | Không | Timeout ghi response | `15s` |
| `DB_HOST` | Có | Host PostgreSQL | `localhost` |
| `DB_PORT` | Không | Port PostgreSQL | `5432` |
| `DB_USER` | Có | User PostgreSQL | `postgres` |
| `DB_PASSWORD` | **Có** | Password PostgreSQL | _(trống)_ |
| `DB_NAME` | Không | Tên database | `auth_db` |
| `DB_SSLMODE` | Không | SSL mode kết nối DB | `disable` |
| `JWT_PRIVATE_KEY_PATH` | **Có** | Đường dẫn RSA private key (PEM) | `configs/keys/private.pem` |
| `JWT_PUBLIC_KEY_PATH` | **Có** | Đường dẫn RSA public key (PEM) | `configs/keys/public.pem` |
| `JWT_ACCESS_TOKEN_TTL` | Không | Thời hạn access token | `15m` |
| `JWT_REFRESH_TOKEN_TTL` | Không | Thời hạn refresh token | `168h` (7 ngày) |
| `JWT_ISSUER` | Không | JWT issuer claim | `auth-service` |
| `PASSWORD_MIN_LENGTH` | Không | Độ dài mật khẩu tối thiểu | `8` |
| `PASSWORD_REQUIRE_UPPERCASE` | Không | Bắt buộc chữ hoa | `false` |
| `PASSWORD_REQUIRE_LOWERCASE` | Không | Bắt buộc chữ thường | `false` |
| `PASSWORD_REQUIRE_DIGIT` | Không | Bắt buộc chữ số | `false` |
| `PASSWORD_REQUIRE_SPECIAL` | Không | Bắt buộc ký tự đặc biệt | `false` |
| `EMAIL_VERIFICATION_TOKEN_TTL` | Không | TTL token xác thực email | `24h` |
| `PASSWORD_RESET_TOKEN_TTL` | Không | TTL token đặt lại mật khẩu | `30m` |
| `LOGIN_LOCKOUT_MAX_FAILED_ATTEMPTS` | Không | Số lần đăng nhập sai tối đa | `5` |
| `LOGIN_LOCKOUT_DURATION` | Không | Thời gian khoá tài khoản | `15m` |
| `SMTP_HOST` | Không | SMTP server host | _(trống = log-only)_ |
| `SMTP_PORT` | Không | SMTP server port | `587` |
| `SMTP_USERNAME` | Không | SMTP username | _(trống)_ |
| `SMTP_PASSWORD` | Không | SMTP password | _(trống)_ |
| `SMTP_FROM_EMAIL` | Không | Email gửi đi | `no-reply@example.com` |
| `SMTP_FROM_NAME` | Không | Tên hiển thị email | `Base Auth` |

> **Lưu ý SMTP**: Nếu `SMTP_HOST` để trống, email sẽ được log ra stdout thay vì gửi thật — tiện cho local dev.

### Gateway Service (`services/gateway/configs/.env.example`)

| Biến | Bắt buộc | Mô tả | Giá trị mặc định |
|------|----------|-------|-----------------|
| `APP_ENV` | Không | Môi trường chạy | `development` |
| `SERVER_PORT` | Không | Port HTTP của Gateway | `8080` |
| `SERVER_READ_TIMEOUT` | Không | Timeout đọc request | `15s` |
| `SERVER_WRITE_TIMEOUT` | Không | Timeout ghi response | `30s` |
| `JWT_PUBLIC_KEY_PATH` | **Có** | Đường dẫn RSA public key (PEM) | `configs/keys/public.pem` |
| `JWT_ISSUER` | Không | JWT issuer cần match với Auth | `auth-service` |
| `ROUTES_CONFIG_PATH` | Không | Đường dẫn file routes.yaml | `configs/routes.yaml` |
| `RATE_LIMIT_ENABLED` | Không | Bật rate limiting per-IP | `true` |
| `RATE_LIMIT_RPS` | Không | Requests/giây tối đa | `100` |
| `RATE_LIMIT_BURST` | Không | Burst size | `200` |
| `PROXY_TIMEOUT` | Không | Timeout proxy request | `30s` |
| `PROXY_MAX_IDLE_CONNS` | Không | Max idle connections | `100` |
| `PROXY_IDLE_CONN_TIMEOUT` | Không | Idle connection timeout | `90s` |
<!-- AUTO-GENERATED -->

---

## Chạy Test

### Unit Tests

```powershell
# Tất cả test
go test ./...

# Chỉ Auth Service
go test ./services/auth/...

# Với verbose output
go test -v ./services/auth/...

# Test cụ thể
go test -v -run TestAuthUseCase_Login ./services/auth/...
```

### Test với Coverage

```powershell
# Báo cáo coverage trên terminal
go test -cover ./...

# Sinh HTML report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

### Integration Tests (cần DB)

```powershell
# Khởi động PostgreSQL trước
docker compose up -d postgres

# Chờ DB healthcheck xong rồi chạy
go test -tags=integration ./...
```

---

## Kiến Trúc Dự Án

Xem chi tiết tại: [docs/huong-dan-them-service-moi.md](../docs/huong-dan-them-service-moi.md)

### Cấu Trúc Thư Mục

```
base/
├── go.mod                    # Module: github.com/base-go/base (Go 1.25)
├── go.sum
├── docker-compose.yml        # Orchestrate: PostgreSQL + Auth + Gateway
├── configs/keys/             # RSA key pair dùng chung cho JWT
├── pkg/                      # Shared packages
│   ├── apperror/             # AppError → HTTP status tự động
│   ├── jwt/                  # JWT Manager (RSA sign & verify)
│   ├── middleware/           # CORS middleware dùng chung
│   └── response/             # Response helper chuẩn cho Gin
├── services/
│   ├── auth/                 # Auth Service (port 8081)
│   └── gateway/              # API Gateway (port 8080)
└── docs/                     # Tài liệu
```

### Quy Tắc Phụ Thuộc (Clean Architecture)

```
delivery (Gin handler) → usecase → domain ← repository (GORM)
                                        ↑
                              platform (config, DB, logger)
```

- **domain**: Không import bất kỳ framework nào ngoài stdlib và `pkg/apperror`
- **usecase**: Chỉ import domain interfaces
- **repository**: Implement domain interfaces, dùng GORM
- **delivery**: Gọi usecase qua domain interface, dùng Gin
- **platform**: Cung cấp hạ tầng, không chứa business logic

---

## Quy Tắc Code Style

### Naming Convention

- Package: `lowercase`, không dấu gạch ngang
- File: `snake_case.go`
- Struct/Interface: `PascalCase`
- Function/Method: `PascalCase` (exported), `camelCase` (unexported)
- Error variable: tiền tố `Err` — VD: `ErrUserNotFound`
- Test file: `*_test.go`, test function: `TestXxx_MethodName`

### Conventions

- Mọi method nhận `context.Context` là tham số đầu tiên
- Error wrapping dùng `fmt.Errorf("context: %w", err)`
- Nullable field dùng pointer (`*string`, `*uuid.UUID`)
- UUID làm primary key cho mọi entity
- Logging dùng `log/slog` (structured logger)
- Response JSON chuẩn dùng `pkg/response`

### PR Checklist

- [ ] Code chạy được: `go build ./...`
- [ ] Tất cả test pass: `go test ./...`
- [ ] Không có warning: `go vet ./...`
- [ ] Không commit file `.env` hay key PEM thật
- [ ] Thêm test cho logic mới
- [ ] Comment tiếng Anh cho exported symbols
- [ ] Cập nhật `.env.example` nếu thêm biến môi trường mới

---

## Quy Trình PR

1. Fork repo → tạo branch `feat/tên-feature` hoặc `fix/tên-bug`
2. Implement + viết test
3. `go test ./... && go vet ./...` pass
4. Mở PR vào `main`, điền đầy đủ mô tả
5. Reviewer sẽ phản hồi trong vòng 48h
