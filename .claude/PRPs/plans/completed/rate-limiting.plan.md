# Plan: Rate Limiting Middleware

## Summary
Implement an IP-based rate limiting middleware for the Auth Service using Redis (via `go-redis`) with an in-memory fallback. This will protect public endpoints like `/login` and `/register` from brute-force and DDoS attacks. 

## User Story
As a System Administrator, I want the Auth API to limit requests from a single IP, so that the system is safe from brute-force credential stuffing and denial of service.

## Problem → Solution
No IP-based request limits on auth endpoints → Redis-backed rate limiting middleware returning 429 Too Many Requests when limits are exceeded.

## Metadata
- **Complexity**: Medium
- **Source PRD**: `.claude/PRPs/prds/core-security-upgrade.prd.md`
- **PRD Phase**: 1 (Rate Limiting)
- **Estimated Files**: 5

---

## UX Design

### Before
N/A — internal change. Attackers can spam `/api/v1/auth/login` infinitely.

### After
N/A — internal change. Attackers spamming the endpoint will receive `429 Too Many Requests` after `N` attempts per minute.

### Interaction Changes
| Touchpoint | Before | After | Notes |
|---|---|---|---|
| Request API `/login` | 200/401 infinitely | Returns 429 after limit | Threshold configured via ENVs |

---

## Mandatory Reading

| Priority | File | Lines | Why |
|---|---|---|---|
| P0 | `pkg/apperror/errors.go` | 1-115 | Follow the `AppError` format, specifically `apperror.RateLimited()` |
| P1 | `services/auth/internal/delivery/http/middleware/auth_middleware.go` | 1-87 | Understand how Gin middleware and `response.AbortWithError` is implemented |
| P2 | `services/auth/internal/platform/config/config.go` | 1-179 | Need to add Redis and RateLimitEnv config here |

## External Documentation

| Topic | Source | Key Takeaway |
|---|---|---|
| go-redis rate limiting | github.com/go-redis/redis_rate/v10 | Standard token bucket implementation for Redis in Go |

---

## Patterns to Mirror

### ERROR_HANDLING
// SOURCE: pkg/middleware/auth_middleware.go:33
```go
if authHeader == "" {
    response.AbortWithError(c, apperror.Unauthorized("authorization header is required"))
    return
}
```

### APP_ERROR
// SOURCE: pkg/apperror/errors.go:103
```go
func RateLimited() *AppError {
    return &AppError{Code: CodeRateLimited, Message: "rate limit exceeded"}
}
```

### CONFIG_PATTERN
// SOURCE: services/auth/internal/platform/config/config.go:102
```go
Host: getEnv("DB_HOST", "localhost"),
```

---

## Files to Change

| File | Action | Justification |
|---|---|---|
| `go.mod` | UPDATE | Run `go get github.com/redis/go-redis/v9` and `github.com/go-redis/redis_rate/v10` |
| `services/auth/internal/platform/config/config.go` | UPDATE | Add `RedisConfig` and `RateLimitConfig` structures |
| `pkg/middleware/rate_limiter.go` | CREATE | Genereic Gin middleware using Redis Rate Limiter |
| `services/auth/internal/delivery/http/router.go` | UPDATE | Inject RateLimiter middleware to public Auth routes |
| `services/auth/cmd/main.go` | UPDATE | Initialize Redis client and pass to Router |

## NOT Building

- Token Bucket algorithms from scratch.
- Distributed lock for rate limiting (we use the atomic redis_rate package).
- Rate limits based on UserID (Phase 1 focuses ONLY on IP-based limits for unauthenticated endpoints).

---

## Step-by-Step Tasks

### Task 1: Install Dependencies
- **ACTION**: Add Redis packages
- **IMPLEMENT**: Run `cd services/auth && go get github.com/redis/go-redis/v9 github.com/go-redis/redis_rate/v10 golang.org/x/time/rate`
- **VALIDATE**: Ensure `go.mod` has the new dependencies.

### Task 2: Configure Redis in Config
- **ACTION**: Modify `services/auth/internal/platform/config/config.go`
- **IMPLEMENT**: 
  - Add `RedisConfig` struct with `Host`, `Port`, `Password`, `DB`.
  - Add `RateLimitConfig` with `LoginLimit` (int), `GeneralLimit` (int).
  - Update `Load()` to read `REDIS_HOST`, `RATE_LIMIT_LOGIN` (default 5 req/min), etc.
- **MIRROR**: CONFIG_PATTERN
- **GOTCHA**: Make sure to provide sane defaults so the service doesn't crash on existing deployments.

### Task 3: Implement Rate Limiter Middleware
- **ACTION**: Create `pkg/middleware/rate_limiter.go`
- **IMPLEMENT**: 
  - Create `RateLimiter` component taking `*redis.Client`.
  - Implement a fallback using `golang.org/x/time/rate` if Redis is down or unavailable (ping fails).
  - Return a Gin `HandlerFunc`. Extract `c.ClientIP()`. Check limit. If exceeded, `response.AbortWithError(c, apperror.RateLimited())`.
- **IMPORTS**: `github.com/gin-gonic/gin`, `github.com/base-go/base/pkg/apperror`, `github.com/base-go/base/pkg/response`
- **MIRROR**: ERROR_HANDLING

### Task 4: Integrate into Setup
- **ACTION**: Update `router.go` and `main.go`.
- **IMPLEMENT**: 
  - In `main.go`, init standard `redis.NewClient`. Pass to `NewRouter`.
  - In `router.go`, add `loginLimiter := middleware.IPRateLimiter(redisClient, config.LoginLimit, time.Minute)` to `/login`, `/register`, `/forgot-password`, `/verify-email`.
- **VALIDATE**: The auth routes must have the middleware attached before `authHandler.Login`.

---

## Testing Strategy

### Unit Tests

| Test | Input | Expected Output | Edge Case? |
|---|---|---|---|
| Middleware IP Limit | 6 HTTP requests from IP `192.168.1.1` to `/login` | 1-5 return 200 (or mocked handler res), 6th returns 429 Too Many Requests | Yes, Rate threshold exceeded |
| Redis down Fallback | Redis connection refused | Memory local limiter takes over, still limits to 5 requests | Yes, Redis failure |

### Edge Cases Checklist
- [x] Redis connection timeout
- [x] Client IP extraction from reverse proxies (X-Forwarded-For) -> Handled by Gin automatically `c.ClientIP()`
- [x] Whitelisted IPs (Skip checking for localhost or internal network if necessary, out of scope for MVP but good to think about)

---

## Validation Commands

### Static Analysis
```bash
cd services/auth && go vet ./...
```

### Unit Tests
```bash
cd services/auth && go test ./internal/delivery/http/... -v
```

### Manual Validation
- [ ] Boot server using `docker-compose up redis db` and `go run services/auth/cmd/main.go`
- [ ] Send 10 identical curl requests to `/api/v1/auth/login`. Verify attempts 6-10 return HTTP 429.

---

## Acceptance Criteria
- [ ] Redis dependency added.
- [ ] `rate_limiter.go` middleware created and supports fallback.
- [ ] Config structures updated.
- [ ] Router protects public auth routes.
- [ ] Validation commands pass.
