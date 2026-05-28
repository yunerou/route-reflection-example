CREATE TABLE IF NOT EXISTS workspaces (
    id          UUID        PRIMARY KEY,
    name        TEXT        NOT NULL,
    slug        TEXT        NOT NULL UNIQUE,
    created_by  UUID        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_workspaces_created_by ON workspaces (created_by);
CREATE INDEX IF NOT EXISTS idx_workspaces_created_at ON workspaces (created_at);

CREATE TABLE IF NOT EXISTS workspace_members (
    id            UUID        PRIMARY KEY,
    workspace_id  UUID        NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    user_id       UUID        NOT NULL,
    role          TEXT        NOT NULL,
    joined_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (workspace_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_workspace_members_user ON workspace_members (user_id);
CREATE INDEX IF NOT EXISTS idx_workspace_members_workspace ON workspace_members (workspace_id);
