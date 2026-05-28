CREATE TABLE IF NOT EXISTS base_knowledge_topics (
    base_knowledge_id   UUID NOT NULL REFERENCES base_knowledge(id) ON DELETE CASCADE,
    topic_id            UUID NOT NULL REFERENCES topics(id)         ON DELETE CASCADE,
    PRIMARY KEY (base_knowledge_id, topic_id)
);

CREATE INDEX IF NOT EXISTS idx_base_knowledge_topics_topic ON base_knowledge_topics (topic_id);
