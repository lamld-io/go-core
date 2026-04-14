# Plan: 2FA Login Flow

## Summary
Refactor the `Login` flow to support a two-step authentication process for accounts with TOTP 2FA enabled. If the password matches and 2FA is active, an intermediate `Temp2FAToken` (JWT) is issued instead of a normal TokenPair. The user must then submit this temporary token alongside their Authenticator App OTP to `/login/2fa` to receive the fully authorized TokenPair.

## User Story
As a user with 2FA enabled, I want to authenticate in two steps with my password and a TOTP code, so that my account is protected even if my password is leaked.

## Problem → Solution
Current `/login` immediately issues `AccessToken` & `RefreshToken` → New flow halts at a `TempToken` if `is_2fa_enabled` is true, delegating final token issuance to a dedicated `/login/2fa` endpoint.

## Metadata
- **Complexity**: Medium
- **Source PRD**: `.claude/PRPs/prds/core-security-upgrade.prd.md`
- **PRD Phase**: 6 (2FA Login Flow)
- **Estimated Files**: 6

---

## Technical Approach

### `pkg/jwt`

- Modify `pkg/jwt/claims.go` to introduce a new `TokenType`: `const Temp2FAToken TokenType = "temp_2fa"`.
- Modify `pkg/jwt/jwt.go` to add the `GenerateTemp2FAToken(userID, email string) (string, error)` logic, explicitly enforcing a 2-minute TTL.

### Service Layer

#### `service.go`
- Create `LoginResult` in the Domain layer to encapsulate conditional flows from the Usecase:
```go
type LoginResult struct {
	User        *User
	TokenPair   *TokenPair
	Requires2FA bool
	TempToken   string
}
```
- Refactor `AuthService.Login` replacing `(*User, *TokenPair, error)` with `(*LoginResult, error)`.
- Explicitly add `Verify2FALogin(ctx context.Context, tempToken, code string, meta ClientMetadata) (*LoginResult, error)` to the `AuthService` interface.

#### `auth_usecase.go`
- Alter `Login` to check `user.Is2FAEnabled` immediately after a successful password verification. If true, generate `Temp2FAToken` via `jwtManager` and return it.
- In `Verify2FALogin`, extract the `UserID` from the decoded TempToken, confirm its `TokenType` is `Temp2FAToken`, retrieve the specific user from the repository, validate `user.TOTPSecret` against the provided code using `pquerna/otp`, and subsequently trigger `uc.generateTokenPair` just as a standard successful login would. Incorporate token revocation for old sessions.

### API Layer

#### `auth_dto.go`
- Add `LoginResponse` DTO structure:
```go
type LoginResponse struct {
	Requires2FA bool           `json:"requires_2fa"`
	TempToken   string         `json:"temp_token,omitempty"`
	Message     string         `json:"message,omitempty"`
	User        *UserResponse  `json:"user,omitempty"`
	Token       *TokenResponse `json:"token,omitempty"`
}
```
- Add `Login2FARequest` DTO structure:
```go
type Login2FARequest struct {
    TempToken string `json:"temp_token" binding:"required"`
    Code      string `json:"code" binding:"required"`
}
```

#### `presenter`/`error_presenter.go`
- Append `ToLoginResponse(result *domain.LoginResult, expiresInSec int64) dto.LoginResponse` mappings to safely map outputs conditionally without breaching security abstractions.

#### `auth_handler.go`
- Overhaul `authHandler.Login` to handle the `domain.LoginResult` output via `presenter.ToLoginResponse`.
- Add `authHandler.Login2FA` method. Validates `Login2FARequest` inputs and handles `Verify2FALogin`. Emits `LoginResponse` upon successful evaluation.

#### `router.go`
- Route `POST /login/2fa` inside the `public` group within `/api/v1/auth`. Attach the `IPRateLimiter` middleware.

---

## Step-by-Step Tasks

### Task 1: JWT & Token Logic
- **ACTION**: Add `Temp2FAToken` logic.
- **IMPLEMENT**: Add `Temp2FAToken` to `claims.go` and `GenerateTemp2FAToken` natively inside `pkg/jwt/jwt.go`.
- **VALIDATE**: `go build` locally.

### Task 2: Service Layer Refactoring
- **ACTION**: Modify `AuthService` interface and domain model.
- **IMPLEMENT**: Add `LoginResult`, update `Login` signature, and structure `Verify2FALogin` interfaces.
- **VALIDATE**: Modify dependent functions to map `LoginResult` correctly in `auth_usecase.go`.

### Task 3: Usecase Core Adjustments
- **ACTION**: Integrate 2FA conditionals parsing into `Login`.
- **IMPLEMENT**: Halt TokenPair generation when `user.Is2FAEnabled == true` and execute `GenerateTemp2FAToken`. Add `Verify2FALogin` logic entirely.
- **VALIDATE**: Write corresponding integrations if possible; ensure strict `go vet` passes against the interface updates.

### Task 4: API Layer DTOs and Payload Adjustments
- **ACTION**: Restructure `LoginResponse`.
- **IMPLEMENT**: Update DTO structural mapping arrays, enforce `ToLoginResponse` payload modifications. Ensure previous consumers aren't functionally locked during generic updates.
- **VALIDATE**: Compile correctly referencing structural changes.

### Task 5: HTTP Implementation & Routing
- **ACTION**: Add `Login2FA` Handler.
- **IMPLEMENT**: Extract dependencies, parse JSON correctly onto the newly created endpoint structure mapping via `authHandler`. Expose via Gin bindings (`/login/2fa`).

### Task 6: Testing & Compiling
- **ACTION**: Validate application suite.
- **IMPLEMENT**: Overhaul existing `mockAuthService` in `auth_handler_test.go` and check structural test integrations matching `Login` returns safely.
- **VALIDATE**: `go test ./...`.

---

## Validation Commands
```bash
go vet ./...
go test ./...
go build ./...
```
EXPECT: Zero compile issues across mock suites following signature transformations.

---

## Acceptance Criteria
- [ ] Users with 2FA enabled do NOT receive an Access Token immediately after `/login`.
- [ ] Temporary token accurately binds `UserID` constraints for less than 2 minutes.
- [ ] Supplying correct OTP parameters to `/login/2fa` emits authorized token payload correctly.
- [ ] All previous `/login` validations structurally map cleanly utilizing `LoginResult`.
