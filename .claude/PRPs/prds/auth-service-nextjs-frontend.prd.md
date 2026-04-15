# Auth Service Next.js Frontend

## Problem Statement

QA nội bộ, dev tích hợp, và một phần người dùng cuối hiện chưa có giao diện tương ứng với auth service hiện tại, nên phải kiểm thử hoặc demo bằng Postman/Swagger thay vì đi qua trải nghiệm thực tế. Chi phí của việc không giải quyết là auth flow khó demo end-to-end, khó kiểm thử trải nghiệm người dùng, và khó tái sử dụng làm nền tảng cho app frontend sau này.

## Evidence

- Hiện chưa có thư mục `frontend/` trong repo; chưa có frontend app sẵn để tiêu thụ auth service.
- Auth service đã có nhóm endpoint đầy đủ cho public và protected flows tại `services/auth/internal/delivery/http/router.go:29`.
- Test hiện tại xác nhận các flow cốt lõi đã chạy end-to-end ở mức API: đăng ký, verify email, login, forgot/reset password, lockout policy tại `services/auth/internal/delivery/http/handler/auth_handler_test.go:406`.
- Assumption - needs validation through stakeholder review: giao diện này sẽ được dùng thường xuyên bởi QA nội bộ, dev tích hợp, và cho demo sản phẩm.

## Proposed Solution

Tạo một frontend app mới bằng Next.js trong thư mục `frontend/`, bám sát auth API hiện tại thay vì thay đổi backend trước. Giải pháp ưu tiên một auth portal rõ trạng thái, hỗ trợ đầy đủ các flow cốt lõi như register, login, verify email, forgot/reset password, profile, sessions, và 2FA; cách tiếp cận này phù hợp hơn các lựa chọn hiện tại như Postman/Swagger vì cho phép kiểm thử, demo, và tái sử dụng trên trải nghiệm gần với sản phẩm thực tế.

## Key Hypothesis

We believe a dedicated Next.js auth portal wired to the existing auth service APIs will reduce friction in testing and demoing authentication flows for internal QA, integration developers, and early end users.
We'll know we're right when the full MVP auth journey can be completed end-to-end through the UI without Postman/Swagger for the core flows, and internal users prefer the UI for demo/test scenarios.

## What We're NOT Building

- Full backoffice administration - không phải mục tiêu chính của giao diện này trong v1.
- New auth backend capabilities - v1 bám API hiện tại, không mở rộng backend nếu chưa có bằng chứng cần thiết.
- Production-grade customer identity platform features như social login, SSO, RBAC management UI - chưa cần để validate nhu cầu trước mắt.

## Success Metrics

| Metric | Target | How Measured |
|--------|--------|--------------|
| Core auth flows completed via UI | 100% các flow MVP chạy end-to-end | Checklist kiểm thử thủ công/E2E trên UI |
| Reduced reliance on technical tools | QA/dev có thể demo và test không cần Postman/Swagger cho flow MVP | UAT nội bộ và sign-off từ stakeholder |
| Time to demo auth journey | TBD - needs research | So sánh thời gian thao tác trước và sau khi có UI |

## Open Questions

- [ ] Frontend có cần SSR cho các màn protected hay chỉ cần client-side app với token storage phù hợp?
- [ ] Token nên được lưu bằng cookie, memory, hay local storage trong bối cảnh hiện tại?
- [ ] `GET /api/v1/auth/validate` có phải endpoint chuẩn để hydrate auth state trên frontend hay chỉ dành cho internal service call?
- [ ] Giao diện admin cho `lockout-policy` có thuộc MVP hay nên tách sau?
- [ ] Có cần route chuyên biệt cho verify/reset qua query params từ email link hay chỉ cần form nhập token ở giai đoạn đầu?

---

## Users & Context

**Primary User**
- **Who**: QA nội bộ, dev tích hợp, và người dùng cuối ở bối cảnh early/internal rollout.
- **Current behavior**: Gọi API trực tiếp bằng Postman/Swagger hoặc test ở mức backend thay vì dùng UI.
- **Trigger**: Cần demo, kiểm thử, hoặc tích hợp nhanh một auth flow cụ thể.
- **Success state**: Có thể hoàn thành flow auth cần thiết qua giao diện rõ ràng, có feedback trạng thái và lỗi đúng ngữ cảnh.

**Job to Be Done**
When I need to test, demo, or integrate authentication flows, I want to use a real frontend wired to the current auth APIs, so I can validate the end-to-end experience without relying on technical API tools.

**Non-Users**
Admin vận hành sâu không phải đối tượng chính của v1. Các nhu cầu backoffice phức tạp nên được bỏ qua trừ khi trực tiếp phục vụ việc xác thực flow hiện tại.

---

## Solution Detail

### Core Capabilities (MoSCoW)

| Priority | Capability | Rationale |
|----------|------------|-----------|
| Must | App Next.js mới trong `frontend/` | Đây là ràng buộc kỹ thuật đã được xác nhận |
| Must | Public auth flows: register, login, verify email, forgot/reset password | Đây là lõi của trải nghiệm auth hiện tại |
| Must | Handle auth states như `requires_email_verification`, `requires_2fa`, token refresh, lỗi xác thực | DTO hiện tại đã biểu diễn rõ các trạng thái này |
| Must | Protected user area: profile, sessions, logout | Có API sẵn và cần để chứng minh lifecycle đăng nhập hoàn chỉnh |
| Should | 2FA setup và verify qua UI | API đã có, giúp portal phản ánh đầy đủ auth capabilities hiện tại |
| Should | Route-based UX cho email verification và reset password | Giảm ma sát khi dùng link từ email |
| Could | Admin lockout policy screen | Có giá trị cho demo/test nhưng không phải nhu cầu phổ quát |
| Won't | Social login / SSO / enterprise auth | Chưa có bằng chứng hoặc API scope tương ứng trong hiện trạng |

### MVP Scope

MVP là một Next.js app trong `frontend/` cho phép người dùng nội bộ hoàn thành các flow sau bằng UI: đăng ký, verify email, đăng nhập, xử lý trường hợp cần 2FA, quên mật khẩu, đặt lại mật khẩu, xem profile, xem sessions, logout. Nếu thời gian hạn chế, màn hình quản trị `lockout-policy` và polish sâu về design system sẽ được để sau.

### User Flow

Người dùng truy cập auth portal, chọn đăng ký hoặc đăng nhập. Nếu đăng ký, họ nhận trạng thái chờ verify email và tiếp tục qua flow verify. Nếu đăng nhập mà tài khoản yêu cầu 2FA, UI chuyển sang bước nhập mã bằng `temp_token`. Sau khi đăng nhập thành công, người dùng vào khu vực protected để xem profile, sessions, bật 2FA nếu cần, và logout. Nếu quên mật khẩu, họ đi qua flow yêu cầu reset rồi đặt mật khẩu mới.

---

## Technical Approach

**Feasibility**: HIGH

**Architecture Notes**
- Tạo app mới bằng Next.js trong `frontend/` thay vì gắn vào service Go hiện tại.
- Frontend bám các route có sẵn tại `services/auth/internal/delivery/http/router.go:31`.
- UI cần xử lý response đặc thù hiện có như `RegisterResponse.RequiresEmailVerification`, `LoginResponse.Requires2FA`, `LoginResponse.TempToken`, `SessionResponse`, `Setup2FAResponse` tại `services/auth/internal/usecase/dto/auth_dto.go:81`.
- Gateway/config hiện có cho thấy repo đang đi theo kiến trúc service tách biệt; frontend nên coi auth service là upstream độc lập, không nhúng logic auth vào backend UI mới.
- TBD - needs research: chọn App Router hay Pages Router, cơ chế env/config cho base URL auth service, và chiến lược token storage phù hợp.

**Technical Risks**

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Chưa có frontend baseline trong repo | H | Giữ MVP nhỏ, tạo cấu trúc Next.js tối thiểu trước rồi map từng flow |
| Mơ hồ về token handling trên Next.js | H | Spike sớm cho auth state, refresh flow, và persistence strategy |
| Flow verify/reset phụ thuộc token từ email | M | Thiết kế route nhận token từ query param và fallback form nhập token |
| Lệch kỳ vọng giữa UI và API error semantics | M | Bám test/DTO hiện có và review với stakeholder trước khi polish UX |
| Scope creep sang admin/security portal | M | Giữ `lockout-policy` ngoài MVP mặc định nếu chưa có yêu cầu bắt buộc |

---

## Implementation Phases

<!--
  STATUS: pending | in-progress | complete
  PARALLEL: phases that can run concurrently (e.g., "with 3" or "-")
  DEPENDS: phases that must complete first (e.g., "1, 2" or "-")
  PRP: link to generated plan file once created
-->

| # | Phase | Description | Status | Parallel | Depends | PRP Plan |
|---|-------|-------------|--------|----------|---------|----------|
| 1 | Frontend Bootstrap | Tạo app Next.js trong `frontend/`, cấu hình env, base layout, API client foundation | complete | - | - | `.claude/PRPs/plans/completed/auth-service-nextjs-frontend-bootstrap.plan.md` |
| 2 | Public Auth Flows | Implement register, login, verify email, forgot/reset password, lỗi và state cơ bản | pending | - | 1 | - |
| 3 | Protected User Flows | Implement profile, logout, sessions, auth guards, refresh handling | pending | with 4 | 2 | - |
| 4 | 2FA Flows | Implement login 2FA, setup 2FA, verify 2FA setup | pending | with 3 | 2 | - |
| 5 | Validation & Hardening | E2E/manual validation, UX cleanup, docs/run instructions | pending | - | 3, 4 | - |

### Phase Details

**Phase 1: Frontend Bootstrap**
- **Goal**: Có một app Next.js chạy được trong `frontend/` và kết nối được tới auth service.
- **Scope**: Scaffold app, cấu hình env, routing foundation, layout, HTTP client, error envelope parsing.
- **Success signal**: App chạy local và gọi được ít nhất một auth endpoint thành công.
- **Report**: `.claude/PRPs/reports/auth-service-nextjs-frontend-bootstrap-report.md`

**Phase 2: Public Auth Flows**
- **Goal**: Bao phủ các hành trình công khai cốt lõi của auth.
- **Scope**: Register, login, verify email, resend verification, forgot password, reset password, hiển thị auth errors.
- **Success signal**: Người dùng có thể hoàn thành public flows mà không cần tool API thủ công.

**Phase 3: Protected User Flows**
- **Goal**: Bao phủ lifecycle sau đăng nhập.
- **Scope**: Auth state bootstrap, protected route gating, profile, sessions list, session revoke, logout.
- **Success signal**: Sau login thành công, người dùng có thể quản lý session cơ bản qua UI.

**Phase 4: 2FA Flows**
- **Goal**: Phản ánh đầy đủ auth capability hiện có liên quan đến 2FA.
- **Scope**: Chuyển bước khi login trả về `requires_2fa`, nhập code với `temp_token`, setup 2FA, verify 2FA setup.
- **Success signal**: Các tài khoản bật 2FA đi được hết flow mà không cần thao tác backend thủ công.

**Phase 5: Validation & Hardening**
- **Goal**: Xác nhận MVP dùng được cho demo và test nội bộ.
- **Scope**: Manual test matrix hoặc E2E, polish UX tối thiểu, tài liệu chạy app.
- **Success signal**: Stakeholder nội bộ sign off rằng auth portal đủ để demo/test flow chính.

### Parallelism Notes

Phase 3 và 4 có thể triển khai song song sau khi public flows và auth foundation ở Phase 2 ổn định, vì cùng phụ thuộc vào cơ chế auth state nhưng tác động đến các phần màn hình khác nhau. Phase 5 phải chờ cả hai phase này để tránh xác nhận trên trải nghiệm chưa hoàn chỉnh.

---

## Decisions Log

| Decision | Choice | Alternatives | Rationale |
|----------|--------|--------------|-----------|
| Frontend location | `frontend/` | Tạo trong service hiện có, repo riêng | Đã được xác nhận bởi user |
| Frontend framework | Next.js | React thuần, Vite, framework khác | Đã được xác nhận bởi user |
| Backend scope | Reuse auth API hiện tại | Mở rộng API trước | Mục tiêu là tạo UI tương ứng với API đang có |
| Initial target users | QA nội bộ, dev tích hợp, early end users | Chỉ end users production, chỉ admin | Phù hợp với pain hiện tại quanh test/demo/integration |
| Admin lockout UI | Deferred by default | Build ngay trong MVP | Có API sẵn nhưng không phải nhu cầu phổ quát nhất |

---

## Research Summary

**Market Context**
Các hệ auth phổ biến như Firebase, Auth0, Supabase đều tổ chức auth UI quanh các flow chuẩn: sign up, sign in, verification, reset password, session handling, và MFA tùy chọn. Pattern tốt là biểu diễn rõ trạng thái auth đặc thù thay vì chỉ render form tĩnh; anti-pattern là bỏ qua các state như verify email hoặc 2FA, khiến flow bị đứt khi đi từ API sang UI.

**Technical Context**
Auth service hiện có đủ endpoint để dựng auth portal hoàn chỉnh ở mức MVP: public flows, protected flows, sessions, 2FA, và một phần security policy. DTO và test hiện có cung cấp đủ ngữ nghĩa để thiết kế UI phản ánh đúng response contract mà chưa cần thay đổi backend. Điểm chưa chắc lớn nhất là frontend architecture details của Next.js app mới vì repo chưa có baseline frontend.

---

*Generated: 2026-04-15 14:48:37 +07:00*
*Status: DRAFT - needs validation*
