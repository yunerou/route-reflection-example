CREATE TABLE IF NOT EXISTS raw_documents (
    id                   UUID        PRIMARY KEY,
    filename             TEXT        NOT NULL,
    mime_type            TEXT        NOT NULL,
    s3_key               TEXT        NOT NULL UNIQUE,
    file_size            BIGINT      NOT NULL,
    uploaded_by          UUID        NOT NULL,

    previous_version_id  UUID        NULL,
    change_type          TEXT        NULL,
    changed_sections     TEXT[]      NULL,

    parse_status         TEXT        NULL,
    parsed_content       JSONB       NULL,
    parsed_at            TIMESTAMPTZ NULL,

    verification_status  TEXT        NOT NULL DEFAULT 'pending',
    verification_method  TEXT        NULL,
    verified_at          TIMESTAMPTZ NULL,
    verified_by          UUID        NULL,
    rejection_reason     TEXT        NULL,

    base_knowledge_id    UUID        NULL,

    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_raw_documents_verification_status ON raw_documents (verification_status);
CREATE INDEX IF NOT EXISTS idx_raw_documents_uploaded_by        ON raw_documents (uploaded_by);
CREATE INDEX IF NOT EXISTS idx_raw_documents_created_at         ON raw_documents (created_at);
CREATE INDEX IF NOT EXISTS idx_raw_documents_change_type        ON raw_documents (change_type);
CREATE INDEX IF NOT EXISTS idx_raw_documents_mime_type          ON raw_documents (mime_type);
