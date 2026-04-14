-- 000001_create_users_table.up.sql
-- Tạo bảng users cho Auth Service.

-- Đảm bảo extension uuid-ossp hoặc pgcrypto có sẵn để dùng gen_random_uuid().
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS users (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    email         VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    full_name     VARCHAR(255) NULL,
    role          VARCHAR(50)  NOT NULL DEFAULT 'user',
    is_active     BOOLEAN      NOT NULL DEFAULT true,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at    TIMESTAMPTZ  NULL
);

-- Index unique trên email, chỉ áp dụng cho record chưa bị soft-delete.
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email_unique
    ON users (email)
    WHERE deleted_at IS NULL;

-- Index cho tìm kiếm theo role.
CREATE INDEX IF NOT EXISTS idx_users_role ON users (role);

-- Index cho soft-delete queries.
CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users (deleted_at);

COMMENT ON TABLE users IS 'Bảng lưu trữ thông tin người dùng cho Auth Service';
COMMENT ON COLUMN users.email IS 'Email đăng nhập, unique khi chưa bị xoá';
COMMENT ON COLUMN users.password_hash IS 'Bcrypt hash của mật khẩu';
COMMENT ON COLUMN users.full_name IS 'Họ tên đầy đủ, nullable';
COMMENT ON COLUMN users.role IS 'Role: admin, user';
COMMENT ON COLUMN users.is_active IS 'Trạng thái tài khoản, false = bị khoá';
COMMENT ON COLUMN users.deleted_at IS 'Soft-delete timestamp, NULL = chưa xoá';
