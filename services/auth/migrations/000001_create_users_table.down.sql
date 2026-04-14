-- 000001_create_users_table.down.sql
-- Rollback: xoá bảng users.

DROP INDEX IF EXISTS idx_users_deleted_at;
DROP INDEX IF EXISTS idx_users_role;
DROP INDEX IF EXISTS idx_users_email_unique;
DROP TABLE IF EXISTS users;
