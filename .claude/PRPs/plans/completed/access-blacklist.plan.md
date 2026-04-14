# Plan: Access Blacklist

## Summary
Triển khai cơ chế Blacklist cho Access Token bằng Redis để khi người dùng gọi API `/logout`, Access Token hiện tại sẽ lập tức bị vô hiệu hóa, ngặn chặn việc bị lợi dụng sau khi đã đăng xuất. Đây là Phase 3 của Auth Core Security Upgrade.

## User Story
As a Người dùng cuối, I want Access Token của tôi bị chặn ngay lập tức sau khi tôi nhấn Đăng xuất, so that kẻ gian không thể dùng token đánh cắp được để truy cập dữ liệu của tôi.

## Problem → Solution
Access token (JWT) hiện tại chỉ hết hạn theo TTL tự nhiên của nó (ví dụ: 15 phút), gọi `/logout` chỉ xóa Refresh Token. → Sau chỉnh sửa, `/logout` sẽ thu hồi Refresh Token ĐỒNG THỜI lưu `jti` (JWT ID) của Access Token vào Redis Blacklist. Các API bảo vệ sẽ kiểm tra Blacklist này thông qua middleware.

## Metadata
- **Complexity**: Medium
- **Source PRD**: `.claude/PRPs/prds/core-security-upgrade.prd.md`
- **PRD Phase**: Phase 3
- **Estimated Files**: 8

---

## UX Design

### Internal Change
N/A — Đây là thay đổi nội bộ hệ thống Backend. Trải nghiệm người dùng phía client không đổi (gọi `/logout` như bình thường, nhưng request truy cập sau đó sẽ bị block).

---

## Mandatory Reading

| Priority | File | Lines | Why |
|---|---|---|---|
| P0 (critical) | `services/auth/internal/delivery/http/middleware/auth_middleware.go` | 27-68 | Nơi Access Token được xác thực. Cần cắm thêm bước kiểm tra Redis. |
| P1 (important) | `services/auth/internal/delivery/http/handler/auth_handler.go` | 155-175 | Cập nhật handler `Logout` để trích xuất token hiện tại. |
| P1 (important) | `pkg/jwt/jwt.go` | 117-135 | Hiểu cấu trúc `Claims` để lấy `claims.ID` (đóng vai trò `jti`) và `claims.ExpiresAt`. |
| P2 (reference) | `services/auth/internal/usecase/auth_usecase.go` | 343-350 | Nơi thực thi logic `Logout` hiện tại. |

## External Documentation

| Topic | Source | Key Takeaway |
|---|---|---|
| go-redis v9 | [redis.uptrace.dev](https://redis.uptrace.dev/) | Cần check `if err == redis.Nil` để xử lý khi token chưa bị blacklist mà không quăng lỗi. |

---

## Patterns to Mirror

### REPOSITORY_PATTERN
```go
// Tương tự các repository khác, tạo domain interface cho TokenBlacklist
package domain
type TokenBlacklist interface { ... }
```

### ERROR_HANDLING
```go
// Nếu Redis sập, fail-open (trả về false, nil) 
// và chỉ log warning, KHÔNG block user nếu không truy cập được redis.
if err != nil && err != redis.Nil {
    slog.ErrorContext(ctx, "failed to check token blacklist", "error", err)
    return false, nil
}
```

---

## Files to Change

| File | Action | Justification |
|---|---|---|
| `services/auth/internal/domain/service.go` | UPDATE | Thêm interface `TokenBlacklist` và đổi tham số `AuthService.Logout`. |
| `services/auth/internal/repository/redis/token_blacklist.go` | CREATE | Xử lý giao tiếp với Redis cho danh sách blacklist (set và get). |
| `services/auth/internal/usecase/auth_usecase.go` | UPDATE | Cập nhật hàm `Logout` gọi `TokenBlacklist` đồng thời inject nó vào cấu trúc usecase. |
| `services/auth/internal/delivery/http/middleware/auth_middleware.go` | UPDATE | Kiểm tra xem Token ID (`claims.ID`) có nằm trong Blacklist hay không. |
| `services/auth/internal/delivery/http/handler/auth_handler.go` | UPDATE | Truyền token chuỗi thực tế vào AuthService. |
| `services/auth/internal/delivery/http/router.go` | UPDATE | Cập nhật tham số cho `middleware.AuthMiddleware` để nhận thêm `TokenBlacklist`. |
| `services/auth/internal/bootstrap/app.go` | UPDATE | Khởi tạo repo `redis.NewTokenBlacklist` và inject vào các module. |
| Các file `_test.go` liên quan | UPDATE | Cập nhật các mock `AuthService` và `TokenBlacklist` để biên dịch thành công và test logic mới. |

## NOT Building
- Không xây dựng Refresh Token Blacklist (Refresh Token đã tự được lưu ở DB gốc).
- Không thêm API để thủ công block Token từ Admin lúc này (chỉ có user qua endpoint `logout`).

---

## Step-by-Step Tasks

### Task 1: Định nghĩa Interface TokenBlacklist & Cập nhật Domain AuthService
- **ACTION**: Thêm file / cập nhật file Interface. 
- **IMPLEMENT**:
  - Tại `domain/service.go` hoặc tạo file `domain/token_blacklist.go`: thêm `TokenBlacklist` với `BlacklistToken(ctx, tokenID string, expiresAt time.Time) error` và `IsBlacklisted(ctx, tokenID string) (bool, error)`.
  - Thay đổi `Logout(ctx context.Context, userID uuid.UUID, accessTokenStr string) error` trong `domain.AuthService`.
- **VALIDATE**: `go build` có thể báo lỗi do chưa implement đồng bộ ở các file khác.

### Task 2: Cài đặt Redis Token Blacklist Repository
- **ACTION**: Tạo file `services/auth/internal/repository/redis/token_blacklist.go`.
- **IMPLEMENT**: 
  - Struct `tokenBlacklistRepo` nhận `*redis.Client`. 
  - Nếu `client == nil`, return early không lỗi (fail-open) như thiết kế trong PR.
  - Sử dụng khóa `blacklist:token:<tokenID>`.
- **IMPORTS**: `"github.com/redis/go-redis/v9"`, `"log/slog"`, `"time"`, `"context"`, `"fmt"`
- **VALIDATE**: Code định nghĩa struct này pass kiểm tra 문 pháp `go build`.

### Task 3: Cập nhật Usecase và Handler cho việc Logout
- **ACTION**: Bổ sung token vào `Logout`.
- **IMPLEMENT**:
  - `NewAuthUsecase` phải nhận `domain.TokenBlacklist` làm tham số và lưu vào struct `authUsecase`.
  - Trong logic hàm `uc.Logout`, validate JWT string đưa vào, trích xuất `Claims`, và gọi `BlacklistToken` với TTL = `ExpiresAt.Time` trừ đi thời gian hiện tại.
  - Sửa `AuthHandler.Logout` lấy token thô từ `Bearer` Header (tùy theo logic extract có sẵn ở middleware hoặc tách ra). Bỏ token đó vào `h.authService.Logout`.
- **GOTCHA**: Nếu giải mã bằng `ValidateToken` failed, vẫn thử thu hồi refresh token để người dùng đăng xuất triệt để.

### Task 4: Cập nhật AuthMiddleware để kiểm tra Blacklist
- **ACTION**: Dính `TokenBlacklist` vào middleware.
- **IMPLEMENT**: 
  - Sửa signature `AuthMiddleware(jwtManager *pkgjwt.Manager, blacklist domain.TokenBlacklist) gin.HandlerFunc`.
  - Sau hàm `jwtManager.ValidateToken`, gọi `blacklist.IsBlacklisted(c.Request.Context(), claims.ID)`.
  - Nếu blacklist return true -> block với mã lỗi 401: `apperror.Unauthorized("token has been revoked")`.
- **IMPORTS**: `domain` package.
- **VALIDATE**: Lỗi phải trả về đúng mã 401 khi test thủ công (sẽ cập test ở task 6).

### Task 5: Đi dây (Wire-up) tại Bootstrap và Router
- **ACTION**: Khởi tạo component ở `app.go`
- **IMPLEMENT**:
  - Trong `NewApp`, khởi tạo `tokenBlacklist := redisrepo.NewTokenBlacklist(redisClient)`.
  - Inject vào `NewAuthUsecase` và tiếp tục đưa `tokenBlacklist` đi qua `NewRouter`.
  - Truyền `tokenBlacklist` vào `AuthMiddleware(jwtManager, tokenBlacklist)`.
- **VALIDATE**: `go build ./cmd/auth` phải thành công mà không có syntax error.

### Task 6: Vá và cải tiến Unit Tests
- **ACTION**: Điều chỉnh Mock structs.
- **IMPLEMENT**:
  - Tại `auth_usecase_test.go`: Tạo Mock `TokenBlacklist` implementation cho struct. Update init `NewAuthUsecase`.
  - Tại `auth_handler_test.go`: Cập nhật mock `AuthService` với signature `Logout` mới. Khởi tạo `NewRouter` có thêm Mock `TokenBlacklist`.
- **VALIDATE**: `go test ./...` qua sạch sẽ.

---

## Testing Strategy

### Unit Tests
| Test | Input | Expected Output | Edge Case? |
|---|---|---|---|
| Usecase - Logout | Token hợp lệ | Calls Revoke Refresh & Blacklist Access | X |
| Middleware - Check | Token có ID bị blacklist | 401 Unauthorized | |
| Middleware - Check | Redis fail/nil | Bypass, cho phép tiếp tục | Có (Fail-open) |

### Edge Cases Checklist
- [x] Redis sập (cần handle fail-open mà không báo block)
- [x] Tính TTL cho Redis <= 0 (token đã tự outdate, không cần save redis)
- [x] Bắn request `/logout` bằng token đã hết hạn -> vẫn xóa refresh tokens DB

---

## Validation Commands

### Static Analysis
```bash
go vet ./...
```
EXPECT: Zero type errors

### Unit Tests
```bash
go test ./... -v
```
EXPECT: All tests pass

### Full Test Suite / Build
```bash
go build -o auth-server.exe ./cmd/auth
```
EXPECT: Biên dịch thành công.

---

## Acceptance Criteria
- [ ] Tất cả Tasks được lập trình xong
- [ ] Middleware kiểm tra Redis Token Blacklist hoạt động chuẩn
- [ ] Tính năng fail-open của Redis vẫn bảo vệ hệ thống không uptime error
- [ ] Unit test cho middleware và handler verify token bị hủy được vượt qua toàn bộ.

## Completion Checklist
- [ ] Code follows discovered patterns (Redis Fail-open).
- [ ] Error handling matches codebase style (trả về struct response đúng của Gin).
- [ ] No hardcoded values.
- [ ] Self-contained — Codebase sẳn sàng thực thi ngay Phase này qua Plan.

## Risks
| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Khởi tạo Redis cache có thể trả panic nếu mock thiếu kỹ | Vừa | Build Failed | Xây dựng implementation Mock cho `TokenBlacklist` chuẩn trước khi chạy UT. |
| Delay kết nối Redis làm chậm Request Authentication | Thấp | Low | Mặc định Go-redis tự caching connections và RTT nội mạng nhỏ hơn 1ms. |
