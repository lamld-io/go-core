<!-- Generated: 2026-04-14 | Files scanned: 52 | Token estimate: ~300 -->

# Frontend Architecture

> **Không có frontend trong repo này.**  
> `github.com/base-go/base` là **backend-only** — cung cấp REST API thông qua Gateway và Auth Service.

## API Consumer Pattern

Các client (frontend, mobile app) tương tác với hệ thống qua:

```
Client App (any)
    │
    ▼  HTTP/JSON  (port 8080)
Gateway Service
    │
    ├─ /api/v1/auth/* → Auth Service (public endpoints)
    │
    └─ /* (other routes) → Downstream services (via routes.yaml)
```

## Authentication Flow (cho frontend)

```
1. POST /api/v1/auth/register  → nhận user info
2. POST /api/v1/auth/login     → nhận { access_token, refresh_token }
3. GET  /api/v1/auth/validate  → kiểm tra token còn hợp lệ không
4. POST /api/v1/auth/refresh   → làm mới access_token (dùng refresh_token)
5. POST /api/v1/auth/logout    → thu hồi refresh_token
```

## Response Format

Tất cả response dùng format chuẩn từ `pkg/response`:

```json
{
  "success": true,
  "data": { ... },
  "error": null
}
```

## Notes cho Frontend Developer

- Access token TTL: **15 phút** — cần implement token refresh tự động
- Refresh token TTL: **7 ngày**
- JWT thuật toán: **RS256** (asymmetric)
- Email verification bắt buộc sau khi đăng ký
- Account lockout sau nhiều lần đăng nhập thất bại (cấu hình qua API)
