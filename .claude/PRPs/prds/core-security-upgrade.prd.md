# Auth Core Security Upgrade (Nâng cấp Bảo mật Cốt lõi)

## Problem Statement (Tuyên bố Vấn đề)

Hệ thống Auth hiện tại đang đối mặt với 3 rủi ro bảo mật nghiêm trọng: (1) Dễ bị tấn công Brute-force & DDoS vì không có Rate Limiting (Giới hạn lượt truy cập) theo IP, (2) Nếu người dùng bị lộ mật khẩu, tài khoản sẽ bị chiếm đoạt hoàn toàn do thiếu Xác thực đa yếu tố (2FA/MFA), và (3) Hệ thống hiện tại không cho phép theo dõi và đăng xuất từng thiết bị cụ thể (Device Session Management), đồng thời Access Token đã cấp không thể bị thu hồi trước khi hết hạn (thiếu Token Blacklisting). Nếu không giải quyết, chi phí khắc phục sự cố rò rỉ dữ liệu và downtime hệ thống sẽ rất lớn.

## Evidence (Bằng chứng & Dữ liệu)

- **Assumption - needs validation qua Penetration Testing**: API `/login` và `/register` hiện không có bất cứ middleware nào ngăn chặn flood requests từ một IP.
- **Quan sát từ Source Code (`auth_usecase.go`)**: Hàm `Logout` đang xoá toàn bộ refresh tokens của user qua `RevokeByUserID()`, không truyền thông tin DeviceID, IP, hay UserAgent, làm mất trải nghiệm người dùng khi phải login lại ở mọi thiết bị.
- **Tiêu chuẩn bảo mật ngành (OWASP)**: Multi-Factor Authentication (MFA) và Rate Limiting là những yêu cầu bắt buộc tối thiểu đối với hệ thống định danh (IAM).

## Proposed Solution (Giải pháp đề xuất)

Chúng ta sẽ triển khai gói Nâng cấp Bảo mật Cốt lõi bao gồm 3 tính năng:
1. **Global IP-based Rate Limiting**: Thiết lập Middleware ở Gateway/Router sử dụng Redis-cell hoặc token bucket để cản lọc tần suất request bất thường vào các API nhạy cảm.
2. **Session Management & Access Token Blacklist**: Cập nhật DB schema để lưu `DeviceID`, `IP`, `UserAgent` đi kèm `RefreshToken`. Xây dựng Redis cache để lưu các Access Token bị thu hồi chủ động.
3. **TOTP-based 2FA (Xác thực 2 bước)**: Cho phép người dùng bật 2FA bằng Google Authenticator/Authy. Endpoint login sẽ trả về token trung gian nếu 2FA được bật, yêu cầu gọi thêm endpoint `/verify-2fa`.

## Key Hypothesis (Giả thuyết cốt lõi)

Chúng ta tin rằng việc **cung cấp Rate Limiting, 2FA và Session Management** sẽ **ngăn chặn hoàn toàn các cuộc tấn công Brute-force/Credential Stuffing và Account Takeover** cho **người dùng và hệ thống**.
Chúng ta sẽ biết mình đúng khi **đạt 100% tỷ lệ cản lọc bot tấn công (0 downtime) và người dùng có thể chủ động ngắt kết nối thiết bị lạ mà không ảnh hưởng thiết bị khác.**

## What We're NOT Building (Điều gì nằm ngoài phạm vi - Out of Scope)

- **OAuth2 / Social Login (Sign in with Google/Apple)** - Quá phức tạp và thiên về User Acquisition hơn là Security lúc này.
- **SMS OTP / Email OTP cho 2FA** - Tốn kém chi phí (SMS) và kém an toàn hơn so với Authenticator App (TOTP). Only focus on TOTP (Time-Based One-Time Password).
- **Phân tích hành vi / Dò tìm bắt thường (Anomaly Detection AI)** - Cảnh báo tự động cấu trúc phức tạp. Trong V1, user sẽ tự kiểm tra thiết bị thủ công.

## Success Metrics (Chỉ số Thành công)

| Metric (Chỉ số) | Target (Mục tiêu) | How Measured (Cách đo lường) |
|--------|--------|--------------|
| Tốc độ khoá IP tấn công | < 5 giây | Logging/Metrics từ Rate Limiting Middleware |
| Downtime do quá tải API Login | 0% | Server monitoring (Grafana/Prometheus) |
| Tỷ lệ lỗi khi revoke Session | 0% | Kiểm tra API /sessions và Redis Blacklist TTL |

## Open Questions (Câu hỏi Mở)

- [ ] Rate Limit threshold (ngưỡng giới hạn) hợp lý cho API Login là bao nhiêu? (ví dụ: 5 requests / phút / IP?)
- [ ] Thời gian sống (TTL) của Access Token hiện tại là bao nhiêu để cấu hình Redis Blacklist tối ưu về RAM?

---

## Users & Context (Người dùng & Bối cảnh)

**Primary User (Người dùng chính)**
- **Ai**: Quản trị viên hệ thống (Người bảo vệ) và Người dùng cuối (Người bảo mật dữ liệu cá nhân).
- **Hành vi hiện tại**: Devops phải can thiệp WAF block IP bằng tay khi có bão requests. Người dùng khôi phục tài khoản thủ công qua email khi bị hack.
- **Kích hoạt (Trigger)**: Cảm thấy bất an khi đăng nhập trên máy tính công cộng, hoặc khi máy chủ báo động CPU load cao do Spam.
- **Trạng thái thành công**: Server yên ổn loại bỏ requests lạ trong im lặng. Người dùng có danh sách thiết bị gọn gàng và 2FA an tâm.

**Job to Be Done**
Khi **tài khoản của tôi đối mặt với rủi ro rò rỉ trên internet**, tôi muốn **kiểm soát lớp bảo mật bằng 2FA và Quản lý Thiết bị**, để tôi có thể **đảm bảo dữ liệu kinh doanh/cá nhân của mình không lọt vào tay kẻ xấu.**

**Non-Users**
Người dùng ẩn danh (Guest) - Các tính năng 2FA và Session Management không áp dụng cho họ.

---

## Solution Detail (Chi tiết Giải pháp)

### Core Capabilities (Độ ưu tiên - MoSCoW)

| Priority | Capability (Tính năng) | Rationale (Lý do) |
|----------|------------|-----------|
| Must | IP-based Rate Limiter Middleware | Chống sập hệ thống (DDoS/Brute-force). |
| Must | Đăng xuất/Thu hồi phiên trên từng thiết bị | Phục vụ Single Logout đúng nghĩa cho 1 user. |
| Must | Tích hợp Redis Token Blacklist | Xác thực vô hiệu hóa JWT Access Token ngay lập tức. |
| Must | Luồng bật/tắt và sử dụng TOTP 2FA | User tự bảo vệ account của mình. |
| Should| API danh sách "Lịch sử phiên đăng nhập" | Cho phép Client App hiển thị UI quản lý thiết bị. |
| Won't | Đăng nhập bằng SMS OTP | Chi phí cao, SMS có thể bị chặn/chuyển hướng (SIM Swap). |

### MVP Scope

1. Thêm Gin Middleware `rate-limit` dùng `go-redis` hoặc memory cache cho các endpoint `:auth`.
2. Sửa Schema cho `RefreshToken`, ghi lại IP, UserAgent.
3. Viết Middleware check ID (`jti` hoặc token hash) trong Redis cache. Khi Logout, ghi Access Token vào Redis Blacklist kèm TTL = thời gian sống còn lại của token.
4. Thêm field `totp_secret`, `is_2fa_enabled` vào model `User`. Bổ sung hàm `/auth/2fa/setup`, `/auth/2fa/verify`, update endpoint `/login` theo luồng 2FA.

### User Flow (Luồng nghiệp vụ Core 2FA Login)

1. User nhập Email + Pass ở API `/login`.
2. Server check Pass thành công. Kiểm tra `is_2fa_enabled == true`.
3. Server không cấp TokenPair, mà trả về `temp_token` (JWT có hạn 2 phút) kèm thông báo "Requires 2FA".
4. User đưa `temp_token` và `OTP` từ Authenticator App vào API `/login/2fa`.
5. Server check OTP. Đúng -> Cấp `AccessToken` và `RefreshToken`.

---

## Technical Approach (Cách tiếp cận Kỹ thuật)

**Feasibility (Khả thi)**: CAO (HIGH) - Các chuẩn thư viện Go có sẵn như `pquerna/otp` (TOTP), `go-redis/redis_rate` và cấu trúc usecase hiện tại rất tách bạch dễ mở rộng.

**Architecture Notes (Ghi chú Kiến trúc)**
- Hệ thống cần một Redis Server (nếu chưa có) để chia sẻ Rate Limit Cache và Blacklist Cache giữa các in-stance.
- Access Token Middleware sẽ phải đọc Redis cho mỗi request đến các Private Endpoint -> Cần tối ưu connection pool (Redis rất nhanh nên độ trễ dưới 1ms có thể chấp nhận).

**Technical Risks (Rủi ro Kỹ thuật)**

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Redis sập sẽ block mọi API (Single Point of Failure) | Thấp (L) | Nếu lỗi Redis connection, bỏ block Blacklist/RateLimit (Fail-Open mode), ưu tiên business vẫn chạy được bằng logic Validate Signature thuần |
| Sai lệch thời gian Clock Skew ở TOTP | Vừa (M) | Dùng time window +/- 30s khi verify TOTP code. |

---

## Implementation Phases (Các giai đoạn triển khai)

| # | Phase | Description | Status | Parallel | Depends | PRP Plan |
|---|-------|-------------|--------|----------|---------|----------|
| 1 | Rate Limiting | Cài đặt & cấu hình Gin Rate Limiter Middleware qua Redis | complete | - | - | .claude/PRPs/reports/rate-limiting-report.md |
| 2 | Device Data & DB | Cập nhật Table RefreshToken (IP, Agent, DeviceID) | complete | with 1 | - | .claude/PRPs/reports/device-data-and-db-report.md |
| 3 | Access Blacklist | Cơ chế Revoke Token và Middleware check Blacklist | complete | with 4 | 2 | .claude/PRPs/reports/access-blacklist-report.md |
| 4 | Session UI API | Các API Get / Delete Session cho Client Apps | pending | with 3 | 2 | - |
| 5 | TOTP 2FA Core | Tích hợp thư viện OTP, API Enable/Verify 2FA | pending | - | 1, 2 | - |
| 6 | 2FA Login Flow | Tái cấu trúc hàm AuthUsecase.Login hỗ trợ luồng 2 bước | pending | - | 5 | - |

### Phase Details

**Phase 1: Rate Limiting**
- **Goal**: Chặn brute-force `/login` và spam email.
- **Scope**: Golang middleware chặn request theo IP. Fail-over nếu Redis lỗi. Cấu hình linh hoạt.
- **Success signal**: Dùng script curl spam API liên tục bị trả mã lỗi `429 Too Many Requests`.

**Phase 2: Device Data & DB**
- **Goal**: Lưu vết nơi đăng nhập.
- **Scope**: Update Ent/GORM schema, chạy migration.
- **Success signal**: Các row refresh token mới tạo có đầy đủ thông tin UserAgent và IP.

**Phase 3: Access Blacklist**
- **Goal**: Chặn Access Token đã bị logout.
- **Scope**: Khi gọi `/logout`, lấy Access Token từ request -> Đưa hash/ID vào Redis. Update AuthMiddleware.
- **Success signal**: Logout xong lấy Token cũ chạy tiếp nhận mã `401 Unauthorized`.

---

## Decisions Log (Nhật ký Quyết định Đầu vào)

| Decision | Choice | Alternatives | Rationale |
|----------|--------|--------------|-----------|
| Cách giới hạn tốc độ | Token Bucket qua Redis | Golang `time/rate` local memory | Vì application chạy qua docker/K8s multiple pods, local memory rate limit sẽ bị lách nếu round-robin. Phải dùng Redis. |
| Công nghệ Token Blacklist | REDIS | Postgres Database | Access Token cần verify ở mọi API request. Truy vấn DB quá nặng. Redis tốc độ O(1) in-memory phù hợp với short-TTL access tokens. |
| Loại MFA | TOTP App | SMS OTP, Email OTP | Miễn phí dịch vụ bên thứ 3. An toàn không dính lỗi Sim-swap. Dễ maintain hệ thống ngoài vùng. |

---

## Research Summary (Tóm tắt Khảo sát)

**Market Context**
Hầu hết các hệ thống IAM hàng đầu (Auth0, Clerk, Keycloak) luôn ưu tiên thiết kế Quản lý Phiên (Session) trên gốc DB ngay từ đầu và đều mặc định hỗ trợ TOTP. SMS đang dần bị khai tử do rủi ro an ninh mạng cao (SIM Hijacking) và tốn kém chi phí viễn thông cho start-up.

**Technical Context**
Trong UseCase hiện tại `auth_usecase.go`, module quản lý token (`generateTokenPair`, `RefreshToken`, `RevokeByUserID`) khá sạch (clean). Để làm 2FA, ta chỉ việc tách tác vụ Login thành: "Kiểm tra pass -> IF 2fa return tempToken -> IF not cấp AuthTokens", sửa API rất ít. Sự phức tạp sẽ dồn ở phase Update Schema Auth_Refresh_Tokens và cài Redis.

---

*Generated: 2026-04-14*
*Status: DRAFT - needs validation*
