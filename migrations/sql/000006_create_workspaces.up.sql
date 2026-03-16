CREATE TABLE IF NOT EXISTS workspaces (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(255) NOT NULL,
    description TEXT         DEFAULT NULL,
    owner_id    UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at  TIMESTAMP  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMP  NOT NULL DEFAULT NOW(),
    created_by UUID,
    created_by_names VARCHAR(255),
    updated_by VARCHAR(255),
    updated_by_names VARCHAR(255),
    deleted_at TIMESTAMP DEFAULT NULL,

    CONSTRAINT chk_workspaces_name CHECK (name <> '')
);

CREATE INDEX IF NOT EXISTS idx_workspaces_owner_id ON workspaces (owner_id);

CREATE TABLE IF NOT EXISTS workspace_members (
    workspace_id UUID        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    user_id      UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role         VARCHAR(50) NOT NULL DEFAULT 'member',
    joined_at    TIMESTAMP NOT NULL DEFAULT NOW(),

    PRIMARY KEY (workspace_id, user_id),
    CONSTRAINT chk_workspace_members_role CHECK (role IN ('admin', 'member', 'guest'))
);

CREATE INDEX IF NOT EXISTS idx_workspace_members_user_id ON workspace_members (user_id);

CREATE TABLE IF NOT EXISTS workspace_invites (
    id           UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID         NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    invited_by   UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    email        VARCHAR(255) NOT NULL,
    role         VARCHAR(50)  NOT NULL DEFAULT 'member',
    token        VARCHAR(64)  NOT NULL UNIQUE,
    used         BOOLEAN      NOT NULL DEFAULT FALSE,
    expires_at   TIMESTAMP  NOT NULL,
    created_at   TIMESTAMP  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMP  NOT NULL DEFAULT NOW(),
    created_by UUID,
    created_by_names VARCHAR(255),
    updated_by VARCHAR(255),
    updated_by_names VARCHAR(255),
    deleted_at TIMESTAMP DEFAULT NULL,

    CONSTRAINT chk_workspace_invites_role CHECK (role IN ('admin', 'member', 'guest'))
);

CREATE INDEX IF NOT EXISTS idx_workspace_invites_workspace_id ON workspace_invites (workspace_id);
CREATE INDEX IF NOT EXISTS idx_workspace_invites_email        ON workspace_invites (email);
CREATE INDEX IF NOT EXISTS idx_workspace_invites_token        ON workspace_invites (token);