CREATE TABLE IF NOT EXISTS messages (
    id         UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    recipient  VARCHAR(255) NOT NULL,
    type       VARCHAR(50)  NOT NULL,
    status     VARCHAR(20)  NOT NULL,
    error      TEXT         DEFAULT NULL,
    event_type VARCHAR(100) DEFAULT NULL,
    event_id   UUID         DEFAULT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    created_by UUID,
    created_by_names VARCHAR(255),
    updated_by VARCHAR(255),
    updated_by_names VARCHAR(255),
    deleted_at TIMESTAMP DEFAULT NULL,

    CONSTRAINT chk_messages_status CHECK (status IN ('sent', 'failed')),
    CONSTRAINT chk_messages_type   CHECK (type   IN ('email_verification', 'change_password'))
);

CREATE INDEX IF NOT EXISTS idx_messages_user_id        ON messages (user_id);
CREATE INDEX IF NOT EXISTS idx_messages_status         ON messages (status);
CREATE INDEX IF NOT EXISTS idx_messages_type           ON messages (type);
CREATE INDEX IF NOT EXISTS idx_messages_event_type_id  ON messages (event_type, event_id);