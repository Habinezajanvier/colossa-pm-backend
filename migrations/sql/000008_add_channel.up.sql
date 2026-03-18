
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS type VARCHAR(10) NOT NULL DEFAULT 'dm';
ALTER TABLE conversations ADD CONSTRAINT chk_conversations_type CHECK (type IN ('dm', 'channel'));
ALTER TABLE conversations DROP CONSTRAINT IF EXISTS uq_conversations_members;
ALTER TABLE conversations DROP CONSTRAINT IF EXISTS chk_conversations_members;
ALTER TABLE conversations DROP COLUMN IF EXISTS member_one;
ALTER TABLE conversations DROP COLUMN IF EXISTS member_two;

-- Conversation participants — replaces member_one/member_two for DMs
-- and channel_members for channels
CREATE TABLE IF NOT EXISTS conversation_participants (
    conversation_id UUID        NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    user_id         UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    is_admin        BOOLEAN     NOT NULL DEFAULT FALSE,
    joined_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (conversation_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_conversation_participants_user_id ON conversation_participants (user_id);
CREATE INDEX IF NOT EXISTS idx_conversation_participants_conv_id ON conversation_participants (conversation_id);

-- Channels — metadata only, messaging goes through conversations
CREATE TABLE IF NOT EXISTS channels (
    id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id    UUID         NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    conversation_id UUID         NOT NULL UNIQUE REFERENCES conversations(id) ON DELETE CASCADE,
    created_by      UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name            VARCHAR(100) NOT NULL,
    description     TEXT         DEFAULT NULL,
    is_private      BOOLEAN      NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_channels_workspace_name UNIQUE (workspace_id, name),
    CONSTRAINT chk_channels_name CHECK (name <> '')
);

CREATE INDEX IF NOT EXISTS idx_channels_workspace_id    ON channels (workspace_id);
CREATE INDEX IF NOT EXISTS idx_channels_conversation_id ON channels (conversation_id);
CREATE INDEX IF NOT EXISTS idx_channels_created_by      ON channels (created_by);