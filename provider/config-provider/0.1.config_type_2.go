package configprovider

import (
	"time"

	"github.com/samber/lo"
)

type InfoT struct {
	Env     string `koanf:"ENV"`
	PodName string `koanf:"POD_NAME"`
}

func (r *InfoT) validate() {
	if r.Env == "" {
		panic("info env must not be empty")
	}
}

type ExtractHeaderT struct {
	TraceIDHeader string `koanf:"TRACE_ID_HEADER"`
	IpHeader      string `koanf:"IP_HEADER"`
	AuthHeader    string `koanf:"AUTH_HEADER"`
}

type CorsT struct {
	Enabled        bool     `koanf:"ENABLED"`
	ExactlyOrigins []string `koanf:"EXACTLY_ORIGINS"`
	RegexOrigins   []string `koanf:"REGEX_ORIGINS"`
}

func (c *CorsT) validate() {
	if c.Enabled {
		if len(c.ExactlyOrigins) == 0 && len(c.RegexOrigins) == 0 {
			panic("cors config: either exactly origins or regex origins must be provided when cors is enabled")
		}
	}
}

type WorkerPoolT struct {
	Buffer         int `koanf:"BUFFER"`
	UpperThreshold int `koanf:"UPPER_THRESHOLD"`
	LowerThreshold int `koanf:"LOWER_THRESHOLD"`
}

func (w *WorkerPoolT) validate() {
	if w.Buffer <= 0 {
		panic("worker pool buffer must be greater than 0")
	}
	if w.UpperThreshold <= 0 {
		panic("worker pool upper threshold must be greater than 0")
	}
	if w.LowerThreshold < 0 {
		panic("worker pool lower threshold must be greater than or equal to 0")
	}
	if w.UpperThreshold <= w.LowerThreshold {
		panic("worker pool upper threshold must be greater than lower threshold")
	}
}

func (w *WorkerPoolT) loadDefault() {
	if w.Buffer == 0 {
		w.Buffer = 64
	}
	if w.UpperThreshold == 0 {
		w.UpperThreshold = 48
	}
	if w.LowerThreshold == 0 {
		w.LowerThreshold = 16
	}
}

type RateLimiterT struct {
	UserRate  int `koanf:"USER_RATE"`
	UserBurst int `koanf:"USER_BURST"`

	AnonymousRate  int `koanf:"ANONYMOUS_RATE"`
	AnonymousBurst int `koanf:"ANONYMOUS_BURST"`
}

func (r *RateLimiterT) validate() {
	if r.UserRate <= 0 {
		panic("rate limiter user rate must be greater than 0")
	}
	if r.UserBurst < r.UserRate {
		panic("rate limiter user burst must be greater than or equal to user rate")
	}
	if r.AnonymousRate <= 0 {
		panic("rate limiter anonymous rate must be greater than 0")
	}
	if r.AnonymousBurst < r.AnonymousRate {
		panic("rate limiter anonymous burst must be greater than or equal to anonymous rate")
	}
}

func (r *RateLimiterT) loadDefault() {
	if r.UserRate == 0 {
		r.UserRate = 30
	}
	if r.UserBurst == 0 {
		r.UserBurst = 60
	}
	if r.AnonymousRate == 0 {
		r.AnonymousRate = 5
	}
	if r.AnonymousBurst == 0 {
		r.AnonymousBurst = 10
	}
}

type ValkeyT struct {
	PrimaryAddress string `koanf:"PRIMARY_ADDRESS"`
	ReplicaAddress string `koanf:"REPLICA_ADDRESS"`
	Password       string `koanf:"PASSWORD"`
	DatabaseIdx    int    `koanf:"DB_INDEX"`
}

// PostgresT contains PostgreSQL connection configuration
type PostgresT struct {
	CreateTables bool   `koanf:"CREATE_TABLES"`
	Host         string `koanf:"HOST"`
	Port         int    `koanf:"PORT"`
	DBName       string `koanf:"DB_NAME"`
	User         string `koanf:"USER"`
	Password     string `koanf:"PASSWORD"`
	SSLMode      string `koanf:"SSL_MODE"` // allow value: disable, require
	MaxConns     int32  `koanf:"MAX_CONNS"`
	MinConns     int32  `koanf:"MIN_CONNS"`
}

// QdrantT contains Qdrant connection configuration.
type QdrantT struct {
	URL     string        `koanf:"URL"`
	APIKey  *string       `koanf:"API_KEY"`
	Timeout time.Duration `koanf:"TIMEOUT"`
}

// OtelT contains OpenTelemetry configuration
type OtelT struct {
	Enabled bool `koanf:"ENABLED"`
	// Debug = -4, Info = 0, Warn = 4, Error = 8
	LogLevel            int    `koanf:"LOG_LEVEL"`
	AutoInstrumentation bool   `koanf:"AUTO_INSTRUMENTATION"`
	ExporterType        string `koanf:"EXPORTER_TYPE"`
	FilePath            string `koanf:"FILE_PATH"`
	CollectorEndpoint   string `koanf:"COLLECTOR_ENDPOINT"`
	CollectorInsecure   bool   `koanf:"COLLECTOR_INSECURE"`
}

// Validate checks if the OtelT configuration is valid
func (o *OtelT) validate() {
	if o.Enabled {
		allowExporterType := []string{"discard", "stdout", "file", "otlp-grpc", "otlp-http"}
		if !lo.Contains(allowExporterType, o.ExporterType) {
			panic("OTEL.EXPORTER_TYPE must be one of [discard, stdout, file, otlp-grpc, otlp-http]")
		}
		if o.ExporterType == "file" && o.FilePath == "" {
			panic("OTEL.FILE_PATH is required when OTEL.EXPORTER_TYPE is file")
		}
		if o.ExporterType == "otlp-grpc" && o.CollectorEndpoint == "" {
			panic("OTEL.COLLECTOR_ENDPOINT is required when OTEL.EXPORTER_TYPE is otlp-grpc")
		}
		if o.ExporterType == "otlp-http" && o.CollectorEndpoint == "" {
			panic("OTEL.COLLECTOR_ENDPOINT is required when OTEL.EXPORTER_TYPE is otlp-http")
		}
	}
}

// S3T contains S3-compatible object storage configuration
type S3T struct {
	Region          string  `koanf:"REGION"`
	Bucket          string  `koanf:"BUCKET"`
	Endpoint        *string `koanf:"ENDPOINT"`         // MinIO / rustfs / custom endpoint
	ForcePathStyle  bool    `koanf:"FORCE_PATH_STYLE"` // true for MinIO/rustfs
	StaticAccessKey *string `koanf:"STATIC_ACCESS_KEY"`
	StaticSecretKey *string `koanf:"STATIC_SECRET_KEY"`
}

// LLMEmbeddingT contains LLM embedding service configuration
type LLMEmbeddingT struct {
	// Provider selector. "" or "openai" → existing openai-compat adapter (Ollama, vLLM, ...).
	// Other accepted values: "gemini-aistudio", "gemini-vertex".
	Provider string `koanf:"PROVIDER"`

	// Common fields. BaseURL semantics:
	//   - openai: required (e.g. http://localhost:11434/v1)
	//   - gemini-aistudio: optional override; default https://generativelanguage.googleapis.com/v1beta
	//   - gemini-vertex: ignored (URL derived from Location/ProjectID/Model)
	BaseURL   string  `koanf:"BASE_URL"`
	Model     string  `koanf:"MODEL"`
	APIKey    *string `koanf:"API_KEY"`
	Dimension int     `koanf:"DIMENSION"`

	// Gemini-only (both aistudio + vertex). One of:
	// RETRIEVAL_DOCUMENT | RETRIEVAL_QUERY | SEMANTIC_SIMILARITY | CLASSIFICATION | CLUSTERING.
	// Nil → omitted from request (Gemini uses default).
	TaskType *string `koanf:"TASK_TYPE"`

	// Vertex-only.
	ProjectID       *string `koanf:"PROJECT_ID"`
	Location        *string `koanf:"LOCATION"`         // e.g. "us-central1"
	CredentialsFile *string `koanf:"CREDENTIALS_FILE"` // optional path to SA JSON; nil → ADC
}

// LLMChatT contains LLM chat completion service configuration.
// Used for tasks like document classification via OpenAI-compatible APIs.
type LLMChatT struct {
	// Provider selector. "" or "openai" → existing openai-compat adapter (Ollama, vLLM, ...).
	// Other accepted values: "gemini-aistudio", "gemini-vertex".
	Provider string `koanf:"PROVIDER"`

	// Common fields. BaseURL semantics:
	//   - openai: required (e.g. http://localhost:11434/v1)
	//   - gemini-aistudio: optional override; default https://generativelanguage.googleapis.com/v1beta
	//   - gemini-vertex: ignored (URL derived from Location/ProjectID/Model)
	BaseURL string  `koanf:"BASE_URL"`
	Model   string  `koanf:"MODEL"`
	APIKey  *string `koanf:"API_KEY"`

	// Vertex-only.
	ProjectID       *string `koanf:"PROJECT_ID"`
	Location        *string `koanf:"LOCATION"`         // e.g. "us-central1"
	CredentialsFile *string `koanf:"CREDENTIALS_FILE"` // optional path to SA JSON; nil → ADC
}
