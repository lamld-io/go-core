# Hướng Dẫn Đóng Góp (Contributing Guide)

> Sinh từ source code thực tế. Cập nhật lần cuối: 2026-04-15.

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

| Công cụ         | Phiên bản tối thiểu | Ghi chú                                      |
|-----------------|---------------------|----------------------------------------------|
| Go              | 1.25.0              | Xem `go.mod`                                 |
| Docker          | 24+                 | Cần cho PostgreSQL và chạy toàn bộ stack     |
| Docker Compose  | v2+                 | Tích hợp sẵn trong Docker Desktop            |
| Redis           | 7+                  | Tuỳ chọn — có fallback in-memory             |
| `psql` / `migrate` | Tuỳ chọn         | Nếu muốn chạy migration thủ công             |
| OpenSSL         | Bất kỳ              | Cần để sinh RSA key pair cho JWT             |

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
# Chỉnh sửa .env nếu cần (DB password, SMTP, Redis, v.v.)

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

### 4. (Tuỳ chọn) Khởi động Redis

```powershell
# Redis cho token blacklist và rate limiting
docker run -d --name base-redis -p 6379:6379 redis:7-alpine
```

> **Lưu ý**: Nếu Redis không chạy, Auth Service tự động fallback sang in-memory. Rate limiter và token blacklist vẫn hoạt động nhưng mất trạng thái khi service restart.

### 5. Kiểm tra health

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
# Auth Service (cần PostgreSQL đang chạy, Redis tuỳ chọn)
$env:SERVER_PORT="8081"; $env:DB_HOST="localhost"; go run ./services/auth/cmd/auth

# Gateway
$env:SERVER_PORT="8080"; go run ./services/gateway/cmd/gateway
```
<!-- AUTO-GENERATED -->

---

## Biến Môi Trường

Xem danh sách đầy đủ tại [docs/ENV.md](ENV.md). Dưới đây là tóm tắt các biến quan trọng:

<!-- AUTO-GENERATED -->
### Auth Service — Biến quan trọng

| Biến | Bắt buộc | Mô tả | Giá trị mặc định |
|------|----------|-------|-----------------| 
| `DB_HOST` / `DB_PASSWORD` | **Có** | Kết nối PostgreSQL | `localhost` / `postgres` |
| `JWT_PRIVATE_KEY_PATH` | **Có** | RSA private key (PEM) để ký JWT | `configs/keys/private.pem` |
| `JWT_PUBLIC_KEY_PATH` | **Có** | RSA public key (PEM) để verify JWT | `configs/keys/public.pem` |
| `REDIS_HOST` | Không | Host Redis (blacklist, rate limit) | `localhost` |
| `REDIS_PORT` | Không | Port Redis | `6379` |
| `RATE_LIMIT_LOGIN` | Không | Giới hạn request login per IP per phút | `5` |
| `RATE_LIMIT_GENERAL` | Không | Giới hạn request chung per IP per phút | `100` |
| `SMTP_HOST` | Không | SMTP server (trống = log-only) | _(trống)_ |

### Gateway Service — Biến quan trọng

| Biến | Bắt buộc | Mô tả | Giá trị mặc định |
|------|----------|-------|-----------------| 
| `JWT_PUBLIC_KEY_PATH` | **Có** | RSA public key — cùng key với Auth | `configs/keys/public.pem` |
| `RATE_LIMIT_ENABLED` | Không | Bật rate limiting per-IP | `true` |
| `RATE_LIMIT_RPS` / `RATE_LIMIT_BURST` | Không | Request/giây và burst size | `100` / `200` |
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

### Integration Tests (cần DB + Redis)

```powershell
# Khởi động dependencies
docker compose up -d postgres
docker run -d --name base-redis -p 6379:6379 redis:7-alpine

# Chạy integration tests
go test -tags=integration ./...
```

---

## Kiến Trúc Dự Án

Xem chi tiết tại: [docs/huong-dan-them-service-moi.md](huong-dan-them-service-moi.md)

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
│   ├── middleware/           # CORS, IP Rate Limiter (Redis)
│   └── response/             # Response helper chuẩn cho Gin
├── services/
│   ├── auth/                 # Auth Service (port 8081)
│   │   └── internal/
│   │       ├── domain/       # Entity, interfaces, errors
│   │       ├── usecase/      # Business logic
│   │       ├── delivery/http/ # Gin handlers, router
│   │       ├── repository/   # PostgreSQL (GORM) + Redis repos
│   │       ├── platform/     # Config, DB, logger, email
│   │       └── bootstrap/    # Dependency wiring
│   └── gateway/              # API Gateway (port 8080)
└── docs/                     # Tài liệu
```

### Quy Tắc Phụ Thuộc (Clean Architecture)

```
delivery (Gin handler) → usecase → domain ← repository (GORM + Redis)
                                        ↑
                              platform (config, DB, Redis, logger, email)
```

- **domain**: Không import bất kỳ framework nào ngoài stdlib và `pkg/apperror`
- **usecase**: Chỉ import domain interfaces
- **repository**: Implement domain interfaces, dùng GORM (PostgreSQL) và go-redis
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
- [ ] Kiểm tra Redis fallback hoạt động khi Redis down

---

## Quy Trình PR

1. Fork repo → tạo branch `feat/tên-feature` hoặc `fix/tên-bug`
2. Implement + viết test
3. `go test ./... && go vet ./...` pass
4. Mở PR vào `main`, điền đầy đủ mô tả
5. Reviewer sẽ phản hồi trong vòng 48h
