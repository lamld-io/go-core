CREATE TABLE IF NOT EXISTS action_tokens (
    id         UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID         NOT NULL,
    type       VARCHAR(64)  NOT NULL,
    token_hash VARCHAR(255) NOT NULL,
    expires_at TIMESTAMPTZ  NOT NULL,
    used_at    TIMESTAMPTZ  NULL,
    revoked    BOOLEAN      NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT fk_action_tokens_user
        FOREIGN KEY (user_id)
        REFERENCES users (id)
        ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_action_tokens_token_hash
    ON action_tokens (token_hash);

CREATE INDEX IF NOT EXISTS idx_action_tokens_user_type
    ON action_tokens (user_id, type);

CREATE INDEX IF NOT EXISTS idx_action_tokens_active
    ON action_tokens (type, expires_at)
    WHERE revoked = false AND used_at IS NULL;

COMMENT ON TABLE action_tokens IS 'Token mot lan cho xac thuc email va dat lai mat khau';
COMMENT ON COLUMN action_tokens.type IS 'email_verification | password_reset';
