<!-- Generated: 2026-04-15 | Files scanned: 60+ | Token estimate: ~700 -->

# Architecture Overview

## System Diagram

```
Internet
    │
    ▼
┌──────────────────────────────────────────────────┐
│  Gateway Service  :8080                           │
│  Recovery → CORS → RequestID →                   │
│  Logging → RateLimit (in-memory) → ProxyHandler  │
└───────────────┬──────────────────────────────────┘
                │ HTTP proxy (longest prefix match)
                │
    ┌───────────▼────────────────────────┐
    │  Auth Service  :8081               │
    │  Recovery → CORS →                 │
    │  IPRateLimiter (Redis) →           │
    │  AuthMiddleware (JWT + Blacklist)   │
    └────┬──────────────────┬────────────┘
         │ GORM / pgx       │ go-redis
         ▼                  ▼
    ┌────────────┐   ┌──────────────┐
    │ PostgreSQL │   │    Redis     │
    │  :5432     │   │   :6379     │
    │  auth_db   │   │ (tuỳ chọn)  │
    └────────────┘   └──────────────┘
```

## Services

| Service    | Port | Responsibility |
|------------|------|----------------|
| Gateway    | 8080 | Entry point, reverse proxy, JWT pre-validation, rate limiting (in-memory) |
| Auth       | 8081 | User registration, login, 2FA (TOTP), token issuance, session management, token blacklist, lockout policy |
| PostgreSQL | 5432 | Persistent store (auth_db) |
| Redis      | 6379 | Token blacklist (logout), per-IP rate limiting (tuỳ chọn, có fallback) |

## Module

```
github.com/base-go/base   (Go 1.25.0)
```

## Request Flow

```
Client → Gateway:8080
  → match route (longest prefix)
  → validate JWT if RequiresAuth
  → forward to upstream (X-Forwarded-* headers)
    → Auth:8081 handles business logic
      → IP Rate Limiter (Redis) for public endpoints
      → AuthMiddleware: JWT verify + Token Blacklist (Redis) for protected
      → GORM → PostgreSQL
```

## Shared Packages (`pkg/`)

- `pkg/jwt`        — RS256 token issue/validate
- `pkg/middleware`  — CORS, IP Rate Limiter (Redis-backed)
- `pkg/apperror`   — typed domain errors → HTTP status auto-mapping
- `pkg/response`   — standardised HTTP response helpers
