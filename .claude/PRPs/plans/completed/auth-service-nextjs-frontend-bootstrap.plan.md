# Plan: Auth Service Next.js Frontend Bootstrap

## Summary
Phase này thiết lập nền móng cho frontend app mới trong `frontend/` dùng Next.js, đủ để chạy local, cấu hình environment, dựng routing/layout cơ bản, và có API client hiểu đúng response contract của auth service. Mục tiêu là sau phase này, implementation các flow auth ở các phase sau có thể bám trực tiếp vào cấu trúc app, error envelope, và endpoint contracts mà không cần tìm lại codebase.

## User Story
As an internal QA or integration developer, I want a bootstrapped Next.js frontend wired to the current auth service contract, so that future auth flows can be implemented and validated through a real UI instead of API tools.

## Problem → Solution
Repo hiện chỉ có backend Go services và chưa có `frontend/` app để tiêu thụ auth APIs → Tạo một Next.js app có cấu trúc route/layout rõ ràng, env rõ ràng, và API foundation bám chính xác backend response envelope.

## Metadata
- **Complexity**: Large
- **Source PRD**: `.claude/PRPs/prds/auth-service-nextjs-frontend.prd.md`
- **PRD Phase**: `Frontend Bootstrap`
- **Estimated Files**: 12-16

---

## UX Design

### Before
```text
┌─────────────────────────────┐
│ No frontend app exists      │
│ No routes, no layout,       │
│ no auth-aware client        │
│ QA/dev must call APIs       │
│ through Postman/Swagger     │
└─────────────────────────────┘
```

### After
```text
┌─────────────────────────────┐
│ Next.js app in /frontend    │
│ Root layout + route groups  │
│ Shared API client + env     │
│ Auth shell ready for flows  │
│ Home page proves connectivity│
└─────────────────────────────┘
```

### Interaction Changes
| Touchpoint | Before | After | Notes |
|---|---|---|---|
| Repo root | Chỉ có Go services | Có thêm `frontend/` Next.js app | New app, not extension of Go service |
| Local startup | Chạy `auth-service`/`gateway` riêng | Chạy thêm `frontend` dev server | Likely `http://localhost:3000` |
| Auth API consumption | Manual via Postman/Swagger | Qua shared API client | Client phải parse `code/message/data` envelope |
| Routing | Không có UI routes | Có route foundation cho public/protected areas | Use App Router route groups |

---

## Mandatory Reading

Files that MUST be read before implementing:

| Priority | File | Lines | Why |
|---|---|---|---|
| P0 (critical) | `services/auth/internal/delivery/http/router.go` | 29-65 | Canonical auth endpoint surface and public/protected grouping |
| P0 (critical) | `pkg/response/response.go` | 11-81 | Standard JSON success/error envelope the frontend must parse |
| P0 (critical) | `services/auth/internal/usecase/dto/auth_dto.go` | 1-144 | Request/response contracts for all auth flows |
| P0 (critical) | `services/auth/internal/delivery/http/handler/auth_handler.go` | 33-176 | Actual status codes and payload semantics for public auth endpoints |
| P1 (important) | `services/auth/internal/delivery/http/middleware/auth_middleware.go` | 14-93 | Authorization header contract and protected-route expectations |
| P1 (important) | `docker-compose.yml` | 23-80 | Local ports and how services are expected to run together |
| P1 (important) | `services/gateway/configs/routes.docker.yaml` | 7-13 | Gateway path and upstream target for `/api/v1/auth` |
| P1 (important) | `pkg/middleware/cors.go` | 19-75 | CORS defaults show browser calls are allowed in dev |
| P1 (important) | `services/auth/internal/platform/config/config.go` | 102-160 | Existing env-driven config style to mirror in frontend |
| P2 (reference) | `services/auth/internal/bootstrap/app.go` | 33-135 | Repo favors explicit bootstrap wiring over hidden magic |
| P2 (reference) | `services/auth/internal/delivery/http/handler/auth_handler_test.go` | 406-555 | End-to-end flow order and expected statuses/messages |
| P2 (reference) | `.gitignore` | 31-43, 138-142 | Env/build outputs already ignored at repo level |

## External Documentation

| Topic | Source | Key Takeaway |
|---|---|---|
| Next.js installation | `https://nextjs.org/docs/app/getting-started/installation` | `create-next-app` defaults now include TypeScript, ESLint, App Router, Tailwind, Turbopack; Node 20.9+ required |
| Next.js project structure | `https://nextjs.org/docs/app/getting-started/project-structure` | Use `app/` with route groups and private folders for route-safe organization |
| Next.js env variables | `https://nextjs.org/docs/app/guides/environment-variables` | Server-only envs stay private; browser envs must use `NEXT_PUBLIC_` prefix |

---

## Unified Discovery Table

| Category | File:Lines | Pattern | Key Snippet |
|---|---|---|---|
| Similar implementation | `services/auth/internal/bootstrap/app.go:33-135` | Explicit bootstrap sequence by dependency layer | `config -> logger -> database -> jwt -> repository -> usecase -> handler -> router -> server` |
| Naming | `services/auth/internal/usecase/dto/auth_dto.go:7-50` | Explicit request/response type names by action | `type RegisterRequest`, `type ResetPasswordRequest` |
| Error handling | `pkg/response/response.go:47-81` | Central envelope with code/message and hidden internals | `c.JSON(appErr.HTTPStatus(), Response{Code: ..., Message: ...})` |
| Logging | `services/auth/internal/usecase/auth_usecase.go:156-173` | Structured `slog.*Context` with stable keys | `slog.InfoContext(ctx, "user logged in", "user_id", user.ID)` |
| Types/contracts | `services/auth/internal/usecase/dto/auth_dto.go:81-144` | DTOs encode auth-specific states, not just happy-path payloads | `RequiresEmailVerification`, `Requires2FA`, `TempToken` |
| Test pattern | `services/auth/internal/delivery/http/handler/auth_handler_test.go:409-439` | End-to-end flow assertions against real HTTP router | `postJSON(.../register)`, `postJSON(.../verify-email)`, `postJSON(.../login)` |
| Configuration | `services/auth/internal/platform/config/config.go:102-160` | Load from env with sane fallbacks | `getEnv(...)`, `getDurationEnv(...)` |
| Dependencies | `docker-compose.yml:26-80` | Local stack runs auth on `8081`, gateway on `8080` | `auth-service`, `gateway` |

---

## Patterns to Mirror

Code patterns discovered in the codebase. Follow these exactly.

### NAMING_CONVENTION
// SOURCE: `services/auth/internal/usecase/dto/auth_dto.go:7-44`
```go
type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
	FullName string `json:"full_name"`
}

type ResetPasswordRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required"`
}
```
Mirror: use explicit, action-based names in frontend too, e.g. `RegisterFormData`, `ResetPasswordPayload`, `AuthApiResponse` instead of vague `Data` or `FormValues`.

### ERROR_HANDLING
// SOURCE: `pkg/response/response.go:47-64`
```go
func Error(c *gin.Context, err error) {
	var appErr *apperror.AppError
	if errors.As(err, &appErr) {
		c.JSON(appErr.HTTPStatus(), Response{
			Code:    string(appErr.Code),
			Message: appErr.Message,
		})
		return
	}

	c.JSON(http.StatusInternalServerError, Response{
		Code:    string(apperror.CodeInternalError),
		Message: "an unexpected error occurred",
	})
}
```
Mirror: frontend client must treat `code` and `message` as canonical error contract and never depend on raw backend error bodies.

### LOGGING_PATTERN
// SOURCE: `services/auth/internal/usecase/auth_usecase.go:156-173`
```go
slog.InfoContext(ctx, "user requires 2FA login", "user_id", user.ID)
slog.WarnContext(ctx, "failed to revoke old tokens on login", "error", err, "user_id", user.ID)
slog.InfoContext(ctx, "user logged in", "user_id", user.ID)
```
Mirror: when adding browser/server logs in Next.js, keep structured event-style names (`auth_api_request_failed`, `auth_bootstrap_invalid_env`) and attach stable fields instead of free-form prose.

### REPOSITORY_PATTERN
// SOURCE: `services/auth/internal/domain/repository.go:35-53`
```go
type TokenRepository interface {
	Create(ctx context.Context, token *RefreshToken) error
	GetByTokenHash(ctx context.Context, tokenHash string) (*RefreshToken, error)
	ListByUserID(ctx context.Context, userID uuid.UUID) ([]*RefreshToken, error)
	RevokeByUserID(ctx context.Context, userID uuid.UUID) error
	RevokeByIDAndUserID(ctx context.Context, id, userID uuid.UUID) error
	DeleteExpired(ctx context.Context) error
}
```
Mirror: keep frontend API access behind a small explicit module (`lib/api/auth.ts` or similar), with one function per backend capability instead of scattering `fetch` calls through pages.

### SERVICE_PATTERN
// SOURCE: `services/auth/internal/bootstrap/app.go:33-35`
```go
// config → logger → database → jwt → repository → usecase → handler → router → server
```
Mirror: build frontend bootstrap in layers: env/config -> HTTP client -> auth helpers/storage -> layouts/routes. Do not mix route UI with transport setup in one file.

### TEST_STRUCTURE
// SOURCE: `services/auth/internal/delivery/http/handler/auth_handler_test.go:409-439`
```go
registerResp, err := postJSON(ts.URL+"/api/v1/auth/register", dto.RegisterRequest{
	Email:    "handler@test.com",
	Password: "password123",
	FullName: "Handler User",
})
require.NoError(t, err)
assert.Equal(t, http.StatusCreated, registerResp.StatusCode)

loginResp, err := postJSON(ts.URL+"/api/v1/auth/login", dto.LoginRequest{
	Email:    "handler@test.com",
	Password: "password123",
})
require.NoError(t, err)
assert.Equal(t, http.StatusForbidden, loginResp.StatusCode)
```
Mirror: frontend smoke tests or manual validation should follow full user journeys, not isolated component-only checks.

---

## Files to Change

| File | Action | Justification |
|---|---|---|
| `frontend/package.json` | CREATE | Own dependencies and scripts for Next.js app |
| `frontend/package-lock.json` | CREATE | NPM lockfile, consistent with existing JS subprojects using npm lockfiles |
| `frontend/tsconfig.json` | CREATE | TypeScript config and import alias support |
| `frontend/next.config.ts` or `frontend/next.config.mjs` | CREATE | Explicit Next.js config at app root |
| `frontend/eslint.config.mjs` | CREATE | Linting entrypoint for new app |
| `frontend/.gitignore` | CREATE | App-specific ignores generated by create-next-app |
| `frontend/.env.example` | CREATE | Document required env vars for auth base URL and app metadata |
| `frontend/app/layout.tsx` | CREATE | Required App Router root layout |
| `frontend/app/page.tsx` | CREATE | Bootstrap landing page / connectivity proof |
| `frontend/app/(auth)/layout.tsx` | CREATE | Route group shell for auth pages |
| `frontend/app/(app)/layout.tsx` | CREATE | Route group shell for future protected area |
| `frontend/app/(auth)/login/page.tsx` | CREATE | Placeholder route proving routing foundation |
| `frontend/lib/env.ts` | CREATE | Centralized env access and runtime validation |
| `frontend/lib/api/types.ts` | CREATE | Shared response envelope and auth DTO mirrors |
| `frontend/lib/api/client.ts` | CREATE | Shared fetch wrapper for backend envelope parsing |
| `frontend/lib/auth/storage.ts` | CREATE | Token storage abstraction chosen for MVP bootstrap |
| `.claude/PRPs/prds/auth-service-nextjs-frontend.prd.md` | UPDATE | Mark phase 1 as `in-progress` and attach this plan path |

## NOT Building

- Full public auth forms and end-to-end UI for register/login/reset in this phase.
- 2FA setup/verify screens in this phase.
- SSR auth session hydration finalized; only the bootstrap seam for future auth state work.
- Backoffice/admin screens such as lockout policy UI.
- Backend API changes, proxy rewrites, or gateway changes unless strictly needed for local access.

---

## Step-by-Step Tasks

### Task 1: Scaffold Next.js App in `frontend/`
- **ACTION**: Create the new app with App Router, TypeScript, ESLint, and default Next.js structure.
- **IMPLEMENT**: Use `create-next-app` to generate `frontend/` with TypeScript, App Router, and ESLint. Prefer npm because existing JS subprojects in the repo already commit `package-lock.json`.
- **MIRROR**: `SERVICE_PATTERN` for explicit bootstrap layers; Next.js installation docs for required root files.
- **IMPORTS**: Generated by scaffold; no manual imports required at this step.
- **GOTCHA**: Do not create the app inside an existing service folder. It must live at repo root as `frontend/` to match the PRD.
- **VALIDATE**: `frontend/` exists with `package.json`, `app/layout.tsx`, `app/page.tsx`, `tsconfig.json`, `eslint.config.mjs`, and lockfile.

### Task 2: Establish App Structure for Public and Protected Areas
- **ACTION**: Add route groups and layout placeholders that match planned auth surface.
- **IMPLEMENT**: Keep `app/layout.tsx` minimal and create route groups such as `app/(auth)` and `app/(app)` with lightweight layouts/placeholders. This should prepare for later phases without implementing full flows.
- **MIRROR**: Next.js project-structure docs and repo preference for clear grouping (`router.go` separates public/protected auth endpoints).
- **IMPORTS**: `ReactNode` typing in layouts; no custom state imports yet unless truly needed.
- **GOTCHA**: Route groups must not alter URLs; avoid overbuilding nested layout logic during bootstrap.
- **VALIDATE**: App routes compile, placeholder pages render, and filesystem structure makes intended public/protected separation obvious.

### Task 3: Create Centralized Environment and API Configuration
- **ACTION**: Add a single env module and document required variables.
- **IMPLEMENT**: Create `frontend/lib/env.ts` and `.env.example`. Include at minimum a browser-safe auth API base URL like `NEXT_PUBLIC_AUTH_API_BASE_URL`, and optionally app name/title variables if used by layout metadata.
- **MIRROR**: `services/auth/internal/platform/config/config.go:102-160` and Next.js env docs.
- **IMPORTS**: `process.env`; optional lightweight validation helpers if added locally.
- **GOTCHA**: Only expose values to browser code when prefixed with `NEXT_PUBLIC_`. Keep secrets out of client bundle.
- **VALIDATE**: App starts with valid envs; invalid or missing required envs fail clearly in one place rather than at random fetch call sites.

### Task 4: Implement Shared HTTP Client and Response Envelope Parsing
- **ACTION**: Create a reusable fetch layer that understands backend success/error responses.
- **IMPLEMENT**: Add `frontend/lib/api/types.ts` with TS mirrors of `Response`, selected auth DTOs, and a typed error shape. Add `frontend/lib/api/client.ts` that performs `fetch`, decodes `{ code, message, data }`, and throws a normalized frontend error when `response.ok` is false or the envelope is malformed.
- **MIRROR**: `ERROR_HANDLING`, `NAMING_CONVENTION`, and backend response envelope in `pkg/response/response.go`.
- **IMPORTS**: `process.env` from env module; native `fetch`; any local `ApiError` type.
- **GOTCHA**: Backend uses `204 No Content` for logout/session delete flows. Client wrapper must handle empty bodies without trying to JSON decode them.
- **VALIDATE**: One smoke call, ideally to `/health` or one auth endpoint, succeeds and returns typed data; malformed responses surface a stable error object.

### Task 5: Add Auth Storage Seam Without Finalizing Full Auth Architecture
- **ACTION**: Introduce a narrow token-storage abstraction needed for later phases.
- **IMPLEMENT**: Create `frontend/lib/auth/storage.ts` with tiny get/set/clear helpers for access and refresh tokens, but keep storage strategy deliberately minimal and replaceable. Document that the phase does not finalize SSR-vs-client persistence.
- **MIRROR**: `REPOSITORY_PATTERN` and PRD open questions about token handling.
- **IMPORTS**: Browser storage API if using client-only implementation; otherwise no-op server guards.
- **GOTCHA**: Do not couple route layouts directly to `localStorage` access in server components. Keep browser-only logic isolated.
- **VALIDATE**: Storage helper can be imported without crashing server-rendered code paths.

### Task 6: Add Bootstrap Smoke UX and Connectivity Proof
- **ACTION**: Make the root page prove the app is wired and explain next routes.
- **IMPLEMENT**: Use `frontend/app/page.tsx` to show app identity, configured auth base URL (if safe), and links or placeholders to future auth routes. If feasible, add a simple connectivity check on the server or client to a non-sensitive endpoint.
- **MIRROR**: Test journey mindset from `TEST_STRUCTURE`; use backend ports and gateway routes from `docker-compose.yml` and `routes.docker.yaml`.
- **IMPORTS**: `Link` from `next/link`, env module, API client if checking connectivity.
- **GOTCHA**: Avoid exposing sensitive internals. If showing URLs, only show the public base URL already intended for browser use.
- **VALIDATE**: Opening `/` makes it obvious the bootstrap succeeded and future work has a stable base.

### Task 7: Align Repo Artifacts and PRD State
- **ACTION**: Update the PRD phase row after writing the plan.
- **IMPLEMENT**: Change phase 1 status from `pending` to `in-progress` and set PRP Plan column to `.claude/PRPs/plans/auth-service-nextjs-frontend-bootstrap.plan.md`.
- **MIRROR**: Existing PRP workflow contract from the user instruction.
- **IMPORTS**: N/A.
- **GOTCHA**: Only update the selected phase. Do not touch later phases.
- **VALIDATE**: PRD shows phase 1 as active and references this plan path exactly.

---

## Testing Strategy

### Unit Tests

| Test | Input | Expected Output | Edge Case? |
|---|---|---|---|
| Env module loads required public base URL | Valid `NEXT_PUBLIC_AUTH_API_BASE_URL` | Returns normalized config object | No |
| Env module rejects missing required URL | Missing env | Throws clear bootstrap/config error | Yes |
| API client parses success envelope | `{ code: "SUCCESS", message: "success", data: {...} }` | Returns typed `data` | No |
| API client parses error envelope | Non-2xx with `{ code, message }` | Throws normalized `ApiError` | Yes |
| API client handles 204 response | Empty body with `204` | Returns `undefined` / void safely | Yes |
| Auth storage avoids SSR crash | Imported during server render | No `window` access on server | Yes |

### Edge Cases Checklist
- [x] Empty input
- [ ] Maximum size input
- [x] Invalid types
- [ ] Concurrent access
- [x] Network failure (if applicable)
- [ ] Permission denied

Notes:
- For this phase, concurrency and permission cases are not primary bootstrap concerns.
- If no JS test runner is added in phase 1, cover the table above with manual smoke checks and postpone automated unit tests to the first feature phase that introduces richer client logic.

---

## Validation Commands

### Static Analysis
```bash
cd frontend && npm run lint
```
EXPECT: Zero lint errors

### Type Check
```bash
cd frontend && npm exec tsc --noEmit
```
EXPECT: Zero type errors

### Unit Tests
```bash
cd frontend && npm test
```
EXPECT: If a test runner is configured in this phase, tests pass. If no runner is added, document `N/A for bootstrap phase` in the implementation notes instead of inventing tests.

### Full Test Suite
```bash
go test ./...
```
EXPECT: Existing backend tests still pass; frontend bootstrap should not break repo state

### Browser Validation
```bash
cd frontend && npm run dev
```
EXPECT: Next.js dev server starts successfully and the home page renders at `http://localhost:3000`

### Manual Validation
- [ ] Start backend locally via `docker compose up auth-service gateway postgres`
- [ ] Start frontend via `cd frontend && npm run dev`
- [ ] Open `http://localhost:3000`
- [ ] Confirm root page renders with app shell, no crash
- [ ] Confirm placeholder routes under auth/protected groups render
- [ ] Confirm API client can reach configured auth base URL or fail with clear user-facing/bootstrap message
- [ ] Confirm browser console/server logs do not show secret env values

---

## Acceptance Criteria
- [ ] All tasks completed
- [ ] All validation commands pass
- [ ] API client handles backend response envelope correctly
- [ ] No type errors
- [ ] No lint errors
- [ ] Next.js app exists in `frontend/` and runs locally
- [ ] PRD phase 1 is marked `in-progress` with this plan path attached

## Completion Checklist
- [ ] Code follows discovered patterns
- [ ] Error handling matches codebase style
- [ ] Logging follows codebase conventions
- [ ] Tests follow test patterns or are explicitly deferred with rationale
- [ ] No hardcoded values
- [ ] Documentation updated (if needed)
- [ ] No unnecessary scope additions
- [ ] Self-contained — no questions needed during implementation

## Risks
| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Storage strategy chosen in bootstrap conflicts with later SSR decision | High | Medium | Keep storage behind narrow abstraction and avoid hard-coupling layouts to it |
| Frontend uses auth-service direct URL while later deployment expects gateway | Medium | Medium | Centralize base URL in env and never hardcode endpoints |
| Overbuilding phase 1 into a full auth feature set | High | Medium | Limit scope to app shell, env, client, and route foundation only |
| Client wrapper misreads backend envelope or 204 responses | Medium | High | Mirror `pkg/response/response.go` exactly and test success/error/empty cases |

## Notes
- No similar frontend implementation exists in this repo. The closest internal analog is the backend bootstrap pattern plus the auth handler tests that define journey order and payload semantics.
- Prefer App Router because the current Next.js docs treat it as the default path, and the PRD already anticipates route groups for public/protected sections.
- Use npm unless the user later standardizes on another package manager. This is the least surprising option because the repo already contains committed `package-lock.json` files in JS subprojects.
- Initial auth base URL can point either to gateway (`http://localhost:8080`) or auth service (`http://localhost:8081`). Default to gateway in `.env.example` if the intent is to exercise the real external surface; keep it configurable because local direct-to-auth may still be useful during debugging.
- `GET /health` is a safe bootstrap connectivity probe. `GET /api/v1/auth/validate` should not be assumed as the default auth-state bootstrap mechanism until later phases decide session strategy.

*Generated: 2026-04-15 14:52:21 +07:00*
