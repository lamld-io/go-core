# Plan: TOTP 2FA Core

## Summary
Add core schema and library support for TOTP (Time-Based One-Time Password) 2FA. This covers Phase 5 of the Core Security Upgrade PRD. We will install the `github.com/pquerna/otp` library, update the `User` schema (`TOTPSecret`, `Is2FAEnabled`), and construct APIs for authenticated users to setup and verify their 2FA settings.

## User Story
As an authenticated user, I want to set up an Authenticator app, so that my account is protected with a second factor of authentication.

## Problem → Solution
Accounts can be hacked if a password is stolen. → Users can optionally enable a TOTP app, tying their login to a securely generated 2FA code.

## Metadata
- **Complexity**: Medium
- **Source PRD**: `.claude/PRPs/prds/core-security-upgrade.prd.md`
- **PRD Phase**: Phase 5 - TOTP 2FA Core
- **Estimated Files**: 9

---

## UX Design

### Before
N/A - Internal APIs with no exposed 2FA capabilities.

### After
N/A - UI apps will render the Setup QR code using `SecretURL` returned by the new `/api/v1/auth/2fa/setup` API.

### Interaction Changes
| Touchpoint | Before | After | Notes |
|---|---|---|---|
| `POST /api/v1/auth/2fa/setup` | None | Returns Setup2FAResponse containing `secret` and `secret_url` | Protected route |
| `POST /api/v1/auth/2fa/verify` | None | Activates 2FA and returns success | Protected route, expects `code` (string) |

---

## Mandatory Reading

| Priority | File | Lines | Why |
|---|---|---|---|
| P0 | `services/auth/internal/domain/user.go` | 10-22 | To inject 2FA properties cleanly. |
| P1 | `services/auth/internal/repository/postgres/model/user_model.go` | 12-25 | To add GORM schema fields. |

## External Documentation

| Topic | Source | Key Takeaway |
|---|---|---|
| pquerna/otp | https://github.com/pquerna/otp | `totp.Generate(totp.GenerateOpts{Issuer: "lamld-io", AccountName: "email"})` generates secret. Use `totp.Validate(code, secret)` for validation. |

---

## Patterns to Mirror

### NAMING_CONVENTION
// SOURCE: `services/auth/internal/delivery/http/handler/auth_handler.go:68-82`
Same routing structure handling `Verify2FARequest` via `c.ShouldBindJSON`.

### ERROR_HANDLING
// SOURCE: `pkg/apperror/errors.go`
Use or add standard application errors. We can use `apperror.ValidationError("invalid OTP code")`.

---

## Files to Change

| File | Action | Justification |
|---|---|---|
| `go.mod/go.sum` | UPDATE | Install `github.com/pquerna/otp` |
| `services/auth/internal/domain/user.go` | UPDATE | Add `TOTPSecret (*string)` and `Is2FAEnabled (bool)` |
| `services/auth/internal/repository/postgres/model/user_model.go` | UPDATE | Add `TOTPSecret` and `Is2FAEnabled` GORM tags |
| `services/auth/internal/repository/postgres/mapper/user_mapper.go` | UPDATE | Map the two new fields back and forth |
| `services/auth/internal/domain/service.go` | UPDATE | Add `Setup2FA` and `Verify2FA` to `AuthService` interface |
| `services/auth/internal/usecase/dto/auth_dto.go` | UPDATE | Add Setup2FAResponse, Verify2FARequest |
| `services/auth/internal/usecase/auth_usecase.go` | UPDATE | Implement `Setup2FA` and `Verify2FA` logic |
| `services/auth/internal/delivery/http/handler/auth_handler.go` | UPDATE | Create `Setup2FA` and `Verify2FA` handlers |
| `services/auth/internal/delivery/http/router.go` | UPDATE | Mount the new routes under `/auth/2fa` or protected Group |

## NOT Building
- Login flow modifications (Phase 6 scope)
- Backup codes generation
- SMS / Email OTP fallbacks

---

## Step-by-Step Tasks

### Task 1: Install Dependencies
- **ACTION**: Install OTP library
- **IMPLEMENT**: `go get github.com/pquerna/otp`
- **VALIDATE**: `go mod tidy` succeeds.

### Task 2: Domain and Persistence Models
- **ACTION**: Add 2FA fields to `User`
- **IMPLEMENT**: Add `TOTPSecret *string` and `Is2FAEnabled bool` to `domain.User` and `model.UserModel`. Set default `false` for boolean in GORM tag. Update `mapper/user_mapper.go` appropriately.
- **GOTCHA**: Ensure pointers are mapped cleanly.
- **VALIDATE**: `go build` passes.

### Task 3: API Contracts
- **ACTION**: Define Service Interface and DTOs
- **IMPLEMENT**: 
  - Add `Setup2FAResponse` (Secret, SecretURL) in `auth_dto.go`
  - Add `Verify2FARequest` (Code) in `auth_dto.go`
  - Add `Setup2FA(ctx, userID) (*dto.Setup2FAResponse, error)` to `AuthService`
  - Add `Verify2FASetup(ctx, userID, code) error` to `AuthService`
- **VALIDATE**: No type errors.

### Task 4: Usecase Implementation
- **ACTION**: Write OTP logic
- **IMPLEMENT**: In `auth_usecase.go`.
  - `Setup2FA`: Call `userRepo.GetByID`. If `Is2FAEnabled` is true, return error "2FA already enabled". Run `totp.Generate` with user Email and Issuer "lamld-io". Save secret to `TOTPSecret` using `userRepo.Update`. Return DTO.
  - `Verify2FASetup`: Call `userRepo.GetByID`. Check if `TOTPSecret` is nil. Run `totp.Validate()`. If invalid, return error. If valid, set `Is2FAEnabled = true` and `userRepo.Update`.
- **IMPORTS**: `github.com/pquerna/otp/totp`
- **VALIDATE**: `go test` integration.

### Task 5: Delivery
- **ACTION**: Expose HTTP endpoints
- **IMPLEMENT**: Add `Setup2FA` and `Verify2FASetup` Handlers. Map responses. Mount to `protected` route group: `protected.POST("/2fa/setup", ...)` and `protected.POST("/2fa/verify", ...)`.
- **VALIDATE**: `go build` and `go vet` passes.

### Task 6: Testing
- **ACTION**: Update mocks and add tests
- **IMPLEMENT**: Update mock `AuthService` in handler tests. Add unit tests for 2FA features if necessary or rely on robust mock validation in existing specs.

---

## Testing Strategy

### Unit Tests
| Test | Input | Expected Output | Edge Case? |
|---|---|---|---|
| Try setup again | Setup2FA when Is2FAEnabled=true | `400 Bad Request` | Yes |
| Verify bad code | Verify2FA with "000000" | Validation error | |
| Verify missing secret | Verify2FA when TOTPSecret=nil | Initialization error | |

---

## Validation Commands

### Static Analysis
```bash
go vet ./...
```
EXPECT: Zero type errors

### Build Validation
```bash
go build ./...
```
EXPECT: Builds OK.

---

## Acceptance Criteria
- [ ] OTP Library installed
- [ ] Schema handles `is_2fa_enabled` and `totp_secret`
- [ ] `POST /2fa/setup` generates secret properly
- [ ] `POST /2fa/verify` confirms the code and enables 2FA flag 
- [ ] Code builds without errors

## Completion Checklist
- [ ] Code follows discovered patterns
- [ ] Error handling matches codebase style

## Risks
| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Unsynced clock times | Medium | Medium | Use `totp.ValidateCustom` if default 30s skew is too tight in production, though `totp.Validate` is generally sufficient. |
