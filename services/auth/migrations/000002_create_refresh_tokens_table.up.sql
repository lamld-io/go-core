-- 000002_create_refresh_tokens_table.up.sql
-- Tạo bảng refresh_tokens cho Auth Service.

CREATE TABLE IF NOT EXISTS refresh_tokens (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID        NOT NULL,
    token_hash VARCHAR(255) NOT NULL,
    expires_at TIMESTAMPTZ  NOT NULL,
    revoked    BOOLEAN      NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT now(),

    -- Foreign key tới users, cascade delete khi user bị xoá.
    CONSTRAINT fk_refresh_tokens_user
        FOREIGN KEY (user_id)
        REFERENCES users (id)
        ON DELETE CASCADE
);

-- Index unique trên token_hash để tìm nhanh khi refresh.
CREATE UNIQUE INDEX IF NOT EXISTS idx_refresh_tokens_token_hash
    ON refresh_tokens (token_hash);

-- Index trên user_id để revoke tất cả token của 1 user nhanh.
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id
    ON refresh_tokens (user_id);

-- Index trên expires_at để cleanup job xoá token hết hạn hiệu quả.
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_expires_at
    ON refresh_tokens (expires_at)
    WHERE revoked = false;

COMMENT ON TABLE refresh_tokens IS 'Bảng lưu trữ refresh token (hash) cho Auth Service';
COMMENT ON COLUMN refresh_tokens.token_hash IS 'SHA-256 hash của refresh token, không lưu token gốc';
COMMENT ON COLUMN refresh_tokens.revoked IS 'true = token đã bị thu hồi (logout hoặc refresh rotation)';
