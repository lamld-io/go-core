<!-- Generated: 2026-04-14 | Files scanned: 52 | Token estimate: ~650 -->

# Architecture Overview

## System Diagram

```
Internet
    │
    ▼
┌─────────────────────────────────────────────┐
│  Gateway Service  :8080                      │
│  Recovery → CORS → RequestID →              │
│  Logging → RateLimit → ProxyHandler         │
└───────────────┬─────────────────────────────┘
                │ HTTP proxy (longest prefix match)
                │
    ┌───────────▼───────────┐
    │  Auth Service  :8081  │
    │  Recovery → CORS →    │
    │  AuthMiddleware       │
    └───────────┬───────────┘
                │ GORM / pgx
                ▼
    ┌───────────────────────┐
    │  PostgreSQL  :5432    │
    │  auth_db              │
    └───────────────────────┘
```

## Services

| Service | Port | Responsibility |
|---------|------|----------------|
| Gateway | 8080 | Entry point, reverse proxy, JWT pre-validation, rate limiting |
| Auth    | 8081 | User registration, login, token issuance, lockout policy |
| PostgreSQL | 5432 | Persistent store (auth_db) |

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
      → GORM → PostgreSQL
```

## Shared Packages (`pkg/`)

- `pkg/jwt`        — RS256 token issue/validate
- `pkg/middleware`  — CORS (shared across services)
- `pkg/apperror`   — typed domain errors
- `pkg/response`   — standardised HTTP response helpers
