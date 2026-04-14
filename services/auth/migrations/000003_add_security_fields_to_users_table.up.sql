ALTER TABLE users
    ADD COLUMN IF NOT EXISTS email_verified BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS failed_login_attempts INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS locked_until TIMESTAMPTZ NULL;

COMMENT ON COLUMN users.email_verified IS 'true = email da duoc xac thuc';
COMMENT ON COLUMN users.failed_login_attempts IS 'So lan dang nhap sai lien tiep hien tai';
COMMENT ON COLUMN users.locked_until IS 'Thoi diem mo khoa tam thoi do brute-force protection';
