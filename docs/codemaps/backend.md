<!-- Generated: 2026-04-15 | Files scanned: 60+ | Token estimate: ~1100 -->

# Backend Architecture

## Auth Service (`services/auth`)

### Routes  `services/auth/internal/delivery/http/router.go`

```
GET  /health                                → AuthHandler.HealthCheck

# Public endpoints (rate-limited per-IP via Redis)
POST /api/v1/auth/register                  → AuthHandler.Register
POST /api/v1/auth/login                     → AuthHandler.Login
POST /api/v1/auth/login/2fa                 → AuthHandler.Login2FA
POST /api/v1/auth/verify-email              → AuthHandler.VerifyEmail
POST /api/v1/auth/resend-verification-email → AuthHandler.ResendVerificationEmail
POST /api/v1/auth/forgot-password           → AuthHandler.ForgotPassword
POST /api/v1/auth/reset-password            → AuthHandler.ResetPassword
POST /api/v1/auth/refresh                   → AuthHandler.RefreshToken
GET  /api/v1/auth/validate                  → AuthHandler.ValidateToken   [no auth]

# Protected endpoints (JWT + token blacklist check)
POST /api/v1/auth/logout                    → AuthHandler.Logout          [JWT required]
GET  /api/v1/auth/profile                   → AuthHandler.GetProfile      [JWT required]
GET  /api/v1/auth/sessions                  → AuthHandler.GetSessions     [JWT required]
DELETE /api/v1/auth/sessions/:id            → AuthHandler.DeleteSession   [JWT required]
POST /api/v1/auth/2fa/setup                 → AuthHandler.Setup2FA        [JWT required]
POST /api/v1/auth/2fa/verify                → AuthHandler.Verify2FASetup  [JWT required]
GET  /api/v1/auth/security/lockout-policy   → AuthHandler.GetLoginLockoutPolicy [JWT]
PUT  /api/v1/auth/security/lockout-policy   → AuthHandler.UpdateLoginLockoutPolicy [JWT]
```

### Middleware Chain (Auth)

```
Recovery → CORS → [IPRateLimiter (Redis, public routes)] → [AuthMiddleware (JWT + TokenBlacklist, protected routes)]
```

### Layered Architecture

```
delivery/http/handler/auth_handler.go
    → domain.AuthService (interface)
        ← usecase/auth_usecase.go  (implementation)
            → domain.UserRepository
            → domain.TokenRepository
            → domain.ActionTokenRepository
            → domain.LoginLockoutPolicyRepository
            → domain.TokenBlacklist
            → domain.EmailSender
            → pkg/jwt.Manager
                ← repository/postgres/user_repository.go
                ← repository/postgres/token_repository.go
                ← repository/postgres/action_token_repository.go
                ← repository/postgres/login_lockout_policy_repository.go
                ← repository/redisrepo/token_blacklist.go
                    → GORM → PostgreSQL
                    → go-redis → Redis
```

### Key Files (Auth)

| File | Purpose |
|------|---------|
| `internal/domain/service.go` | AuthService interface (Register, Login, 2FA, Session, Blacklist, Lockout) |
| `internal/domain/repository.go` | Repository interfaces: User, Token, ActionToken, LockoutPolicy |
| `internal/domain/user.go` | User entity: TOTPSecret, Is2FAEnabled, LockedUntil |
| `internal/domain/errors.go` | Domain errors → AppError → HTTP auto-mapping |
| `internal/usecase/auth_usecase.go` | Business logic implementation |
| `internal/delivery/http/handler/auth_handler.go` | HTTP handlers (17 endpoints) |
| `internal/delivery/http/router.go` | Route registration with rate limiter + auth middleware |
| `internal/repository/postgres/` | GORM implementations: user, token, action_token, lockout repos |
| `internal/repository/redisrepo/token_blacklist.go` | Redis token blacklist (logout) |
| `internal/platform/config/config.go` | Config struct: Server, DB, Redis, RateLimit, JWT, Password, Security, Email |
| `internal/platform/database/` | DB connection & migrations runner |
| `internal/platform/email/sender.go` | Email sender (SMTP / log-only fallback) |
| `internal/bootstrap/app.go` | Dependency wiring: config → DB → Redis → JWT → repos → usecase → handler → router |

### Key Domain Types

| Type | Description |
|------|-------------|
| `User` | Entity: ID, Email, PasswordHash, TOTPSecret, Is2FAEnabled, LockedUntil |
| `TokenPair` | AccessToken + RefreshToken |
| `LoginResult` | User + TokenPair + Requires2FA + TempToken |
| `Setup2FAResponse` | Secret + SecretURL (for QR code) |
| `TokenBlacklist` | Interface: BlacklistToken + IsBlacklisted |
| `ClientMetadata` | IP, UserAgent (for session tracking) |

---

## Gateway Service (`services/gateway`)

### Routes  `services/gateway/internal/delivery/http/router.go`

```
GET  /health    → ProxyHandler.HealthCheck
*    /*          → ProxyHandler.Handle  (catch-all reverse proxy)
```

### Middleware Chain (Gateway)

```
Recovery → CORS → RequestID → Logging → [RateLimit (per-IP token bucket, in-memory)] → ProxyHandler
```

### Routing Logic

- Config loaded from `routes.yaml` (or `routes.docker.yaml` in Docker)
- `usecase/proxy_usecase.go` performs **longest prefix match**
- Each route has: `Prefix`, `Target`, `StripPrefix`, `RequiresAuth`, `Methods`
- JWT pre-validation done in Gateway before forwarding if `RequiresAuth = true`

### Key Files (Gateway)

| File | Purpose |
|------|---------|
| `internal/domain/route.go` | Route entity |
| `internal/domain/service.go` | ProxyService interface |
| `internal/usecase/proxy_usecase.go` | Route matching + request building |
| `internal/delivery/http/handler/proxy_handler.go` | Proxy HTTP handler |
| `internal/delivery/http/router.go` | Route registration |

---

## Shared Packages (`pkg/`)

| Package | Files | Purpose |
|---------|-------|---------|
| `pkg/jwt` | `jwt.go`, `claims.go` | RS256 sign/verify, Manager struct |
| `pkg/middleware` | `cors.go`, `rate_limiter.go` | CORS shared middleware, IP Rate Limiter (Redis-backed) |
| `pkg/apperror` | `errors.go` | Typed app errors → HTTP status auto-mapping |
| `pkg/response` | — | Standardised JSON response helpers |
