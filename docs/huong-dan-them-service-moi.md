# Hướng Dẫn Chi Tiết: Thêm Service Mới Vào Hệ Thống

> Tài liệu này hướng dẫn từng bước cách tạo một microservice mới trong repo `base-go/base`,
> dựa trên kiến trúc và convention đang được sử dụng bởi **auth service** hiện có.
>
> Ví dụ minh hoạ xuyên suốt: tạo **product service** — dịch vụ quản lý sản phẩm.

---

## Mục Lục

1. [Tổng quan kiến trúc hiện tại](#1-tổng-quan-kiến-trúc-hiện-tại)
2. [Chuẩn bị thư mục service mới](#2-chuẩn-bị-thư-mục-service-mới)
3. [Bước 1 — Tạo Domain Layer](#3-bước-1--tạo-domain-layer)
4. [Bước 2 — Tạo Platform Layer (hạ tầng)](#4-bước-2--tạo-platform-layer-hạ-tầng)
5. [Bước 3 — Tạo Repository Layer](#5-bước-3--tạo-repository-layer)
6. [Bước 4 — Tạo Usecase Layer](#6-bước-4--tạo-usecase-layer)
7. [Bước 5 — Tạo Delivery Layer (HTTP)](#7-bước-5--tạo-delivery-layer-http)
8. [Bước 6 — Tạo Bootstrap (wiring tất cả)](#8-bước-6--tạo-bootstrap-wiring-tất-cả)
9. [Bước 7 — Tạo Entrypoint (main.go)](#9-bước-7--tạo-entrypoint-maingo)
10. [Bước 8 — Tạo Migration](#10-bước-8--tạo-migration)
11. [Bước 9 — Tạo File Cấu Hình](#11-bước-9--tạo-file-cấu-hình)
12. [Bước 10 — Tạo Dockerfile](#12-bước-10--tạo-dockerfile)
13. [Bước 11 — Cập Nhật docker-compose.yml](#13-bước-11--cập-nhật-docker-composeyml)
14. [Bước 12 — Đăng Ký Route Trên Gateway](#14-bước-12--đăng-ký-route-trên-gateway)
15. [Bước 13 — Viết Test](#15-bước-13--viết-test)
16. [Bước 14 — Chạy Và Kiểm Tra](#16-bước-14--chạy-và-kiểm-tra)
17. [Checklist tổng kết](#17-checklist-tổng-kết)

---

## 1. Tổng Quan Kiến Trúc Hiện Tại

### Cấu trúc mono-repo

```
base/
├── go.mod                    # Module chung: github.com/base-go/base
├── go.sum
├── docker-compose.yml        # Orchestrate tất cả services
├── configs/                  # Config dùng chung (keys, ...)
│   └── keys/                 # RSA keys cho JWT
├── pkg/                      # Package dùng chung giữa các services
│   ├── apperror/             # Chuẩn hoá lỗi ứng dụng (AppError)
│   ├── jwt/                  # JWT manager (sign & verify)
│   ├── middleware/            # Middleware dùng chung (CORS)
│   └── response/             # Response helper chuẩn cho Gin
├── services/
│   ├── auth/                 # Auth Service (xác thực, user, JWT)
│   └── gateway/              # API Gateway (proxy, rate limit)
└── docs/                     # Tài liệu
```

### Kiến trúc Clean Architecture bên trong mỗi service

```
services/<tên-service>/
├── Dockerfile
├── cmd/<tên-service>/
│   └── main.go                        # Entrypoint
├── configs/
│   └── .env.example                   # Mẫu biến môi trường
├── migrations/                        # SQL migration files
│   ├── 000001_create_xxx.up.sql
│   └── 000001_create_xxx.down.sql
└── internal/
    ├── domain/                        # Tầng lõi: entity, interface, errors
    │   ├── <entity>.go                # Entity (struct thuần Go)
    │   ├── repository.go              # Interface repository
    │   ├── service.go                 # Interface usecase/service
    │   └── errors.go                  # Domain-specific errors
    ├── usecase/                       # Tầng nghiệp vụ
    │   ├── dto/                       # Request/Response DTO
    │   │   └── <feature>_dto.go
    │   ├── <feature>_usecase.go       # Implementation của service interface
    │   └── <feature>_usecase_test.go  # Unit test
    ├── delivery/http/                 # Tầng giao tiếp HTTP
    │   ├── handler/                   # Gin handlers
    │   ├── middleware/                 # Middleware riêng của service
    │   ├── presenter/                 # Chuyển domain → response DTO
    │   └── router.go                  # Đăng ký route
    ├── repository/postgres/           # Tầng truy cập dữ liệu
    │   ├── model/                     # GORM model (persistence)
    │   ├── mapper/                    # Chuyển đổi domain ↔ model
    │   └── <entity>_repository.go     # Implementation repository
    ├── platform/                      # Adapter hạ tầng
    │   ├── config/                    # Load cấu hình từ env
    │   ├── database/                  # Khởi tạo kết nối DB
    │   └── logger/                    # Setup structured logger (slog)
    └── bootstrap/
        └── app.go                     # Wire tất cả dependency, khởi chạy server
```

### Nguyên tắc phụ thuộc (Dependency Rule)

```
delivery (handler, middleware) → usecase → domain ← repository
                                              ↑
                                          platform (config, database, logger)
```

- **domain** là lõi, không phụ thuộc bất kỳ package nào ngoài stdlib và `pkg/apperror`.
- **usecase** chỉ phụ thuộc domain interfaces.
- **repository** implement domain interfaces, phụ thuộc GORM.
- **delivery** gọi usecase qua domain interface, phụ thuộc Gin.
- **platform** cung cấp hạ tầng (config, database, logger), không chứa logic nghiệp vụ.

### Công nghệ sử dụng

| Thành phần   | Công nghệ              |
|-------------|------------------------|
| HTTP        | Gin                    |
| ORM         | GORM                   |
| Database    | PostgreSQL 16          |
| Config      | Biến môi trường (env)  |
| Logging     | `log/slog`             |
| Auth        | JWT (RSA, golang-jwt)  |
| Error       | `pkg/apperror`         |
| Response    | `pkg/response`         |
| UUID        | `github.com/google/uuid` |
| Container   | Docker, Docker Compose |

---

## 2. Chuẩn Bị Thư Mục Service Mới

Tạo cấu trúc thư mục đầy đủ cho service mới. Ví dụ cho **product service**:

```powershell
# Tạo tất cả thư mục cần thiết
mkdir -p services/product/cmd/product
mkdir -p services/product/configs
mkdir -p services/product/migrations
mkdir -p services/product/internal/domain
mkdir -p services/product/internal/usecase/dto
mkdir -p services/product/internal/delivery/http/handler
mkdir -p services/product/internal/delivery/http/middleware
mkdir -p services/product/internal/delivery/http/presenter
mkdir -p services/product/internal/repository/postgres/model
mkdir -p services/product/internal/repository/postgres/mapper
mkdir -p services/product/internal/platform/config
mkdir -p services/product/internal/platform/database
mkdir -p services/product/internal/platform/logger
mkdir -p services/product/internal/bootstrap
```

> **Lưu ý**: Tất cả service đều nằm trong cùng 1 Go module (`github.com/base-go/base`) nên **không cần** tạo `go.mod` riêng.

---

## 3. Bước 1 — Tạo Domain Layer

Domain layer là **trung tâm** của service. Nó chứa:
- **Entity**: struct thuần Go mô tả đối tượng nghiệp vụ.
- **Repository interface**: hợp đồng truy cập dữ liệu.
- **Service interface**: hợp đồng các use case nghiệp vụ.
- **Errors**: lỗi đặc thù của domain.

### 3.1. Entity — `internal/domain/product.go`

Entity là struct Go thuần, **không phụ thuộc GORM hay framework nào**. Chỉ dùng các kiểu dữ liệu Go cơ bản.

```go
package domain

import (
    "time"

    "github.com/google/uuid"
)

// Product là entity chính đại diện cho sản phẩm.
// Struct này KHÔNG có GORM tag — nó thuộc tầng domain, tách biệt hoàn toàn
// khỏi persistence layer.
type Product struct {
    ID          uuid.UUID
    Name        string
    Description *string    // nullable — sản phẩm có thể chưa có mô tả
    Price       float64
    Stock       int
    CategoryID  *uuid.UUID // nullable — sản phẩm có thể chưa phân loại
    IsActive    bool
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

// IsAvailable kiểm tra sản phẩm có đang bán và còn hàng hay không.
// Business logic thuộc về entity, giúp reuse ở nhiều nơi.
func (p *Product) IsAvailable() bool {
    return p.IsActive && p.Stock > 0
}

// DecreaseStock giảm tồn kho. Trả error nếu không đủ hàng.
func (p *Product) DecreaseStock(quantity int) error {
    if quantity <= 0 {
        return ErrInvalidQuantity
    }
    if p.Stock < quantity {
        return ErrInsufficientStock
    }
    p.Stock -= quantity
    return nil
}
```

**Giải thích chi tiết:**
- `uuid.UUID` làm primary key — mọi entity trong repo này dùng UUID.
- Trường nullable dùng pointer (`*string`, `*uuid.UUID`) — map trực tiếp sang SQL `NULL`.
- Business method đặt ngay trên entity (VD: `IsAvailable()`, `DecreaseStock()`) — đây là nơi logic nghiệp vụ đơn giản nhất nên sống.
- **KHÔNG** import GORM, Gin, hay bất kỳ framework nào.

### 3.2. Repository Interface — `internal/domain/repository.go`

Interface này mô tả **hợp đồng** giữa domain và tầng lưu trữ. Domain chỉ biết interface, không biết GORM.

```go
package domain

import (
    "context"

    "github.com/google/uuid"
)

// ProductRepository định nghĩa interface truy cập dữ liệu Product.
// Tầng domain chỉ khai báo interface, implementation nằm ở tầng repository.
type ProductRepository interface {
    // Create tạo sản phẩm mới. Trả về ErrProductAlreadyExists nếu trùng tên.
    Create(ctx context.Context, product *Product) error

    // GetByID tìm sản phẩm theo ID. Trả về ErrProductNotFound nếu không tồn tại.
    GetByID(ctx context.Context, id uuid.UUID) (*Product, error)

    // List lấy danh sách sản phẩm với phân trang và tìm kiếm.
    List(ctx context.Context, filter ProductFilter) ([]*Product, int64, error)

    // Update cập nhật thông tin sản phẩm.
    Update(ctx context.Context, product *Product) error

    // Delete xoá sản phẩm (soft delete).
    Delete(ctx context.Context, id uuid.UUID) error
}

// ProductFilter chứa các tiêu chí lọc/phân trang cho danh sách sản phẩm.
type ProductFilter struct {
    Search     string     // Tìm theo tên hoặc mô tả
    CategoryID *uuid.UUID // Lọc theo category
    IsActive   *bool      // Lọc theo trạng thái
    Page       int        // Trang hiện tại (bắt đầu từ 1)
    PageSize   int        // Số item mỗi trang
    SortBy     string     // Cột sắp xếp: name, price, created_at
    SortOrder  string     // asc hoặc desc
}
```

**Giải thích chi tiết:**
- Mọi method đều nhận `context.Context` là tham số đầu tiên — convention chuẩn Go cho cancellation, timeout, tracing.
- Filter struct tách riêng thay vì truyền nhiều tham số — dễ mở rộng mà không đổi signature.
- Comment mô tả rõ lỗi nào sẽ trả về trong từng trường hợp — giúp người implement biết chính xác hành vi mong đợi.

### 3.3. Service Interface — `internal/domain/service.go`

Interface này định nghĩa **các use case** mà delivery layer (handler) sẽ gọi.

```go
package domain

import (
    "context"

    "github.com/google/uuid"
)

// ProductService định nghĩa interface cho các use case quản lý sản phẩm.
// Tầng delivery gọi interface này, tầng usecase implement.
type ProductService interface {
    // CreateProduct tạo sản phẩm mới sau khi validate.
    CreateProduct(ctx context.Context, name string, description *string, price float64, stock int, categoryID *uuid.UUID) (*Product, error)

    // GetProduct lấy thông tin sản phẩm theo ID.
    GetProduct(ctx context.Context, id uuid.UUID) (*Product, error)

    // ListProducts lấy danh sách sản phẩm với filter và phân trang.
    ListProducts(ctx context.Context, filter ProductFilter) ([]*Product, int64, error)

    // UpdateProduct cập nhật thông tin sản phẩm.
    UpdateProduct(ctx context.Context, id uuid.UUID, name *string, description *string, price *float64, stock *int, isActive *bool) (*Product, error)

    // DeleteProduct xoá sản phẩm (soft delete).
    DeleteProduct(ctx context.Context, id uuid.UUID) error
}
```

**Giải thích chi tiết:**
- Tham số Update dùng pointer (`*string`, `*float64`) — cho phép phân biệt "không gửi field" (nil) và "gửi giá trị rỗng/zero".
- Interface này là **hợp đồng** giữa handler và usecase — handler chỉ biết interface, không biết implementation.
- Tên method ở service level thường mô tả hành động nghiệp vụ (`CreateProduct`), không phải hành động CRUD thuần (`Create`).

### 3.4. Domain Errors — `internal/domain/errors.go`

Lỗi domain sử dụng `pkg/apperror` để tự động map sang HTTP status code.

```go
package domain

import (
    "github.com/base-go/base/pkg/apperror"
)

// Domain-specific errors cho Product Service.
// Các lỗi này được định nghĩa ở tầng domain, không phụ thuộc HTTP hay framework.
// pkg/apperror tự động map sang HTTP status code phù hợp.
var (
    // ErrProductNotFound — sản phẩm không tồn tại → 404 Not Found
    ErrProductNotFound = apperror.NotFound("product not found")

    // ErrProductAlreadyExists — tên sản phẩm đã tồn tại → 409 Conflict
    ErrProductAlreadyExists = apperror.Conflict("product with this name already exists")

    // ErrInvalidPrice — giá không hợp lệ → 400 Bad Request
    ErrInvalidPrice = apperror.ValidationError("price must be greater than 0")

    // ErrInvalidQuantity — số lượng không hợp lệ → 400 Bad Request
    ErrInvalidQuantity = apperror.ValidationError("quantity must be greater than 0")

    // ErrInsufficientStock — không đủ tồn kho → 400 Bad Request
    ErrInsufficientStock = apperror.BadRequest("insufficient stock")
)
```

**Giải thích chi tiết:**
- Dùng các constructor từ `pkg/apperror`: `NotFound()`, `Conflict()`, `ValidationError()`, `BadRequest()`, v.v.
- Mỗi error type tự động map sang HTTP status tương ứng thông qua method `HTTPStatus()` trong `AppError`.
- Khai báo ở tầng domain nên có thể dùng ở cả usecase lẫn repository.

### Mapping AppError → HTTP Status (tham khảo)

| Constructor          | Code                | HTTP Status |
|---------------------|---------------------|-------------|
| `BadRequest(msg)`   | `BAD_REQUEST`       | 400         |
| `ValidationError()` | `VALIDATION_ERROR`  | 400         |
| `Unauthorized(msg)` | `UNAUTHORIZED`      | 401         |
| `TokenExpired()`    | `TOKEN_EXPIRED`     | 401         |
| `TokenInvalid()`    | `TOKEN_INVALID`     | 401         |
| `Forbidden(msg)`    | `FORBIDDEN`         | 403         |
| `NotFound(msg)`     | `NOT_FOUND`         | 404         |
| `Conflict(msg)`     | `CONFLICT`          | 409         |
| `RateLimited()`     | `RATE_LIMITED`      | 429         |
| `InternalError()`   | `INTERNAL_ERROR`    | 500         |

---

## 4. Bước 2 — Tạo Platform Layer (Hạ Tầng)

Platform layer cung cấp các adapter kỹ thuật: config, database, logger. Không chứa logic nghiệp vụ.

### 4.1. Config — `internal/platform/config/config.go`

Đọc cấu hình từ biến môi trường. Mỗi service có config riêng.

```go
package config

import (
    "fmt"
    "os"
    "strconv"
    "time"
)

// Config chứa toàn bộ cấu hình cho Product Service.
type Config struct {
    Server   ServerConfig
    Database DatabaseConfig
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
// Format chuẩn cho gorm driver: "host=... port=... user=... password=... dbname=... sslmode=..."
func (d DatabaseConfig) DSN() string {
    return fmt.Sprintf(
        "host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
        d.Host, d.Port, d.User, d.Password, d.DBName, d.SSLMode,
    )
}

// Load đọc cấu hình từ biến môi trường.
// Mỗi biến có giá trị mặc định phù hợp cho development.
func Load() (*Config, error) {
    cfg := &Config{
        Server: ServerConfig{
            Port:         getEnv("SERVER_PORT", "8082"),
            ReadTimeout:  getDurationEnv("SERVER_READ_TIMEOUT", 15*time.Second),
            WriteTimeout: getDurationEnv("SERVER_WRITE_TIMEOUT", 15*time.Second),
        },
        Database: DatabaseConfig{
            Host:     getEnv("DB_HOST", "localhost"),
            Port:     getEnv("DB_PORT", "5432"),
            User:     getEnv("DB_USER", "postgres"),
            Password: getEnv("DB_PASSWORD", "postgres"),
            DBName:   getEnv("DB_NAME", "product_db"),
            SSLMode:  getEnv("DB_SSLMODE", "disable"),
        },
    }
    return cfg, nil
}

// --- Helper functions ---

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
```

**Giải thích chi tiết:**
- Mỗi service dùng **port riêng** — auth dùng 8081, product dùng 8082, v.v.
- Mỗi service có thể dùng **database riêng** (`product_db`) — pattern "database per service" trong microservice.
- Hàm `getEnv()`, `getIntEnv()`, `getDurationEnv()` là pattern chuẩn trong repo — sao chép từ auth service.
- Nếu service cần JWT (VD: để validate token nội bộ), thêm `JWTConfig` vào struct `Config`.

### 4.2. Database — `internal/platform/database/postgres.go`

Có thể **tái sử dụng nguyên file** từ auth service vì logic khởi tạo connection là giống nhau.

```go
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

    // Connection pool settings — dùng default nếu chưa cấu hình.
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

    if err := sqlDB.Ping(); err != nil {
        return nil, fmt.Errorf("ping database: %w", err)
    }

    slog.Info("connected to PostgreSQL")
    return db, nil
}
```

### 4.3. Logger — `internal/platform/logger/logger.go`

Cũng **tái sử dụng nguyên file** từ auth service.

```go
package logger

import (
    "log/slog"
    "os"
)

// Setup khởi tạo structured logger (slog) cho toàn bộ service.
// JSON format cho production (log aggregator), Text format cho development (terminal).
func Setup(env string) {
    var handler slog.Handler

    opts := &slog.HandlerOptions{
        Level: slog.LevelInfo,
    }

    switch env {
    case "production", "prod":
        handler = slog.NewJSONHandler(os.Stdout, opts)
    default:
        opts.Level = slog.LevelDebug
        handler = slog.NewTextHandler(os.Stdout, opts)
    }

    logger := slog.New(handler)
    slog.SetDefault(logger)
}
```

> **Mẹo**: Trong tương lai, nếu các service dùng cùng code platform (database, logger), có thể cân nhắc di chuyển vào `pkg/` để tránh trùng lắp. Hiện tại, mỗi service giữ bản sao riêng để độc lập hoàn toàn.

---

## 5. Bước 3 — Tạo Repository Layer

Repository layer implement các interface đã khai báo ở domain. Nó bao gồm 3 thành phần:

### 5.1. GORM Model — `internal/repository/postgres/model/product_model.go`

Model GORM là **persistence model**, tách biệt hoàn toàn khỏi domain entity.

```go
package model

import (
    "time"

    "github.com/google/uuid"
    "gorm.io/gorm"
)

// ProductModel là GORM persistence model cho bảng products.
// Tách biệt khỏi domain entity — không dùng trực tiếp trong business logic.
type ProductModel struct {
    ID          uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
    Name        string         `gorm:"type:varchar(255);not null"`
    Description *string        `gorm:"type:text"`
    Price       float64        `gorm:"type:decimal(12,2);not null;default:0"`
    Stock       int            `gorm:"not null;default:0"`
    CategoryID  *uuid.UUID     `gorm:"type:uuid;index"`
    IsActive    bool           `gorm:"not null;default:true"`
    CreatedAt   time.Time      `gorm:"not null;autoCreateTime"`
    UpdatedAt   time.Time      `gorm:"not null;autoUpdateTime"`
    DeletedAt   gorm.DeletedAt `gorm:"index"` // Soft delete
}

// TableName trả về tên bảng trong database.
// Convention: tên bảng số nhiều, snake_case.
func (ProductModel) TableName() string {
    return "products"
}
```

**Giải thích chi tiết:**
- **Tại sao tách model khỏi entity?** Vì domain entity thuần logic nghiệp vụ, model GORM chứa annotation DB. Nếu đổi ORM (VD: sang sqlx), chỉ cần sửa model + mapper, domain không bị ảnh hưởng.
- `gorm.DeletedAt` cho phép soft delete — khi gọi `db.Delete()`, GORM tự set `deleted_at` thay vì xoá thật.
- `gen_random_uuid()` là hàm PostgreSQL — tự sinh UUID nếu Go không truyền.

### 5.2. Mapper — `internal/repository/postgres/mapper/product_mapper.go`

Mapper chuyển đổi qua lại giữa domain entity ↔ GORM model.

```go
package mapper

import (
    "github.com/base-go/base/services/product/internal/domain"
    "github.com/base-go/base/services/product/internal/repository/postgres/model"
)

// ProductToModel chuyển domain entity → GORM model.
// Dùng khi cần lưu dữ liệu vào DB.
func ProductToModel(p *domain.Product) *model.ProductModel {
    return &model.ProductModel{
        ID:          p.ID,
        Name:        p.Name,
        Description: p.Description,
        Price:       p.Price,
        Stock:       p.Stock,
        CategoryID:  p.CategoryID,
        IsActive:    p.IsActive,
        CreatedAt:   p.CreatedAt,
        UpdatedAt:   p.UpdatedAt,
    }
}

// ProductToDomain chuyển GORM model → domain entity.
// Dùng khi đọc dữ liệu từ DB trả về cho use case.
func ProductToDomain(m *model.ProductModel) *domain.Product {
    return &domain.Product{
        ID:          m.ID,
        Name:        m.Name,
        Description: m.Description,
        Price:       m.Price,
        Stock:       m.Stock,
        CategoryID:  m.CategoryID,
        IsActive:    m.IsActive,
        CreatedAt:   m.CreatedAt,
        UpdatedAt:   m.UpdatedAt,
    }
}

// ProductsToDomain chuyển slice model → slice domain, helper cho List query.
func ProductsToDomain(models []*model.ProductModel) []*domain.Product {
    products := make([]*domain.Product, len(models))
    for i, m := range models {
        products[i] = ProductToDomain(m)
    }
    return products
}
```

### 5.3. Repository Implementation — `internal/repository/postgres/product_repository.go`

```go
package postgres

import (
    "context"
    "errors"
    "log/slog"
    "strings"

    "github.com/google/uuid"
    "gorm.io/gorm"

    "github.com/base-go/base/services/product/internal/domain"
    "github.com/base-go/base/services/product/internal/repository/postgres/mapper"
    "github.com/base-go/base/services/product/internal/repository/postgres/model"
)

// productRepository implement domain.ProductRepository bằng GORM.
type productRepository struct {
    db *gorm.DB
}

// NewProductRepository tạo ProductRepository mới.
// Return type là domain.ProductRepository (interface) — không expose struct cụ thể.
func NewProductRepository(db *gorm.DB) domain.ProductRepository {
    return &productRepository{db: db}
}

func (r *productRepository) Create(ctx context.Context, product *domain.Product) error {
    m := mapper.ProductToModel(product)

    result := r.db.WithContext(ctx).Create(m)
    if result.Error != nil {
        // Kiểm tra duplicate (unique constraint violation).
        if strings.Contains(result.Error.Error(), "duplicate") ||
            strings.Contains(result.Error.Error(), "unique") {
            return domain.ErrProductAlreadyExists
        }
        slog.ErrorContext(ctx, "failed to create product", "error", result.Error)
        return result.Error
    }

    // Cập nhật ID, timestamp sinh bởi DB vào domain entity.
    product.ID = m.ID
    product.CreatedAt = m.CreatedAt
    product.UpdatedAt = m.UpdatedAt
    return nil
}

func (r *productRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Product, error) {
    var m model.ProductModel

    result := r.db.WithContext(ctx).Where("id = ?", id).First(&m)
    if result.Error != nil {
        if errors.Is(result.Error, gorm.ErrRecordNotFound) {
            return nil, domain.ErrProductNotFound
        }
        slog.ErrorContext(ctx, "failed to get product by id", "error", result.Error, "product_id", id)
        return nil, result.Error
    }

    return mapper.ProductToDomain(&m), nil
}

func (r *productRepository) List(ctx context.Context, filter domain.ProductFilter) ([]*domain.Product, int64, error) {
    var models []*model.ProductModel
    var total int64

    query := r.db.WithContext(ctx).Model(&model.ProductModel{})

    // Áp dụng filter tìm kiếm theo tên hoặc mô tả.
    if filter.Search != "" {
        search := "%" + strings.ToLower(filter.Search) + "%"
        query = query.Where("LOWER(name) LIKE ? OR LOWER(description) LIKE ?", search, search)
    }

    // Lọc theo category.
    if filter.CategoryID != nil {
        query = query.Where("category_id = ?", *filter.CategoryID)
    }

    // Lọc theo trạng thái.
    if filter.IsActive != nil {
        query = query.Where("is_active = ?", *filter.IsActive)
    }

    // Đếm tổng số record (trước khi pagination).
    if err := query.Count(&total).Error; err != nil {
        slog.ErrorContext(ctx, "failed to count products", "error", err)
        return nil, 0, err
    }

    // Sắp xếp.
    orderClause := "created_at DESC" // mặc định
    if filter.SortBy != "" {
        // Whitelist các cột cho phép sort — tránh SQL injection.
        allowedSorts := map[string]bool{"name": true, "price": true, "created_at": true, "stock": true}
        if allowedSorts[filter.SortBy] {
            order := "ASC"
            if strings.ToUpper(filter.SortOrder) == "DESC" {
                order = "DESC"
            }
            orderClause = filter.SortBy + " " + order
        }
    }
    query = query.Order(orderClause)

    // Phân trang.
    page := filter.Page
    if page < 1 {
        page = 1
    }
    pageSize := filter.PageSize
    if pageSize < 1 || pageSize > 100 {
        pageSize = 20
    }
    offset := (page - 1) * pageSize
    query = query.Offset(offset).Limit(pageSize)

    // Thực thi query.
    if err := query.Find(&models).Error; err != nil {
        slog.ErrorContext(ctx, "failed to list products", "error", err)
        return nil, 0, err
    }

    return mapper.ProductsToDomain(models), total, nil
}

func (r *productRepository) Update(ctx context.Context, product *domain.Product) error {
    m := mapper.ProductToModel(product)

    result := r.db.WithContext(ctx).Save(m)
    if result.Error != nil {
        slog.ErrorContext(ctx, "failed to update product", "error", result.Error, "product_id", product.ID)
        return result.Error
    }

    product.UpdatedAt = m.UpdatedAt
    return nil
}

func (r *productRepository) Delete(ctx context.Context, id uuid.UUID) error {
    result := r.db.WithContext(ctx).Where("id = ?", id).Delete(&model.ProductModel{})
    if result.Error != nil {
        slog.ErrorContext(ctx, "failed to delete product", "error", result.Error, "product_id", id)
        return result.Error
    }

    if result.RowsAffected == 0 {
        return domain.ErrProductNotFound
    }

    return nil
}
```

**Những điểm cần chú ý:**
- `WithContext(ctx)` — truyền context xuyên suốt, hỗ trợ timeout/cancellation.
- Sort column dùng whitelist — **bắt buộc** để tránh SQL injection.
- `gorm.ErrRecordNotFound` → map sang domain error (`ErrProductNotFound`) — không leak lỗi GORM ra ngoài.
- `Delete` dùng GORM soft delete (`gorm.DeletedAt`), GORM sẽ tự set `deleted_at` thay vì xoá thật.
- `RowsAffected == 0` kiểm tra record không tồn tại khi delete.

---

## 6. Bước 4 — Tạo Usecase Layer

Usecase layer chứa **logic nghiệp vụ chính**. Nó implement service interface từ domain.

### 6.1. DTO — `internal/usecase/dto/product_dto.go`

DTO (Data Transfer Object) là struct **chuyên dùng cho request/response HTTP**, tách biệt khỏi domain entity.

```go
package dto

import "time"

// --- Request DTOs ---

// CreateProductRequest chứa dữ liệu tạo sản phẩm mới.
// Tag `binding` dùng cho Gin validator.
type CreateProductRequest struct {
    Name        string  `json:"name" binding:"required,min=1,max=255"`
    Description *string `json:"description"`
    Price       float64 `json:"price" binding:"required,gt=0"`
    Stock       int     `json:"stock" binding:"min=0"`
    CategoryID  *string `json:"category_id"` // string vì JSON không có kiểu UUID
    IsActive    *bool   `json:"is_active"`
}

// UpdateProductRequest chứa dữ liệu cập nhật sản phẩm.
// Tất cả field đều là pointer — nil nghĩa là "không thay đổi".
type UpdateProductRequest struct {
    Name        *string  `json:"name" binding:"omitempty,min=1,max=255"`
    Description *string  `json:"description"`
    Price       *float64 `json:"price" binding:"omitempty,gt=0"`
    Stock       *int     `json:"stock" binding:"omitempty,min=0"`
    IsActive    *bool    `json:"is_active"`
}

// ListProductsRequest chứa tham số query cho danh sách sản phẩm.
type ListProductsRequest struct {
    Search     string `form:"search"`
    CategoryID string `form:"category_id"`
    IsActive   *bool  `form:"is_active"`
    Page       int    `form:"page,default=1" binding:"min=1"`
    PageSize   int    `form:"page_size,default=20" binding:"min=1,max=100"`
    SortBy     string `form:"sort_by"`
    SortOrder  string `form:"sort_order,default=desc"`
}

// --- Response DTOs ---

// ProductResponse chứa thông tin sản phẩm trả về cho client.
type ProductResponse struct {
    ID          string    `json:"id"`
    Name        string    `json:"name"`
    Description *string   `json:"description,omitempty"`
    Price       float64   `json:"price"`
    Stock       int       `json:"stock"`
    CategoryID  *string   `json:"category_id,omitempty"`
    IsActive    bool      `json:"is_active"`
    CreatedAt   time.Time `json:"created_at"`
    UpdatedAt   time.Time `json:"updated_at"`
}

// ListProductsResponse trả về danh sách sản phẩm kèm thông tin phân trang.
type ListProductsResponse struct {
    Products   []ProductResponse `json:"products"`
    Total      int64             `json:"total"`
    Page       int               `json:"page"`
    PageSize   int               `json:"page_size"`
    TotalPages int64             `json:"total_pages"`
}

// MessageResponse dùng cho các endpoint chỉ trả thông điệp.
type MessageResponse struct {
    Message string `json:"message"`
}
```

**Giải thích chi tiết:**
- **Không dùng `uuid.UUID` trong DTO request** — client gửi UUID dạng string, handler parse.
- **Update request dùng pointer** — `nil` = không thay đổi, `""` = set rỗng. Đây là pattern chuẩn cho partial update.
- Tag `binding` dùng `go-playground/validator` tích hợp sẵn trong Gin.
- Tag `form` dùng cho query parameter (GET request).

### 6.2. Usecase Implementation — `internal/usecase/product_usecase.go`

```go
package usecase

import (
    "context"
    "log/slog"

    "github.com/google/uuid"

    "github.com/base-go/base/pkg/apperror"
    "github.com/base-go/base/services/product/internal/domain"
)

// productUsecase implement domain.ProductService.
type productUsecase struct {
    productRepo domain.ProductRepository
}

// NewProductUsecase tạo ProductService mới.
// Nhận vào interface (domain.ProductRepository), không phải implementation cụ thể.
func NewProductUsecase(productRepo domain.ProductRepository) domain.ProductService {
    return &productUsecase{
        productRepo: productRepo,
    }
}

func (uc *productUsecase) CreateProduct(
    ctx context.Context,
    name string,
    description *string,
    price float64,
    stock int,
    categoryID *uuid.UUID,
) (*domain.Product, error) {
    // Validate input ở tầng usecase — đây là biên vào cuối cùng trước khi persist.
    if name == "" {
        return nil, apperror.ValidationError("product name is required")
    }
    if price <= 0 {
        return nil, domain.ErrInvalidPrice
    }
    if stock < 0 {
        return nil, apperror.ValidationError("stock cannot be negative")
    }

    product := &domain.Product{
        ID:          uuid.New(),
        Name:        name,
        Description: description,
        Price:       price,
        Stock:       stock,
        CategoryID:  categoryID,
        IsActive:    true,
    }

    if err := uc.productRepo.Create(ctx, product); err != nil {
        return nil, err
    }

    slog.InfoContext(ctx, "product created", "product_id", product.ID, "name", product.Name)
    return product, nil
}

func (uc *productUsecase) GetProduct(ctx context.Context, id uuid.UUID) (*domain.Product, error) {
    product, err := uc.productRepo.GetByID(ctx, id)
    if err != nil {
        return nil, err
    }
    return product, nil
}

func (uc *productUsecase) ListProducts(ctx context.Context, filter domain.ProductFilter) ([]*domain.Product, int64, error) {
    return uc.productRepo.List(ctx, filter)
}

func (uc *productUsecase) UpdateProduct(
    ctx context.Context,
    id uuid.UUID,
    name *string,
    description *string,
    price *float64,
    stock *int,
    isActive *bool,
) (*domain.Product, error) {
    // Lấy product hiện tại.
    product, err := uc.productRepo.GetByID(ctx, id)
    if err != nil {
        return nil, err
    }

    // Cập nhật từng field nếu client gửi lên (partial update).
    if name != nil {
        if *name == "" {
            return nil, apperror.ValidationError("product name cannot be empty")
        }
        product.Name = *name
    }
    if description != nil {
        product.Description = description
    }
    if price != nil {
        if *price <= 0 {
            return nil, domain.ErrInvalidPrice
        }
        product.Price = *price
    }
    if stock != nil {
        if *stock < 0 {
            return nil, apperror.ValidationError("stock cannot be negative")
        }
        product.Stock = *stock
    }
    if isActive != nil {
        product.IsActive = *isActive
    }

    if err := uc.productRepo.Update(ctx, product); err != nil {
        return nil, err
    }

    slog.InfoContext(ctx, "product updated", "product_id", id)
    return product, nil
}

func (uc *productUsecase) DeleteProduct(ctx context.Context, id uuid.UUID) error {
    if err := uc.productRepo.Delete(ctx, id); err != nil {
        return err
    }

    slog.InfoContext(ctx, "product deleted", "product_id", id)
    return nil
}
```

**Những điểm quan trọng:**
- Usecase **không import GORM** — chỉ gọi domain interface.
- Validate input ở **biên vào cuối cùng** (usecase) — handler validate format (binding tag), usecase validate business rule.
- Log dùng `slog` — structured logging, dễ filter/search.
- Pattern partial update: kiểm tra `if field != nil` → cập nhật.

---

## 7. Bước 5 — Tạo Delivery Layer (HTTP)

### 7.1. Presenter — `internal/delivery/http/presenter/product_presenter.go`

Presenter chuyển domain entity → DTO response. Đảm bảo **không leak internal data**.

```go
package presenter

import (
    "github.com/base-go/base/services/product/internal/domain"
    "github.com/base-go/base/services/product/internal/usecase/dto"
)

// ProductToResponse chuyển domain.Product → dto.ProductResponse.
func ProductToResponse(p *domain.Product) dto.ProductResponse {
    resp := dto.ProductResponse{
        ID:          p.ID.String(),
        Name:        p.Name,
        Description: p.Description,
        Price:       p.Price,
        Stock:       p.Stock,
        IsActive:    p.IsActive,
        CreatedAt:   p.CreatedAt,
        UpdatedAt:   p.UpdatedAt,
    }

    if p.CategoryID != nil {
        catID := p.CategoryID.String()
        resp.CategoryID = &catID
    }

    return resp
}

// ProductsToResponse chuyển slice domain.Product → []dto.ProductResponse.
func ProductsToResponse(products []*domain.Product) []dto.ProductResponse {
    responses := make([]dto.ProductResponse, len(products))
    for i, p := range products {
        responses[i] = ProductToResponse(p)
    }
    return responses
}
```

### 7.2. Handler — `internal/delivery/http/handler/product_handler.go`

Handler xử lý HTTP request. Mỗi handler method tương ứng với 1 endpoint.

```go
package handler

import (
    "net/http"

    "github.com/gin-gonic/gin"
    "github.com/google/uuid"

    "github.com/base-go/base/pkg/apperror"
    "github.com/base-go/base/pkg/response"
    "github.com/base-go/base/services/product/internal/delivery/http/presenter"
    "github.com/base-go/base/services/product/internal/domain"
    "github.com/base-go/base/services/product/internal/usecase/dto"
)

// ProductHandler xử lý HTTP request cho các endpoint sản phẩm.
type ProductHandler struct {
    productService domain.ProductService
}

// NewProductHandler tạo ProductHandler mới.
// Nhận vào interface (domain.ProductService) — không phải implementation cụ thể.
func NewProductHandler(productService domain.ProductService) *ProductHandler {
    return &ProductHandler{
        productService: productService,
    }
}

// CreateProduct xử lý POST /api/v1/products.
func (h *ProductHandler) CreateProduct(c *gin.Context) {
    var req dto.CreateProductRequest
    // ShouldBindJSON parse body + validate (tag binding).
    if err := c.ShouldBindJSON(&req); err != nil {
        response.Error(c, apperror.ValidationError(err.Error()))
        return
    }

    // Parse category_id từ string → uuid.UUID (nếu có).
    var categoryID *uuid.UUID
    if req.CategoryID != nil && *req.CategoryID != "" {
        parsed, err := uuid.Parse(*req.CategoryID)
        if err != nil {
            response.Error(c, apperror.ValidationError("invalid category_id format"))
            return
        }
        categoryID = &parsed
    }

    product, err := h.productService.CreateProduct(
        c.Request.Context(), // Truyền context từ request
        req.Name,
        req.Description,
        req.Price,
        req.Stock,
        categoryID,
    )
    if err != nil {
        response.Error(c, err) // response.Error tự map AppError → HTTP status
        return
    }

    response.Created(c, presenter.ProductToResponse(product))
}

// GetProduct xử lý GET /api/v1/products/:id.
func (h *ProductHandler) GetProduct(c *gin.Context) {
    // Parse path parameter.
    id, err := uuid.Parse(c.Param("id"))
    if err != nil {
        response.Error(c, apperror.ValidationError("invalid product id"))
        return
    }

    product, err := h.productService.GetProduct(c.Request.Context(), id)
    if err != nil {
        response.Error(c, err)
        return
    }

    response.OK(c, presenter.ProductToResponse(product))
}

// ListProducts xử lý GET /api/v1/products.
func (h *ProductHandler) ListProducts(c *gin.Context) {
    var req dto.ListProductsRequest
    // ShouldBindQuery parse query parameters + validate.
    if err := c.ShouldBindQuery(&req); err != nil {
        response.Error(c, apperror.ValidationError(err.Error()))
        return
    }

    // Xây dựng filter từ request DTO.
    filter := domain.ProductFilter{
        Search:    req.Search,
        IsActive:  req.IsActive,
        Page:      req.Page,
        PageSize:  req.PageSize,
        SortBy:    req.SortBy,
        SortOrder: req.SortOrder,
    }

    // Parse category_id nếu có.
    if req.CategoryID != "" {
        parsed, err := uuid.Parse(req.CategoryID)
        if err != nil {
            response.Error(c, apperror.ValidationError("invalid category_id"))
            return
        }
        filter.CategoryID = &parsed
    }

    products, total, err := h.productService.ListProducts(c.Request.Context(), filter)
    if err != nil {
        response.Error(c, err)
        return
    }

    // Tính tổng số trang.
    pageSize := req.PageSize
    if pageSize < 1 {
        pageSize = 20
    }
    totalPages := (total + int64(pageSize) - 1) / int64(pageSize)

    response.OK(c, dto.ListProductsResponse{
        Products:   presenter.ProductsToResponse(products),
        Total:      total,
        Page:       req.Page,
        PageSize:   pageSize,
        TotalPages: totalPages,
    })
}

// UpdateProduct xử lý PUT /api/v1/products/:id.
func (h *ProductHandler) UpdateProduct(c *gin.Context) {
    id, err := uuid.Parse(c.Param("id"))
    if err != nil {
        response.Error(c, apperror.ValidationError("invalid product id"))
        return
    }

    var req dto.UpdateProductRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        response.Error(c, apperror.ValidationError(err.Error()))
        return
    }

    product, err := h.productService.UpdateProduct(
        c.Request.Context(),
        id,
        req.Name,
        req.Description,
        req.Price,
        req.Stock,
        req.IsActive,
    )
    if err != nil {
        response.Error(c, err)
        return
    }

    response.OK(c, presenter.ProductToResponse(product))
}

// DeleteProduct xử lý DELETE /api/v1/products/:id.
func (h *ProductHandler) DeleteProduct(c *gin.Context) {
    id, err := uuid.Parse(c.Param("id"))
    if err != nil {
        response.Error(c, apperror.ValidationError("invalid product id"))
        return
    }

    if err := h.productService.DeleteProduct(c.Request.Context(), id); err != nil {
        response.Error(c, err)
        return
    }

    response.OK(c, dto.MessageResponse{Message: "product deleted successfully"})
}

// HealthCheck xử lý GET /health.
func (h *ProductHandler) HealthCheck(c *gin.Context) {
    c.JSON(http.StatusOK, gin.H{
        "status":  "ok",
        "service": "product-service",
    })
}
```

**Pattern cố định trong mọi handler method:**
```
1. Parse + validate input (ShouldBindJSON / ShouldBindQuery / Param)
2. Gọi service method (thông qua domain interface)
3. Xử lý error (response.Error tự map)
4. Trả response qua presenter
```

### 7.3. Router — `internal/delivery/http/router.go`

```go
package http

import (
    "github.com/gin-gonic/gin"

    pkgmiddleware "github.com/base-go/base/pkg/middleware"
    "github.com/base-go/base/services/product/internal/delivery/http/handler"
)

// NewRouter tạo Gin engine với tất cả route cho Product Service.
func NewRouter(productHandler *handler.ProductHandler) *gin.Engine {
    gin.SetMode(gin.ReleaseMode)

    r := gin.New()

    // Global middleware
    r.Use(gin.Recovery())                                       // Recover từ panic
    r.Use(pkgmiddleware.CORS(pkgmiddleware.DefaultCORSConfig())) // CORS dùng chung

    // Health check — không cần auth
    r.GET("/health", productHandler.HealthCheck)

    // API v1 group
    v1 := r.Group("/api/v1")
    {
        products := v1.Group("/products")
        {
            products.POST("", productHandler.CreateProduct)       // POST   /api/v1/products
            products.GET("", productHandler.ListProducts)         // GET    /api/v1/products
            products.GET("/:id", productHandler.GetProduct)       // GET    /api/v1/products/:id
            products.PUT("/:id", productHandler.UpdateProduct)    // PUT    /api/v1/products/:id
            products.DELETE("/:id", productHandler.DeleteProduct) // DELETE /api/v1/products/:id
        }
    }

    return r
}
```

**Giải thích chi tiết:**
- `gin.SetMode(gin.ReleaseMode)` — tắt debug output của Gin trong production.
- `gin.Recovery()` — middleware recover panic, trả 500 thay vì crash server.
- Route versioning: `/api/v1/...` — dễ nâng cấp API sau này.
- Nếu cần authentication, thêm middleware JWT giống auth service (xem phần Auth middleware bên dưới).

**Thêm auth middleware (nếu endpoint cần bảo vệ):**

```go
// Nếu cần bảo vệ endpoint bằng JWT:
import pkgjwt "github.com/base-go/base/pkg/jwt"

func NewRouter(productHandler *handler.ProductHandler, jwtManager *pkgjwt.Manager) *gin.Engine {
    // ... (như trên) ...

    products := v1.Group("/products")
    {
        // Public: ai cũng xem được
        products.GET("", productHandler.ListProducts)
        products.GET("/:id", productHandler.GetProduct)

        // Protected: cần login
        protected := products.Group("")
        protected.Use(middleware.AuthMiddleware(jwtManager))
        {
            protected.POST("", productHandler.CreateProduct)
            protected.PUT("/:id", productHandler.UpdateProduct)
            protected.DELETE("/:id", productHandler.DeleteProduct)
        }
    }
}
```

---

## 8. Bước 6 — Tạo Bootstrap (Wiring Tất Cả)

Bootstrap là **nơi duy nhất** khởi tạo và wire tất cả dependency lại với nhau.

### `internal/bootstrap/app.go`

```go
package bootstrap

import (
    "context"
    "fmt"
    "log/slog"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

    deliveryhttp "github.com/base-go/base/services/product/internal/delivery/http"
    "github.com/base-go/base/services/product/internal/delivery/http/handler"
    "github.com/base-go/base/services/product/internal/platform/config"
    "github.com/base-go/base/services/product/internal/platform/database"
    "github.com/base-go/base/services/product/internal/platform/logger"
    "github.com/base-go/base/services/product/internal/repository/postgres"
    "github.com/base-go/base/services/product/internal/repository/postgres/model"
    "github.com/base-go/base/services/product/internal/usecase"
)

// App chứa tất cả dependency đã wire và server HTTP.
type App struct {
    cfg    *config.Config
    server *http.Server
}

// NewApp khởi tạo toàn bộ dependency theo thứ tự:
// config → logger → database → repository → usecase → handler → router → server
//
// Đây là Dependency Injection thủ công (manual DI) — không dùng framework DI.
// Ưu điểm: dễ đọc, dễ debug, compile-time safe.
func NewApp() (*App, error) {
    // 1. Load config từ env.
    cfg, err := config.Load()
    if err != nil {
        return nil, fmt.Errorf("load config: %w", err)
    }

    // 2. Setup logger.
    env := os.Getenv("APP_ENV")
    logger.Setup(env)
    slog.Info("starting product service", "port", cfg.Server.Port, "env", env)

    // 3. Kết nối database.
    db, err := database.NewPostgresDB(database.Config{
        DSN: cfg.Database.DSN(),
    })
    if err != nil {
        return nil, fmt.Errorf("connect database: %w", err)
    }

    // Auto-migrate: tạo bảng nếu chưa tồn tại.
    // Trong production nên dùng migration tool, không dùng auto-migrate.
    if err := db.AutoMigrate(
        &model.ProductModel{},
    ); err != nil {
        return nil, fmt.Errorf("auto-migrate: %w", err)
    }
    slog.Info("database migration completed")

    // 4. Khởi tạo repository.
    productRepo := postgres.NewProductRepository(db)

    // 5. Khởi tạo usecase.
    productService := usecase.NewProductUsecase(productRepo)

    // 6. Khởi tạo handler.
    productHandler := handler.NewProductHandler(productService)

    // 7. Tạo router.
    router := deliveryhttp.NewRouter(productHandler)

    // 8. Tạo HTTP server.
    server := &http.Server{
        Addr:         ":" + cfg.Server.Port,
        Handler:      router,
        ReadTimeout:  cfg.Server.ReadTimeout,
        WriteTimeout: cfg.Server.WriteTimeout,
    }

    return &App{
        cfg:    cfg,
        server: server,
    }, nil
}

// Run khởi chạy HTTP server và xử lý graceful shutdown.
func (a *App) Run() error {
    // Channel nhận signal để graceful shutdown.
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

    // Chạy server trong goroutine riêng.
    errCh := make(chan error, 1)
    go func() {
        slog.Info("product service is running", "addr", a.server.Addr)
        if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            errCh <- fmt.Errorf("server error: %w", err)
        }
    }()

    // Chờ signal hoặc lỗi server.
    select {
    case err := <-errCh:
        return err
    case sig := <-quit:
        slog.Info("received shutdown signal", "signal", sig.String())
    }

    // Graceful shutdown với timeout 10 giây.
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    if err := a.server.Shutdown(ctx); err != nil {
        return fmt.Errorf("server shutdown: %w", err)
    }

    slog.Info("product service stopped gracefully")
    return nil
}
```

**Thứ tự wire bắt buộc:**
```
config → logger → database → repository(db) → usecase(repo) → handler(usecase) → router(handler) → server(router)
```

Nếu bạn cần thêm JWT (có endpoint cần bảo vệ), thêm bước khởi tạo `pkgjwt.NewManager(...)` giữa bước 3 và 4, rồi truyền vào router.

---

## 9. Bước 7 — Tạo Entrypoint (main.go)

### `cmd/product/main.go`

Đây là file ngắn nhất — chỉ gọi bootstrap.

```go
package main

import (
    "log"

    "github.com/base-go/base/services/product/internal/bootstrap"
)

func main() {
    app, err := bootstrap.NewApp()
    if err != nil {
        log.Fatalf("failed to initialize product service: %v", err)
    }

    if err := app.Run(); err != nil {
        log.Fatalf("product service exited with error: %v", err)
    }
}
```

**Giải thích:**
- `main()` không chứa logic — chỉ gọi `bootstrap.NewApp()` và `app.Run()`.
- Dùng `log.Fatalf` thay vì `slog` vì logger chưa được setup tại thời điểm lỗi bootstrap.

---

## 10. Bước 8 — Tạo Migration

### `migrations/000001_create_products_table.up.sql`

```sql
-- 000001_create_products_table.up.sql
-- Tạo bảng products cho Product Service.

-- Extension để dùng gen_random_uuid() cho UUID.
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS products (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(255)  NOT NULL,
    description TEXT          NULL,
    price       DECIMAL(12,2) NOT NULL DEFAULT 0,
    stock       INTEGER       NOT NULL DEFAULT 0,
    category_id UUID          NULL,
    is_active   BOOLEAN       NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ   NOT NULL DEFAULT now(),
    deleted_at  TIMESTAMPTZ   NULL
);

-- Index cho tìm kiếm theo tên.
CREATE INDEX IF NOT EXISTS idx_products_name ON products USING gin (name gin_trgm_ops);

-- Index cho lọc theo category.
CREATE INDEX IF NOT EXISTS idx_products_category_id ON products (category_id);

-- Index cho soft-delete queries.
CREATE INDEX IF NOT EXISTS idx_products_deleted_at ON products (deleted_at);

-- Index cho lọc theo trạng thái.
CREATE INDEX IF NOT EXISTS idx_products_is_active ON products (is_active) WHERE deleted_at IS NULL;

COMMENT ON TABLE products IS 'Bảng lưu trữ thông tin sản phẩm';
COMMENT ON COLUMN products.name IS 'Tên sản phẩm';
COMMENT ON COLUMN products.price IS 'Giá sản phẩm (VNĐ hoặc USD)';
COMMENT ON COLUMN products.stock IS 'Số lượng tồn kho hiện tại';
COMMENT ON COLUMN products.category_id IS 'FK tới bảng categories (nullable)';
COMMENT ON COLUMN products.is_active IS 'Trạng thái: true = đang bán, false = ngưng bán';
COMMENT ON COLUMN products.deleted_at IS 'Soft-delete timestamp, NULL = chưa xoá';
```

### `migrations/000001_create_products_table.down.sql`

```sql
-- 000001_create_products_table.down.sql
-- Rollback: xoá bảng products.

DROP TABLE IF EXISTS products;
```

**Quy ước đặt tên migration:**
```
<số thứ tự 6 chữ số>_<mô tả hành động>.up.sql
<số thứ tự 6 chữ số>_<mô tả hành động>.down.sql
```

**Lưu ý:**
- GORM AutoMigrate chỉ tạo bảng/cột mới, **không** xoá cột cũ hay thay đổi cấu trúc phức tạp.
- Trong production, dùng migration tool (golang-migrate, goose, atlas) thay vì AutoMigrate.

---

## 11. Bước 9 — Tạo File Cấu Hình

### `configs/.env.example`

```env
# ========================================
# Product Service — Environment Variables
# ========================================

# App
APP_ENV=development

# Server
SERVER_PORT=8082
SERVER_READ_TIMEOUT=15s
SERVER_WRITE_TIMEOUT=15s

# Database (PostgreSQL)
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=product_db
DB_SSLMODE=disable
```

> **Chú ý**: Mỗi service dùng port và database khác nhau:
> - Auth Service: port `8081`, database `auth_db`
> - Product Service: port `8082`, database `product_db`
> - Gateway: port `8080`

---

## 12. Bước 10 — Tạo Dockerfile

### `services/product/Dockerfile`

```dockerfile
# Build stage — dùng multi-stage build để image nhỏ gọn.
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /build

# Copy go.mod trước để cache layer download dependencies.
COPY go.mod go.sum ./
RUN go mod download

# Copy toàn bộ source code.
COPY . .

# Build binary — static linking, strip debug info.
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/product-service ./services/product/cmd/product/

# Run stage — image tối thiểu.
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Copy binary từ builder stage.
COPY --from=builder /app/product-service .

# Copy config files nếu cần.
COPY services/product/configs/ ./configs/

EXPOSE 8082

CMD ["./product-service"]
```

**Giải thích chi tiết:**
- **Multi-stage build**: builder stage (~1GB Go image) chỉ dùng để build, run stage (~10MB Alpine) chỉ chứa binary.
- `CGO_ENABLED=0` — static binary, không cần libc.
- `-ldflags="-s -w"` — strip debug info, giảm kích thước binary.
- `COPY go.mod go.sum` trước `COPY . .` — Docker cache layer download, rebuild nhanh hơn khi chỉ đổi code.
- Build path `./services/product/cmd/product/` — vì Dockerfile context là root repo.

---

## 13. Bước 11 — Cập Nhật docker-compose.yml

Thêm khối service mới vào file `docker-compose.yml` ở root repo:

```yaml
  # =============================================
  # Product Database — PostgreSQL riêng cho Product Service
  # =============================================
  product-postgres:
    image: postgres:16-alpine
    container_name: base-product-postgres
    restart: unless-stopped
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
      POSTGRES_DB: product_db
    ports:
      - "5433:5432"       # Port khác với auth-postgres (5432)
    volumes:
      - product_postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres"]
      interval: 5s
      timeout: 5s
      retries: 5

  # =============================================
  # Product Service — Quản lý sản phẩm
  # =============================================
  product-service:
    build:
      context: .                              # Context là root repo
      dockerfile: services/product/Dockerfile
    container_name: base-product
    restart: unless-stopped
    depends_on:
      product-postgres:
        condition: service_healthy
    environment:
      APP_ENV: development
      SERVER_PORT: "8082"
      DB_HOST: product-postgres              # Tên service trong docker network
      DB_PORT: "5432"                         # Port bên TRONG container
      DB_USER: postgres
      DB_PASSWORD: postgres
      DB_NAME: product_db
      DB_SSLMODE: disable
    ports:
      - "8082:8082"
```

Đừng quên thêm volume vào cuối file:

```yaml
volumes:
  postgres_data:
  product_postgres_data:       # Thêm dòng này
```

**Lưu ý quan trọng:**
- `DB_HOST` dùng **tên service** (`product-postgres`), không dùng `localhost` — vì các container giao tiếp qua Docker network.
- Port expose ra host (`5433:5432`) phải khác port đã dùng (`5432` của auth-postgres).
- Nếu muốn dùng **chung database** với auth service (chia schema), chỉ cần trỏ `DB_HOST: postgres` và `DB_NAME: auth_db` (hoặc tạo schema riêng).

---

## 14. Bước 12 — Đăng Ký Route Trên Gateway

Nếu gateway dùng file config route, cập nhật file `services/gateway/configs/routes.docker.yaml` (hoặc tương đương):

```yaml
routes:
  # Auth Service routes (đã có sẵn)
  - path_prefix: /api/v1/auth
    target: http://auth-service:8081
    strip_prefix: false

  # Product Service routes (thêm mới)
  - path_prefix: /api/v1/products
    target: http://product-service:8082
    strip_prefix: false
```

> **target** dùng **tên service** trong Docker Compose + port nội bộ.

---

## 15. Bước 13 — Viết Test

### 15.1. Unit Test cho Usecase — `internal/usecase/product_usecase_test.go`

Dùng **table-driven test** — convention chuẩn trong Go.

```go
package usecase

import (
    "context"
    "testing"

    "github.com/google/uuid"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"

    "github.com/base-go/base/services/product/internal/domain"
)

// mockProductRepository là mock implementation của domain.ProductRepository.
// Dùng testify/mock để tạo mock.
type mockProductRepository struct {
    mock.Mock
}

func (m *mockProductRepository) Create(ctx context.Context, product *domain.Product) error {
    args := m.Called(ctx, product)
    return args.Error(0)
}

func (m *mockProductRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Product, error) {
    args := m.Called(ctx, id)
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*domain.Product), args.Error(1)
}

func (m *mockProductRepository) List(ctx context.Context, filter domain.ProductFilter) ([]*domain.Product, int64, error) {
    args := m.Called(ctx, filter)
    return args.Get(0).([]*domain.Product), args.Get(1).(int64), args.Error(2)
}

func (m *mockProductRepository) Update(ctx context.Context, product *domain.Product) error {
    args := m.Called(ctx, product)
    return args.Error(0)
}

func (m *mockProductRepository) Delete(ctx context.Context, id uuid.UUID) error {
    args := m.Called(ctx, id)
    return args.Error(0)
}

func TestCreateProduct(t *testing.T) {
    // Table-driven tests — mỗi test case là một hàng trong bảng.
    tests := []struct {
        name        string
        productName string
        price       float64
        stock       int
        setupMock   func(repo *mockProductRepository)
        wantErr     bool
        errMsg      string
    }{
        {
            name:        "thành công — tạo sản phẩm hợp lệ",
            productName: "Laptop Dell XPS 15",
            price:       25000000,
            stock:       50,
            setupMock: func(repo *mockProductRepository) {
                repo.On("Create", mock.Anything, mock.Anything).Return(nil)
            },
            wantErr: false,
        },
        {
            name:        "lỗi — tên rỗng",
            productName: "",
            price:       25000000,
            stock:       50,
            setupMock:   func(repo *mockProductRepository) {},
            wantErr:     true,
            errMsg:      "product name is required",
        },
        {
            name:        "lỗi — giá âm",
            productName: "Test Product",
            price:       -100,
            stock:       10,
            setupMock:   func(repo *mockProductRepository) {},
            wantErr:     true,
            errMsg:      "price must be greater than 0",
        },
        {
            name:        "lỗi — sản phẩm trùng tên",
            productName: "Existing Product",
            price:       100000,
            stock:       10,
            setupMock: func(repo *mockProductRepository) {
                repo.On("Create", mock.Anything, mock.Anything).Return(domain.ErrProductAlreadyExists)
            },
            wantErr: true,
            errMsg:  "product with this name already exists",
        },
    }

    for _, tc := range tests {
        t.Run(tc.name, func(t *testing.T) {
            // Arrange: tạo mock và usecase.
            repo := new(mockProductRepository)
            tc.setupMock(repo)
            uc := NewProductUsecase(repo)

            // Act: gọi method cần test.
            product, err := uc.CreateProduct(
                context.Background(),
                tc.productName,
                nil, // description
                tc.price,
                tc.stock,
                nil, // categoryID
            )

            // Assert: kiểm tra kết quả.
            if tc.wantErr {
                assert.Error(t, err)
                assert.Contains(t, err.Error(), tc.errMsg)
                assert.Nil(t, product)
            } else {
                assert.NoError(t, err)
                assert.NotNil(t, product)
                assert.Equal(t, tc.productName, product.Name)
                assert.Equal(t, tc.price, product.Price)
            }

            repo.AssertExpectations(t)
        })
    }
}
```

### 15.2. Chạy test

```bash
# Chạy test cho toàn bộ product service
go test ./services/product/...

# Chạy test với verbose output
go test -v ./services/product/...

# Chạy test với coverage
go test -cover ./services/product/...
```

---

## 16. Bước 14 — Chạy Và Kiểm Tra

### 16.1. Chạy local (không Docker)

```bash
# 1. Tạo database (cần PostgreSQL chạy sẵn)
psql -U postgres -c "CREATE DATABASE product_db;"

# 2. Set biến môi trường
export APP_ENV=development
export SERVER_PORT=8082
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=postgres
export DB_PASSWORD=postgres
export DB_NAME=product_db
export DB_SSLMODE=disable

# 3. Chạy service
go run ./services/product/cmd/product/
```

### 16.2. Chạy bằng Docker Compose

```bash
# Build và chạy tất cả services
docker compose up --build

# Chỉ chạy product service (và database)
docker compose up --build product-service product-postgres
```

### 16.3. Kiểm tra bằng curl

```bash
# Health check
curl http://localhost:8082/health

# Tạo sản phẩm
curl -X POST http://localhost:8082/api/v1/products \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Laptop Dell XPS 15",
    "description": "Laptop cao cấp cho developer",
    "price": 25000000,
    "stock": 50
  }'

# Lấy danh sách sản phẩm
curl "http://localhost:8082/api/v1/products?page=1&page_size=10"

# Lấy sản phẩm theo ID
curl http://localhost:8082/api/v1/products/<uuid>

# Cập nhật sản phẩm
curl -X PUT http://localhost:8082/api/v1/products/<uuid> \
  -H "Content-Type: application/json" \
  -d '{"price": 23000000}'

# Xoá sản phẩm
curl -X DELETE http://localhost:8082/api/v1/products/<uuid>
```

---

## 17. Checklist Tổng Kết

Khi tạo service mới, đánh dấu từng mục sau khi hoàn thành:

### Domain Layer
- [ ] Entity struct (thuần Go, không GORM tag)
- [ ] Repository interface (định nghĩa trong domain)
- [ ] Service interface (định nghĩa trong domain)
- [ ] Domain errors (dùng `pkg/apperror`)

### Platform Layer
- [ ] Config (đọc env, có default values)
- [ ] Database (kết nối PostgreSQL)
- [ ] Logger (setup slog)

### Repository Layer
- [ ] GORM model (persistence model, có GORM tag)
- [ ] Mapper (domain ↔ model)
- [ ] Repository implementation (implement domain interface)

### Usecase Layer
- [ ] DTO (request/response, tách biệt domain)
- [ ] Usecase implementation (logic nghiệp vụ)

### Delivery Layer
- [ ] Presenter (domain → DTO response)
- [ ] Handler (Gin handler methods)
- [ ] Router (đăng ký route, middleware)
- [ ] Middleware (nếu cần auth riêng)

### Infra & Config
- [ ] Bootstrap `app.go` (wire dependency)
- [ ] Entrypoint `main.go` (cmd/)
- [ ] Migration SQL (up + down)
- [ ] `.env.example`
- [ ] `Dockerfile`
- [ ] Cập nhật `docker-compose.yml`
- [ ] Đăng ký route trên Gateway (nếu cần)

### Quality
- [ ] Unit test usecase (table-driven)
- [ ] Handler test (happy path + error)
- [ ] `go vet ./services/<service>/...` pass
- [ ] `go test ./services/<service>/...` pass
- [ ] Service chạy được local
- [ ] Service chạy được qua Docker Compose
- [ ] Health check endpoint hoạt động

---

## Phụ lục: Cấu Trúc File Hoàn Chỉnh

Sau khi hoàn thành tất cả các bước, thư mục service mới sẽ có cấu trúc:

```
services/product/
├── Dockerfile
├── cmd/product/
│   └── main.go
├── configs/
│   └── .env.example
├── migrations/
│   ├── 000001_create_products_table.up.sql
│   └── 000001_create_products_table.down.sql
└── internal/
    ├── bootstrap/
    │   └── app.go
    ├── domain/
    │   ├── product.go          # Entity
    │   ├── repository.go       # Repository interface
    │   ├── service.go          # Service interface
    │   └── errors.go           # Domain errors
    ├── usecase/
    │   ├── dto/
    │   │   └── product_dto.go  # Request/Response DTO
    │   ├── product_usecase.go  # Usecase implementation
    │   └── product_usecase_test.go
    ├── delivery/http/
    │   ├── handler/
    │   │   ├── product_handler.go
    │   │   └── product_handler_test.go
    │   ├── middleware/         # (tuỳ chọn)
    │   ├── presenter/
    │   │   └── product_presenter.go
    │   └── router.go
    ├── repository/postgres/
    │   ├── model/
    │   │   └── product_model.go
    │   ├── mapper/
    │   │   └── product_mapper.go
    │   └── product_repository.go
    └── platform/
        ├── config/
        │   └── config.go
        ├── database/
        │   └── postgres.go
        └── logger/
            └── logger.go
```

> **Tổng cộng: khoảng 15–18 file** cho một service CRUD cơ bản hoàn chỉnh.
