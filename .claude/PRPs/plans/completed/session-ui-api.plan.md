# Plan: Session UI API

## Summary
Add REST endpoints allowing clients to retrieve all active sessions (Refresh Tokens) for the currently authenticated user, as well as an endpoint to revoke a specific session. This implements Phase 4 of the Core Security Upgrade.

## User Story
As an authenticated user, I want to see a list of my active login sessions and be able to revoke them individually from my device, so that I can remotely log out of recognized devices in case of loss or security breach.

## Problem → Solution
Users cannot view or manage other device logins. → Users can list all non-revoked RefreshTokens and soft-delete/revoke them via `/sessions` APIs.

## Metadata
- **Complexity**: Medium
- **Source PRD**: `.claude/PRPs/prds/core-security-upgrade.prd.md`
- **PRD Phase**: Phase 4 - Session UI API
- **Estimated Files**: 5

---

## UX Design

### Before
N/A — internal change or feature absence.

### After
Client applications can query GET `/sessions` to render a list of devices (built on IP, UserAgent, DeviceID captured in Phase 2) and send DELETE `/sessions/:id` for remote logout.

### Interaction Changes
| Touchpoint | Before | After | Notes |
|---|---|---|---|
| `GET /api/v1/auth/sessions` | None | Returns list of RefreshToken data | Array of objects for that user |
| `DELETE /api/v1/auth/sessions/:id` | None | Revokes the specified session | Soft-delete via DB `revoked=true` |

---

## Mandatory Reading

| Priority | File | Lines | Why |
|---|---|---|---|
| P0 | `services/auth/internal/domain/repository.go` | 35-51 | Learn the current limits of `TokenRepository` to add `ListByUserID`. |
| P2 | `services/auth/internal/delivery/http/handler/auth_handler.go` | all | Learn standard request parsing and API responses `response.OK` etc. |

## External Documentation
- No external research needed — feature uses established internal patterns.

---

## Patterns to Mirror

### NAMING_CONVENTION
// SOURCE: `services/auth/internal/domain/repository.go:42`
```go
	GetByTokenHash(ctx context.Context, tokenHash string) (*RefreshToken, error)
```
Follow this with: `ListByUserID(ctx context.Context, userID uuid.UUID) ([]*RefreshToken, error)`

### ERROR_HANDLING
// SOURCE: `pkg/response/response.go:43`
```go
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, data)
```

---

## Files to Change

| File | Action | Justification |
|---|---|---|
| `services/auth/internal/domain/repository.go` | UPDATE | Add `ListByUserID` to `TokenRepository` interface |
| `services/auth/internal/domain/service.go` | UPDATE | Add `GetSessions` and `DeleteSession` to `AuthService` interface |
| `services/auth/internal/repository/postgres/token_repository.go` | UPDATE | Implement `ListByUserID` in GORM |
| `services/auth/internal/usecase/auth_usecase.go` | UPDATE | Implement `GetSessions` and `DeleteSession` (with ownership check) |
| `services/auth/internal/delivery/http/handler/auth_handler.go` | UPDATE | Add `GetSessions` and `DeleteSession` router handlers |
| `services/auth/internal/delivery/http/router.go` | UPDATE | Mount the GET and DELETE routes to the protected group |

## NOT Building
- UI frontend for sessions list. Only adding backend API capabilities.
- Bulk logout from all devices (handled individually via loops on client, or `RevokeByUserID` triggers full logout, which already exists behind `/logout`). This only adds GET and DELETE for specific sessions.

---

## Step-by-Step Tasks

### Task 1: Update Domain Interfaces
- **ACTION**: Add methods to repositories and services
- **IMPLEMENT**: Add `ListByUserID(ctx context.Context, userID uuid.UUID) ([]*RefreshToken, error)` to `TokenRepository`. Add `GetSessions(ctx context.Context, userID uuid.UUID) ([]*domain.RefreshToken, error)` and `DeleteSession(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID) error` to `AuthService`.
- **MIRROR**: Follow standard Go interface design as in `domain/repository.go`
- **IMPORTS**: N/A
- **GOTCHA**: N/A
- **VALIDATE**: Code compiles successfully

### Task 2: Implement Postgres Token Repository
- **ACTION**: Write GORM query to fetch sessions
- **IMPLEMENT**: In `token_repository.go`, fetch non-revoked sessions via `Where("user_id = ? AND revoked = false", userID).Order("created_at DESC")`. Map result to Domain struct using `mapper.TokenToDomain`.
- **MIRROR**: `GetByTokenHash` logic inside the same file.
- **IMPORTS**: `github.com/base-go/base/services/auth/internal/repository/postgres/mapper`
- **GOTCHA**: Must verify if `mapper.TokenToDomain` supports batch converting. If not, map inside a `for` loop.
- **VALIDATE**: `go build`

### Task 3: Implement Auth Usecase
- **ACTION**: Tie Database queries to Business Logic
- **IMPLEMENT**: Write `GetSessions` calling `uc.tokenRepo.ListByUserID`. Write `DeleteSession` calling `uc.tokenRepo.RevokeByID`, BUT ensuring ownership first? Wait! `RevokeByID` doesn't verify ownership. We should enforce ownership: call `uc.tokenRepo.RevokeByID` only if we can verify the session belongs to `userID`. Since we don't have `GetByID` for RefreshToken, it's safer to implement `RevokeByUserAndID(userID, sessionID)` or verify via the list first.
- **MIRROR**: Error wrapping `apperror.InternalError`
- **GOTCHA**: Ensure users can only delete their OWN sessions. The easiest way is to modify `RevokeByID` in `TokenRepository` to `RevokeByIDAndUserID(ctx context.Context, id, userID uuid.UUID) error`. Update Task 1 and Task 2 to match this stricter signature.
- **VALIDATE**: Logic protects unauthorized session drops.

### Task 4: Implement Handlers & Routing
- **ACTION**: Expose HTTP endpoints
- **IMPLEMENT**: In `auth_handler.go`, fetch `userID` from Context (`h.getUserID`), invoke `GetSessions` returning DTO/Response, and `DeleteSession` via param `:id` parsed into UUID. Mount `/sessions` and `/sessions/:id` to `protected` route in `router.go`.
- **MIRROR**: `GetProfile` logic in `auth_handler.go`
- **IMPORTS**: `github.com/google/uuid`
- **VALIDATE**: API answers correctly via route mapping

### Task 5: Unit Tests
- **ACTION**: Add test cases for UseCase and Handler
- **IMPLEMENT**: Create mock objects and assertions in `_test.go`
- **VALIDATE**: `go test` passes

---

## Testing Strategy

### Unit Tests
| Test | Input | Expected Output | Edge Case? |
|---|---|---|---|
| Test Handler GET | Request to `/sessions` | `200 OK` Array | empty DB returns `[]` |
| Test Handler DELETE | Request to `/sessions/:id` | `204 NoContent` | Invalid UUID format -> 400 Bad Request |
| Test Usecase DELETE | Non-owned session drop attempt | Error | Yes |

---

## Validation Commands

### Static Analysis
```bash
go vet ./...
```
EXPECT: Zero warnings

### Unit Tests
```bash
go test ./... -v
```
EXPECT: All tests pass

### Build
```bash
go build -o auth-server.exe ./services/auth/cmd/auth
```
EXPECT: Build succeeds

---

## Acceptance Criteria
- [ ] Domain Interfaces updated
- [ ] GORM queries added with security ownership scope.
- [ ] Usecase connects layers successfully.
- [ ] API endpoints available on Gin Router.
- [ ] Tests and build pass.

## Completion Checklist
- [ ] Code follows discovered patterns
- [ ] Error handling matches codebase style
- [ ] Logging follows codebase conventions
- [ ] Tests follow test patterns

## Risks
| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| IDOR Vulnerability | M | H | Ensure the DELETE route requires `userID` boundary in DB lookup. |

## Notes
To prevent IDOR, we will replace the planned `RevokeByID` modification and simply add `RevokeSession(ctx, sessionID, userID)` or enforce it directly in the GORM condition `Where("id = ? AND user_id = ?", id, userID).Update("revoked", true)` in the DB layer.
