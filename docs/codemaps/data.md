<!-- Generated: 2026-04-14 | Files scanned: 52 | Token estimate: ~550 -->

# Data Architecture

## Database: PostgreSQL 16 (`auth_db`)

### Tables Overview

```
users
  └── refresh_tokens  (FK: user_id → users.id  CASCADE DELETE)
  └── action_tokens   (FK: user_id → users.id  CASCADE DELETE)
  └── login_lockout_policies  (global policy row, no FK)
```

---

## Table: `users`

Migration: `services/auth/migrations/000001_create_users_table.up.sql`

| Column | Type | Constraints |
|--------|------|-------------|
| id | UUID | PK, default gen_random_uuid() |
| email | VARCHAR(255) | NOT NULL |
| password_hash | VARCHAR(255) | NOT NULL (bcrypt) |
| full_name | VARCHAR(255) | NULL |
| role | VARCHAR(50) | NOT NULL, default 'user' |
| is_active | BOOLEAN | NOT NULL, default true |
| created_at | TIMESTAMPTZ | NOT NULL, default now() |
| updated_at | TIMESTAMPTZ | NOT NULL, default now() |
| deleted_at | TIMESTAMPTZ | NULL (soft delete) |

**Indexes:**
- `idx_users_email_unique` — UNIQUE on `email` WHERE `deleted_at IS NULL`
- `idx_users_role` — on `role`
- `idx_users_deleted_at` — on `deleted_at`

**Roles:** `admin`, `user` (default)

---

## Table: `refresh_tokens`

Migration: `services/auth/migrations/000002_create_refresh_tokens_table.up.sql`

| Column | Type | Constraints |
|--------|------|-------------|
| id | UUID | PK |
| user_id | UUID | FK → users.id CASCADE DELETE |
| token_hash | VARCHAR(255) | NOT NULL (SHA-256 hash, not raw token) |
| expires_at | TIMESTAMPTZ | NOT NULL |
| revoked | BOOLEAN | NOT NULL, default false |
| created_at | TIMESTAMPTZ | NOT NULL |

**Indexes:**
- `idx_refresh_tokens_token_hash` — UNIQUE
- `idx_refresh_tokens_user_id`
- `idx_refresh_tokens_expires_at` WHERE `revoked = false`

---

## Table: `action_tokens`

Migration: `services/auth/migrations/000004_create_action_tokens_table.up.sql`

Dùng cho: email verification, password reset (one-time tokens).

| Column | Type | Constraints |
|--------|------|-------------|
| id | UUID | PK |
| user_id | UUID | FK → users.id CASCADE DELETE |
| type | VARCHAR(64) | `email_verification` \| `password_reset` |
| token_hash | VARCHAR(255) | NOT NULL (SHA-256) |
| expires_at | TIMESTAMPTZ | NOT NULL |
| used_at | TIMESTAMPTZ | NULL |
| revoked | BOOLEAN | NOT NULL, default false |
| created_at | TIMESTAMPTZ | NOT NULL |

**Indexes:**
- `idx_action_tokens_token_hash` — UNIQUE
- `idx_action_tokens_user_type` — on `(user_id, type)`
- `idx_action_tokens_active` — on `(type, expires_at)` WHERE `revoked=false AND used_at IS NULL`

---

## Table: `login_lockout_policies`

Migration: `services/auth/migrations/000005_create_login_lockout_policies_table.up.sql`

Global policy for account lockout (likely single-row table).

---

## Table: Security Fields on `users`

Migration: `services/auth/migrations/000003_add_security_fields_to_users_table.up.sql`

Fields added: `failed_login_attempts`, `locked_until` (nullable timestamp) — tracks brute-force protection state.

---

## Migration History

| # | Description |
|---|-------------|
| 000001 | Create `users` table |
| 000002 | Create `refresh_tokens` table |
| 000003 | Add security fields to `users` (failed_login_attempts, locked_until) |
| 000004 | Create `action_tokens` table |
| 000005 | Create `login_lockout_policies` table |

## ORM

- **GORM** v1.31.1 with `gorm.io/driver/postgres` (pgx v5 underneath)
- Soft delete via `deleted_at` (not using GORM's gorm.Model — custom model layer in `repository/postgres/model/`)
