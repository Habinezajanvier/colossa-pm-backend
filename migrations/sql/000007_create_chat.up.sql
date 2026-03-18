-- ============================================================
-- Chat
-- ============================================================
CREATE TABLE IF NOT EXISTS conversations (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    member_one   UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    member_two   UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_conversations_members UNIQUE (workspace_id, member_one, member_two),
    CONSTRAINT chk_conversations_members CHECK (member_one < member_two)
);

CREATE INDEX IF NOT EXISTS idx_conversations_workspace_id ON conversations (workspace_id);
CREATE INDEX IF NOT EXISTS idx_conversations_member_one   ON conversations (member_one);
CREATE INDEX IF NOT EXISTS idx_conversations_member_two   ON conversations (member_two);

CREATE TABLE IF NOT EXISTS chat_messages (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID        NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    sender_id       UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    parent_id       UUID        REFERENCES chat_messages(id) ON DELETE SET NULL DEFAULT NULL,
    body            TEXT        DEFAULT NULL,
    deleted_at      TIMESTAMPTZ DEFAULT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_chat_messages_conversation_id ON chat_messages (conversation_id);
CREATE INDEX IF NOT EXISTS idx_chat_messages_sender_id       ON chat_messages (sender_id);
CREATE INDEX IF NOT EXISTS idx_chat_messages_parent_id       ON chat_messages (parent_id);
CREATE INDEX IF NOT EXISTS idx_chat_messages_created_at      ON chat_messages (created_at DESC);

CREATE TABLE IF NOT EXISTS message_reactions (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id UUID        NOT NULL REFERENCES chat_messages(id) ON DELETE CASCADE,
    user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    emoji      VARCHAR(10) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_message_reactions UNIQUE (message_id, user_id, emoji)
);

CREATE INDEX IF NOT EXISTS idx_message_reactions_message_id ON message_reactions (message_id);

-- ============================================================
-- Attachments
-- ============================================================
CREATE TABLE IF NOT EXISTS attachments (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_type VARCHAR(100) NOT NULL,
    entity_id   UUID         NOT NULL,
    url         TEXT         NOT NULL,
    name        VARCHAR(255) NOT NULL,
    size        BIGINT       NOT NULL,
    mime_type   VARCHAR(100) NOT NULL,
    file_type   VARCHAR(20)  NOT NULL,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_attachments_file_type CHECK (file_type IN ('image', 'video', 'audio', 'document'))
);

CREATE INDEX IF NOT EXISTS idx_attachments_entity ON attachments (entity_type, entity_id);