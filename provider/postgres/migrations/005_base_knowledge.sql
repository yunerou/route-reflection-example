CREATE TABLE IF NOT EXISTS base_knowledge (
    id                  UUID         PRIMARY KEY,
    raw_doc_id          UUID         NOT NULL REFERENCES raw_documents(id) ON DELETE RESTRICT,
    workspace_id        UUID         NOT NULL REFERENCES workspaces(id)    ON DELETE RESTRICT,
    status              TEXT         NOT NULL DEFAULT 'active',
    superseded_by_id    UUID         NULL REFERENCES base_knowledge(id)    ON DELETE SET NULL,
    superseded_at       TIMESTAMPTZ  NULL,
    source_hash         TEXT         NOT NULL,
    generator_version   TEXT         NOT NULL,
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deprecated_at       TIMESTAMPTZ  NULL,
    UNIQUE (raw_doc_id)
);

CREATE INDEX IF NOT EXISTS idx_base_knowledge_workspace      ON base_knowledge (workspace_id);
CREATE INDEX IF NOT EXISTS idx_base_knowledge_status         ON base_knowledge (status);
CREATE INDEX IF NOT EXISTS idx_base_knowledge_superseded_by  ON base_knowledge (superseded_by_id);
