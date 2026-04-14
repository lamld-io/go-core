-- 000002_create_refresh_tokens_table.down.sql
-- Rollback: xoá bảng refresh_tokens.

DROP INDEX IF EXISTS idx_refresh_tokens_expires_at;
DROP INDEX IF EXISTS idx_refresh_tokens_user_id;
DROP INDEX IF EXISTS idx_refresh_tokens_token_hash;
DROP TABLE IF EXISTS refresh_tokens;
