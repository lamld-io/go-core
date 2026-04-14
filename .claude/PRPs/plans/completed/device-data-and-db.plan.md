# Plan: Device Data & DB

## Summary
Cập nhật cơ sở dữ liệu và Model để lưu trữ thông tin thiết bị (IP, UserAgent, DeviceID) cho `RefreshToken`. Điều này giúp hệ thống lưu vết nơi đăng nhập nhằm hỗ trợ quản lý phiên làm việc của người dùng trong tương lai. Tính năng này cũng đi kèm với việc điều chỉnh giao diện `AuthService` và các handler để truyền dữ liệu Client Metadata từ request.

## User Story
As a user and sysadmin, I want the system to store device metadata (IP, UserAgent, DeviceID) when issuing a session token, so that we can implement fine-grained session revocation and trace suspicious activities.

## Problem → Solution
- **Before:** `generateTokenPair` và bảng `refresh_tokens` chỉ lưu ID, hash, và thời gian. Không thể biết token được tạo ra từ trình duyệt/máy tính nào.
- **After:** Auth handlers trích xuất User-Agent, Client IP, và Header `X-Device-ID` để gom thành struct `ClientMetadata`, truyền xuống lớp `AuthService` và được ánh xạ lưu vào database thông qua GORM.

## Metadata
- **Complexity**: Medium
- **Source PRD**: `.claude/PRPs/prds/core-security-upgrade.prd.md`
- **PRD Phase**: Phase 2 (Device Data & DB)
- **Estimated Files**: 7


---

## UX Design

### Before
N/A — internal change

### After
N/A — internal change

### Interaction Changes
| Touchpoint | Before | After | Notes |
|---|---|---|---|
| HTTP Headers | Chỉ cần Client gọi `/login` cơ bản. | Client nên truyền thêm Header `X-Device-ID` (tuỳ chọn) khi gọi `/login` hoặc `/refresh`. | `X-Device-ID` giúp định danh chính xác thiết bị. |

---

## Mandatory Reading

| Priority | File | Lines | Why |
|---|---|---|---|
| P0 | `services/auth/internal/domain/token.go` | 9-18 | Cần thêm Metadata fields vào domain struct của Token |
| P0 | `services/auth/internal/domain/service.go` | 22-38 | Cần cập nhật chữ ký `Login` và `RefreshToken`. |
| P0 | `services/auth/internal/repository/postgres/model/token_model.go` | 9-20 | Table Schema. |
| P1 | `services/auth/internal/usecase/auth_usecase.go` | 420-450 | Phương thức tạo TokenPair. |

## External Documentation

| Topic | Source | Key Takeaway |
|---|---|---|
| GORM Constraints | gorm.io docs | Sử dụng `type:varchar(length)` cho chuỗi Metadata. |

---

## Patterns to Mirror

### NAMING_CONVENTION
// SOURCE: services/auth/internal/domain/token.go:10
```go
type RefreshToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	...
	UserAgent string
	IP        string
	DeviceID  string
}
```

### REPOSITORY_PATTERN
// SOURCE: services/auth/internal/repository/postgres/model/token_model.go:10
```go
type TokenModel struct {
	...
	IP        string    `gorm:"type:varchar(64)"`
	UserAgent string    `gorm:"type:varchar(512)"`
	DeviceID  string    `gorm:"type:varchar(255)"`
}
```

---

## Files to Change

| File | Action | Justification |
|---|---|---|
| `services/auth/internal/domain/token.go` | UPDATE | Thêm struct `ClientMetadata` và update các attributes của `RefreshToken`. |
| `services/auth/internal/domain/service.go` | UPDATE | Thay đổi signature của `Login` và `RefreshToken` để nhận metadata. |
| `services/auth/internal/repository/postgres/model/token_model.go` | UPDATE | GORM schema để lưu (IP, UserAgent, DeviceID). |
| `services/auth/internal/repository/postgres/mapper/token_mapper.go` | UPDATE | Ánh xạ qua lại giữa domain entity và GORM model cho các trường mới. |
| `services/auth/internal/usecase/auth_usecase.go` | UPDATE | Nhận struct `ClientMetadata`, gắn các trường này khi gọi `generateTokenPair`. |
| `services/auth/internal/delivery/http/handler/auth_handler.go` | UPDATE | Trích xuất IP, User-Agent và X-Device-ID. |
| `services/auth/internal/delivery/http/handler/auth_handler_test.go` | UPDATE | Sửa các lệnh mock/tests khớp với interface signature thay đổi. |

## NOT Building

- API Liệt kê và Quản lý Sessions (thuộc phạm vi Phase 4).
- Phân tích vị trí địa lý của IP.

---

## Step-by-Step Tasks

### Task 1: Update Domain Models & Interfaces
- **ACTION**: Thêm `ClientMetadata` struct và sửa đổi `RefreshToken` / `AuthService`.
- **IMPLEMENT**:
  - Trong `token.go`: định nghĩa `ClientMetadata{IP, UserAgent, DeviceID}`.
  - Thêm fields đó vào `RefreshToken`.
  - Trong `service.go`: Update `Login(..., meta ClientMetadata)` và `RefreshToken(..., meta ClientMetadata)`.
- **MIRROR**: Sử dụng syntax standard Go struct hiện hữu.
- **IMPORTS**: 
- **GOTCHA**: Nên khai báo meta là tham trị, không bắt buộc truyền pointer (dễ test).
- **VALIDATE**: `go build ./internal/domain/...`

### Task 2: Update Database Model & Mapper
- **ACTION**: Update Schema `TokenModel` và `token_mapper.go`.
- **IMPLEMENT**: Bổ sung `IP`, `UserAgent`, `DeviceID` dạng chuỗi vào GORM model, rồi update hàm Map giữa Domain - Database Model để sao chép các properties này.
- **MIRROR**: REPOSITORY_PATTERN hiện có trên file.
- **IMPORTS**: None new expected.
- **GOTCHA**: GORM sẽ tự auto-migrate thay đổi, nhưng hãy chú ý limit char như `varchar(64)` cho IP/DeviceID.
- **VALIDATE**: `go build ./internal/repository/postgres/...`

### Task 3: Update Usecase Logic
- **ACTION**: Truyền ClientMetadata qua usecase flow đến hàm save Token.
- **IMPLEMENT**: 
  - Sửa chữ ký `Login` & `RefreshToken` bên trong `auth_usecase.go`.
  - Cập nhật hàm private `generateTokenPair` để nhận thêm đối số `meta domain.ClientMetadata` và gán vào `storedToken := &domain.RefreshToken{... IP: meta.IP ... }`.
- **MIRROR**: Theo current struct initialization approach.
- **IMPORTS**: Không đổi.
- **GOTCHA**: Đừng quên cập nhật tất cả code flow (register/login/refresh) đang gọi `generateTokenPair`. (Lưu ý: `Register` hiện không sinh tokens theo flow mới, nhưng nếu test mockup dùng thì cần sửa lại).
- **VALIDATE**: `go vet ./internal/usecase/...`

### Task 4: Update HTTP Handlers
- **ACTION**: Trích xuất Request info ở Gin context và truyền qua auth service.
- **IMPLEMENT**: Xây dựng helper func `getClientMeta(c *gin.Context)` để gom data: `c.ClientIP()`, `c.Request.UserAgent()`, và header `X-Device-ID`. Sửa gọi method `Login` & `RefreshToken` ở `auth_handler.go`.
- **MIRROR**: Gin Header extraction normal structure `c.GetHeader(...)`.
- **IMPORTS**: No new imports
- **GOTCHA**: Nếu `X-Device-ID` rỗng, để chuỗi rỗng thay vì UUID ngẫu nhiên để linh hoạt hơn.
- **VALIDATE**: `go build ./internal/delivery/http/...`

### Task 5: Fix AuthHandler Tests
- **ACTION**: Cập nhật lại parameters cho interface calls ở Unit Tests.
- **IMPLEMENT**: Truyền argument bù nhìn (`domain.ClientMetadata{}`) cho mọi test calls `h.authService.Login` / `h.authService.RefreshToken`. 
- **MIRROR**: Sửa chửa mọi compiler error trong `auth_handler_test.go`.
- **IMPORTS**: none new required.
- **GOTCHA**: Các package mock repository cần compile lại. Nếu có mock layer nào được generate (VD `mockery`), hãy nhớ nhắc! Tuy nhiên ở đây đang dùng mock tay `mockUserRepo`.
- **VALIDATE**: `go test ./internal/delivery/http/handler/...`

---

## Testing Strategy

### Unit Tests

| Test | Input | Expected Output | Edge Case? |
|---|---|---|---|
| Handler Login | Login Call w/ Headers | Usecase gets meta correctly | No |

### Edge Cases Checklist
- [ ] X-Device-ID không nằm trong Header.
- [ ] IP ảo từ Cloudflare (Gin `ClientIP()` xử lý đúng X-Forwarded-For).
- [ ] UserAgent siêu dài (Schema giới hạn bằng `varchar(512)` nên DB lưu an toàn).

---

## Validation Commands

### Static Analysis
```bash
go vet ./services/auth/...
```
EXPECT: Zero type errors

### Unit Tests
```bash
go test ./services/auth/... -v
```
EXPECT: All tests pass

### Database Validation (if applicable)
```bash
# Server Auth khởi động để GORM chạy AutoMigrate
cd services/auth && go run cmd/auth/main.go
```
EXPECT: Khởi động thành công, trong CSDL xem thấy `refresh_tokens` được chèn thêm cột.

---

## Acceptance Criteria
- [ ] Interface được update và mọi Test files được vá lỗi.
- [ ] Tất cả Task code hoàn thành theo đúng MIRROR.
- [ ] Compile không hề có lỗi (zero build errors).
- [ ] Bảng RefreshToken chứa Metadata sau khi login.

## Completion Checklist
- [ ] Code follows discovered patterns
- [ ] Error handling matches codebase style
- [ ] Logging follows codebase conventions
- [ ] No hardcoded values
- [ ] Self-contained — no questions needed during implementation

## Risks
| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| UserAgent quá 512 bytes bị Gorm throw error | Thấp | Vừa | Cắt chuỗi limit 512 ký tự nếu vượt ngưỡng trong Handler/Usecase trước khi lưu CSDL. |
