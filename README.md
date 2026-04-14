# Base Go — API Gateway & Microservices Starter

Mã nguồn cơ bản (Mono-repo) cho một hệ thống backend dựa trên kiến trúc Microservices sử dụng Golang. Dự án bao gồm API Gateway và Auth Service, đóng vai trò như một bộ khung khởi tạo vững chắc cho các dự án mới.

## 🚀 Tính Năng Nổi Bật

- **API Gateway**: Quản lý route, reverse proxy, và rate-limiting.
- **Auth Service**: Quản lý xác thực người dùng, JWT (RSA keys), cấp phát và xác minh token theo chuẩn bảo mật.
- **Microservices Design**: Triển khai theo nguyên lý "database per service", mỗi service chạy độc lập.
- **Clean Architecture**: Tách biệt rõ ràng các tầng (Delivery → Usecase → Domain ← Repository/Platform).
- **PostgreSQL**: Cơ sở dữ liệu mặc định với cấu hình tối ưu.
- **Docker Ready**: Tích hợp Docker và Docker Compose cho việc khởi chạy toàn bộ stack cục bộ chỉ với một lệnh.

## 🛠 Tech Stack

- **Ngôn ngữ**: [Go 1.25](https://go.dev/)
- **Framework**: [Gin](https://gin-gonic.com/)
- **ORM**: [GORM](https://gorm.io/)
- **Database**: PostgreSQL 16
- **Bảo mật**: JWT (chuẩn asymmetric RSA keys), `golang.org/x/crypto/bcrypt`
- **Containerization**: Docker & Docker Compose

## 📂 Tổ Chức Mã Nguồn

```text
base/
├── configs/            # Chứa RSA keys dùng chung cho JWT
├── docs/               # Tài liệu chi tiết của hệ thống (Quy trình đóng góp, môi trường, vận hành)
├── pkg/                # Các package dùng chung (error handling, middleware, jwt, response helpers)
├── services/           # Nơi chứa mã nguồn các microservices
│   ├── auth/           # Dịch vụ xác thực và cấp quyền (Port: 8081)
│   └── gateway/        # API Gateway điều phối requests (Port: 8080)
├── docker-compose.yml  # File cấu hình khởi động toàn bộ stack
└── go.mod              # Module github.com/base-go/base
```

## 🏁 Bắt Đầu Nhanh (Quick Start)

### 1. Yêu cầu hệ thống

- [Go](https://go.dev/dl/) ≥ 1.25
- [Docker](https://docs.docker.com/get-docker/) & [Docker Compose](https://docs.docker.com/compose/install/)
- Make hoặc PowerShell

### 2. Thiết lập RSA Keys và Biến Môi Trường

```bash
# Tạo thư mục chứa keys
mkdir -p configs/keys

# Sinh public/private key (yêu cầu OpenSSL)
openssl genrsa -out configs/keys/private.pem 4096
openssl rsa -in configs/keys/private.pem -pubout -out configs/keys/public.pem

# Copy các tệp biến môi trường mẫu
cp services/auth/configs/.env.example services/auth/configs/.env
cp services/gateway/configs/.env.example services/gateway/configs/.env
```

### 3. Kích hoạt toàn bộ ứng dụng bằng Docker Compose

```bash
# Khởi động ở chế độ background
docker compose up -d
```

### 4. Kiểm tra sức khoẻ hệ thống (Health Check)

```bash
curl http://localhost:8080/health
```
Hệ thống sẽ hoạt động nếu nhận được mã trạng thái 200 OK.

## 📚 Tài Liệu Bổ Sung

Để tìm hiểu sâu hơn về nội bộ dự án, vui lòng tham khảo các tài liệu chuyên biệt dưới đây:

- **[Hướng dẫn đóng góp (Contributing Guide)](docs/CONTRIBUTING.md)**: Cách format mã, cấu trúc và viết test.
- **[Hướng dẫn vận hành (Runbook)](docs/RUNBOOK.md)**: Checklist triển khai production, theo dõi hệ thống, giải quyết các sự cố thường gặp.
- **[Biến Môi Trường (ENV Docs)](docs/ENV.md)**: Danh sách đầy đủ và phân loại mọi tham số cấp phát môi trường (`.env`).
- **[Quy Trình Tạo Service Mới](docs/huong-dan-them-service-moi.md)**: Cẩm nang bước-từng-bước để nối thêm một microservice mới cùng framework này.

## ⚖️ Giấy Phép (License)

Dự án phát hành dưới **MIT License**. Bạn có thể tuỳ ý sao chép, chỉnh sửa và ứng dụng.
