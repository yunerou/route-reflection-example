ALTER TABLE raw_documents
    ADD COLUMN IF NOT EXISTS parsed_chunks JSONB NULL;
