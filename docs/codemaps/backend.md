<!-- Generated: 2026-04-14 | Files scanned: 52 | Token estimate: ~900 -->

# Backend Architecture

## Auth Service (`services/auth`)

### Routes  `services/auth/internal/delivery/http/router.go`

```
GET  /health                              → AuthHandler.HealthCheck
POST /api/v1/auth/register               → AuthHandler.Register
POST /api/v1/auth/login                  → AuthHandler.Login
POST /api/v1/auth/verify-email           → AuthHandler.VerifyEmail
POST /api/v1/auth/resend-verification-email → AuthHandler.ResendVerificationEmail
POST /api/v1/auth/forgot-password        → AuthHandler.ForgotPassword
POST /api/v1/auth/reset-password         → AuthHandler.ResetPassword
POST /api/v1/auth/refresh                → AuthHandler.RefreshToken
GET  /api/v1/auth/validate               → AuthHandler.ValidateToken   [no auth]
POST /api/v1/auth/logout                 → AuthHandler.Logout          [JWT required]
GET  /api/v1/auth/profile                → AuthHandler.GetProfile       [JWT required]
GET  /api/v1/auth/security/lockout-policy → AuthHandler.GetLoginLockoutPolicy [JWT]
PUT  /api/v1/auth/security/lockout-policy → AuthHandler.UpdateLoginLockoutPolicy [JWT]
```

### Middleware Chain (Auth)

```
Recovery → CORS → [AuthMiddleware (JWT, protected routes only)]
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
            → domain.EmailSender
            → pkg/jwt.Manager
                ← repository/postgres/user_repository.go
                ← repository/postgres/token_repository.go
                ← repository/postgres/action_token_repository.go
                ← repository/postgres/login_lockout_policy_repository.go
                    → GORM → PostgreSQL
```

### Key Files (Auth)

| File | Purpose | Size |
|------|---------|------|
| `services/auth/internal/domain/service.go` | AuthService interface | 55 ln |
| `services/auth/internal/domain/repository.go` | Repository interfaces (4 interfaces) | 58 ln |
| `services/auth/internal/domain/user.go` | User entity, roles | 52 ln |
| `services/auth/internal/usecase/auth_usecase.go` | Business logic | 18 KB |
| `services/auth/internal/delivery/http/handler/auth_handler.go` | HTTP handler | 7.8 KB |
| `services/auth/internal/delivery/http/router.go` | Route registration | 56 ln |
| `services/auth/internal/repository/postgres/user_repository.go` | UserRepo impl | 2.5 KB |
| `services/auth/internal/repository/postgres/token_repository.go` | RefreshToken impl | 2.7 KB |
| `services/auth/internal/platform/database/` | DB connection & migrations runner | — |
| `services/auth/internal/platform/email/sender.go` | Email sender (SMTP) | 3 KB |

---

## Gateway Service (`services/gateway`)

### Routes  `services/gateway/internal/delivery/http/router.go`

```
GET  /health    → ProxyHandler.HealthCheck
*    /*          → ProxyHandler.Handle  (catch-all reverse proxy)
```

### Middleware Chain (Gateway)

```
Recovery → CORS → RequestID → Logging → [RateLimit (per-IP token bucket, if enabled)] → ProxyHandler
```

### Routing Logic

- Config loaded from `routes.yaml` (or `routes.docker.yaml` in Docker)
- `usecase/proxy_usecase.go` performs **longest prefix match**
- Each route has: `Prefix`, `Target`, `StripPrefix`, `RequiresAuth`, `Methods`
- JWT pre-validation done in Gateway before forwarding if `RequiresAuth = true`

### Key Files (Gateway)

| File | Purpose | Size |
|------|---------|------|
| `services/gateway/internal/domain/route.go` | Route entity | 34 ln |
| `services/gateway/internal/domain/service.go` | ProxyService interface | — |
| `services/gateway/internal/usecase/proxy_usecase.go` | Route matching + request building | 105 ln |
| `services/gateway/internal/delivery/http/handler/proxy_handler.go` | Proxy HTTP handler | 2.7 KB |
| `services/gateway/internal/delivery/http/router.go` | Route registration | 54 ln |

---

## Shared Packages (`pkg/`)

| Package | Files | Purpose |
|---------|-------|---------|
| `pkg/jwt` | `jwt.go`, `claims.go` | RS256 sign/verify, Manager struct |
| `pkg/middleware` | `cors.go` | CORS shared middleware |
| `pkg/apperror` | `errors.go` | Typed app errors |
| `pkg/response` | — | Standardised JSON response helpers |
