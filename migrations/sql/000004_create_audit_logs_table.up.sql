CREATE TABLE IF NOT EXISTS audit_logs (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID         REFERENCES users(id) ON DELETE SET NULL DEFAULT NULL,
    action      VARCHAR(100) NOT NULL,
    entity_type VARCHAR(100) DEFAULT NULL,
    entity_id   UUID         DEFAULT NULL,
    old_values  JSONB        DEFAULT NULL,
    new_values  JSONB        DEFAULT NULL,
    request     JSONB        DEFAULT NULL,
    created_at  TIMESTAMP  NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_audit_logs_action CHECK (action <> '')
);

CREATE INDEX IF NOT EXISTS idx_audit_logs_user_id    ON audit_logs (user_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_entity     ON audit_logs (entity_type, entity_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs (created_at DESC);