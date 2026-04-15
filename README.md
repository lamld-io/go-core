# Base Go — API Gateway & Microservices Starter

Mã nguồn cơ bản (Mono-repo) cho một hệ thống backend dựa trên kiến trúc Microservices sử dụng Golang. Dự án bao gồm API Gateway và Auth Service, đóng vai trò như một bộ khung khởi tạo vững chắc cho các dự án mới.

## 🚀 Tính Năng Nổi Bật

- **API Gateway**: Quản lý route, reverse proxy, rate-limiting (Redis-backed).
- **Auth Service**: Xác thực người dùng, JWT (RSA keys), quản lý phiên (session management), token blacklist.
- **Two-Factor Authentication (2FA)**: Xác thực hai yếu tố TOTP theo chuẩn RFC 6238.
- **Session Management**: Quản lý các phiên đăng nhập, cho phép thu hồi phiên từ xa.
- **Redis Integration**: Token blacklist (logout an toàn), per-IP rate limiting cho endpoint nhạy cảm.
- **Clean Architecture**: Tách biệt rõ ràng các tầng (Delivery → Usecase → Domain ← Repository/Platform).
- **PostgreSQL**: Cơ sở dữ liệu mặc định với migration SQL tự động.
- **Docker Ready**: Docker Compose khởi chạy toàn bộ stack chỉ với một lệnh.

## 🛠 Tech Stack

| Thành phần       | Công nghệ                              |
|-----------------|-----------------------------------------|
| Ngôn ngữ        | [Go 1.25](https://go.dev/)             |
| HTTP Framework  | [Gin](https://gin-gonic.com/)          |
| ORM             | [GORM](https://gorm.io/)              |
| Database        | PostgreSQL 16 (Alpine)                 |
| Cache/Queue     | [Redis](https://redis.io/) (go-redis)  |
| Auth            | JWT RSA (`golang-jwt/jwt/v5`)          |
| 2FA             | TOTP (`pquerna/otp`)                   |
| Password        | `golang.org/x/crypto/bcrypt`           |
| Container       | Docker & Docker Compose                |

## 📂 Tổ Chức Mã Nguồn

```text
base/
├── configs/keys/         # RSA key pair dùng chung cho JWT
├── docs/                 # Tài liệu hệ thống
├── pkg/                  # Shared packages
│   ├── apperror/         # AppError → HTTP status tự động
│   ├── jwt/              # JWT Manager (RSA sign & verify)
│   ├── middleware/        # CORS, IP Rate Limiter (Redis)
│   └── response/         # Response helper chuẩn cho Gin
├── services/
│   ├── auth/             # Auth Service (port 8081)
│   │   ├── internal/
│   │   │   ├── domain/           # Entity, interfaces, errors
│   │   │   ├── usecase/          # Business logic
│   │   │   ├── delivery/http/    # Gin handlers, router, middleware
│   │   │   ├── repository/       # PostgreSQL (GORM) + Redis repos
│   │   │   ├── platform/         # Config, DB, logger, email sender
│   │   │   └── bootstrap/        # Dependency wiring
│   │   └── migrations/           # SQL migration files
│   └── gateway/          # API Gateway (port 8080)
├── docker-compose.yml    # PostgreSQL + Auth + Gateway
└── go.mod                # Module: github.com/base-go/base
```

## 🔐 API Endpoints

<!-- AUTO-GENERATED -->
### Auth Service (`/api/v1/auth`)

| Method   | Path                        | Auth     | Mô tả                                    |
|----------|-----------------------------|----------|-------------------------------------------|
| `POST`   | `/register`                 | Không    | Đăng ký tài khoản mới                    |
| `POST`   | `/login`                    | Không    | Đăng nhập (trả token hoặc yêu cầu 2FA)  |
| `POST`   | `/login/2fa`                | Không    | Xác thực mã TOTP khi login 2FA           |
| `POST`   | `/verify-email`             | Không    | Xác thực email bằng token                |
| `POST`   | `/resend-verification-email`| Không    | Gửi lại email xác thực                   |
| `POST`   | `/forgot-password`          | Không    | Yêu cầu reset password                   |
| `POST`   | `/reset-password`           | Không    | Đặt lại mật khẩu bằng token             |
| `POST`   | `/refresh`                  | Không    | Làm mới access token                     |
| `GET`    | `/validate`                 | Không    | Validate JWT (dùng nội bộ bởi Gateway)   |
| `POST`   | `/logout`                   | Bearer   | Đăng xuất (blacklist token)              |
| `GET`    | `/profile`                  | Bearer   | Lấy thông tin user hiện tại              |
| `GET`    | `/sessions`                 | Bearer   | Danh sách phiên đăng nhập đang active    |
| `DELETE` | `/sessions/:id`             | Bearer   | Thu hồi phiên đăng nhập cụ thể           |
| `POST`   | `/2fa/setup`                | Bearer   | Khởi tạo TOTP 2FA (trả secret + QR URL) |
| `POST`   | `/2fa/verify`               | Bearer   | Xác nhận mã OTP để bật 2FA              |
| `GET`    | `/security/lockout-policy`  | Bearer   | Lấy cấu hình lockout policy             |
| `PUT`    | `/security/lockout-policy`  | Bearer   | Cập nhật lockout policy                  |
<!-- AUTO-GENERATED -->

### Gateway (`:8080`)

| Method | Path       | Mô tả                  |
|--------|------------|------------------------|
| `GET`  | `/health`  | Health check            |
| `*`    | `/api/v1/auth/*` | Proxy tới Auth Service |

## 🏁 Bắt Đầu Nhanh (Quick Start)

### 1. Yêu cầu hệ thống

- [Go](https://go.dev/dl/) ≥ 1.25
- [Docker](https://docs.docker.com/get-docker/) & [Docker Compose](https://docs.docker.com/compose/install/)
- [Redis](https://redis.io/) (tuỳ chọn — có fallback in-memory)
- OpenSSL (để sinh RSA keys)

### 2. Thiết lập RSA Keys và Biến Môi Trường

```bash
# Tạo thư mục chứa keys
mkdir -p configs/keys

# Sinh RSA 4096-bit key pair
openssl genrsa -out configs/keys/private.pem 4096
openssl rsa -in configs/keys/private.pem -pubout -out configs/keys/public.pem

# Copy các tệp biến môi trường mẫu
cp services/auth/configs/.env.example services/auth/configs/.env
cp services/gateway/configs/.env.example services/gateway/configs/.env
```

### 3. Khởi động stack

```bash
# Khởi động PostgreSQL + Auth + Gateway
docker compose up -d

# (Tuỳ chọn) Khởi động Redis cho rate limiting và token blacklist
docker run -d --name base-redis -p 6379:6379 redis:7-alpine
```

> **Lưu ý**: Nếu Redis không khả dụng, Auth Service tự động fallback sang in-memory. Rate limiter và token blacklist vẫn hoạt động nhưng mất trạng thái khi restart.

### 4. Kiểm tra Health

```bash
curl http://localhost:8080/health    # Gateway
curl http://localhost:8081/health    # Auth Service
```

### 5. Test nhanh flow đăng ký → đăng nhập

```bash
# Đăng ký
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email": "test@example.com", "password": "MyPassword123", "full_name": "Test User"}'

# Đăng nhập
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "test@example.com", "password": "MyPassword123"}'
```

## 📚 Tài Liệu Bổ Sung

| Tài liệu | Mô tả |
|-----------|--------|
| [Hướng dẫn đóng góp](docs/CONTRIBUTING.md) | Dev setup, code convention, test, PR checklist |
| [Hướng dẫn vận hành (Runbook)](docs/RUNBOOK.md) | Deploy, health check, troubleshooting, rollback |
| [Biến Môi Trường (ENV)](docs/ENV.md) | Danh sách đầy đủ mọi biến `.env` |
| [Thêm Service Mới](docs/huong-dan-them-service-moi.md) | Hướng dẫn chi tiết tạo microservice mới |

## ⚖️ Giấy Phép (License)

Dự án phát hành dưới **MIT License**. Bạn có thể tuỳ ý sao chép, chỉnh sửa và ứng dụng.
