ALTER TABLE raw_documents
    ADD COLUMN IF NOT EXISTS workspace_id UUID NULL REFERENCES workspaces(id) ON DELETE RESTRICT;

CREATE INDEX IF NOT EXISTS idx_raw_documents_workspace ON raw_documents (workspace_id);

CREATE TABLE IF NOT EXISTS raw_document_topics (
    raw_doc_id  UUID NOT NULL REFERENCES raw_documents(id) ON DELETE CASCADE,
    topic_id    UUID NOT NULL REFERENCES topics(id)        ON DELETE CASCADE,
    PRIMARY KEY (raw_doc_id, topic_id)
);

CREATE INDEX IF NOT EXISTS idx_raw_document_topics_topic ON raw_document_topics (topic_id);
