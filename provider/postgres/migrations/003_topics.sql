CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE IF NOT EXISTS topics (
    id                    UUID         PRIMARY KEY,
    workspace_id          UUID         NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name                  TEXT         NOT NULL,
    slug                  TEXT         NOT NULL,
    description           TEXT         NOT NULL,
    description_embedding vector(768)  NULL,
    created_by            UUID         NOT NULL,
    created_at            TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE (workspace_id, slug)
);

CREATE INDEX IF NOT EXISTS idx_topics_workspace ON topics (workspace_id);
CREATE INDEX IF NOT EXISTS idx_topics_created_at ON topics (created_at);
