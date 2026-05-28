CREATE TABLE IF NOT EXISTS chunk_references (
    id                  UUID         PRIMARY KEY,
    base_knowledge_id   UUID         NOT NULL REFERENCES base_knowledge(id) ON DELETE CASCADE,
    chunk_id            TEXT         NOT NULL,
    qdrant_point_id     TEXT         NOT NULL,
    title               TEXT         NOT NULL,
    header_path         TEXT[]       NOT NULL DEFAULT '{}',
    token_count         INTEGER      NOT NULL DEFAULT 0,
    source_start_line   INTEGER      NOT NULL DEFAULT 0,
    source_end_line     INTEGER      NOT NULL DEFAULT 0,
    description         TEXT         NOT NULL DEFAULT '',
    keywords            TEXT[]       NOT NULL DEFAULT '{}',
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE (base_knowledge_id, chunk_id)
);

CREATE INDEX IF NOT EXISTS idx_chunk_references_bk             ON chunk_references (base_knowledge_id);
CREATE INDEX IF NOT EXISTS idx_chunk_references_qdrant_point   ON chunk_references (qdrant_point_id);
