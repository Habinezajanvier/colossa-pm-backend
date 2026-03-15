CREATE TABLE IF NOT EXISTS verification_tokens (
    id         UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token      VARCHAR(64)  NOT NULL,
    type       VARCHAR(32)  NOT NULL,
    used       BOOLEAN      NOT NULL DEFAULT FALSE,
    expires_at TIMESTAMPTZ  NOT NULL,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
 
CREATE INDEX IF NOT EXISTS idx_verification_tokens_user_id ON verification_tokens (user_id);
CREATE INDEX IF NOT EXISTS idx_verification_tokens_token   ON verification_tokens (token);
CREATE INDEX IF NOT EXISTS idx_verification_tokens_type    ON verification_tokens (type);
 
ALTER TABLE users ADD COLUMN IF NOT EXISTS is_verified BOOLEAN NOT NULL DEFAULT FALSE;