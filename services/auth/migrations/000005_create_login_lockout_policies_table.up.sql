CREATE TABLE IF NOT EXISTS login_lockout_policies (
    id                    BIGINT       PRIMARY KEY,
    max_failed_attempts   INTEGER      NOT NULL,
    lock_duration_seconds BIGINT       NOT NULL,
    created_at            TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ  NOT NULL DEFAULT now()
);

COMMENT ON TABLE login_lockout_policies IS 'Policy khoa tai khoan khi dang nhap sai, co the chinh runtime';
