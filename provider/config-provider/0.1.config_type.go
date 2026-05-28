package configprovider

type EnvType struct {
	Info *InfoT `koanf:"INFO"`

	AutoMigration bool `koanf:"AUTO_MIGRATION"`

	RateLimiter *RateLimiterT `koanf:"RATE_LIMITER"`

	WorkerPool *WorkerPoolT `koanf:"WORKER_POOL"`

	ExtractHeader *ExtractHeaderT `koanf:"EXTRACT_HEADER"`
	Cors          *CorsT          `koanf:"CORS"`

	Otel   *OtelT   `koanf:"OTEL"`
	Valkey *ValkeyT `koanf:"VALKEY"`

	Postgres *PostgresT `koanf:"POSTGRES"`
	Qdrant   *QdrantT   `koanf:"QDRANT"`
	S3       *S3T       `koanf:"S3"`

	LLMEmbedding *LLMEmbeddingT `koanf:"LLM_EMBEDDING"`
	LLMChat      *LLMChatT      `koanf:"LLM_CHAT"`
}

func (e *EnvType) Validate() {
	e.Info.validate()
	e.Cors.validate()
	e.RateLimiter.validate()
	e.Otel.validate()
	e.WorkerPool.validate()
}

func (e *EnvType) LoadDefault() {
	e.RateLimiter.loadDefault()
	e.WorkerPool.loadDefault()
}
