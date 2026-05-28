package openaiadapter

import (
	"net/http"
	"strings"

	configprovider "github.com/yunerou/niarb/provider/config-provider"
	llmembedding "github.com/yunerou/niarb/provider/llm-embedding"
	exhttp "github.com/yunerou/niarb/shared/helper/ex-http"
)

type client struct {
	httpClient *http.Client
	baseURL    string
	model      string
	apiKey     *string
	dimension  int
}

func New(cfg *configprovider.LLMEmbeddingT) llmembedding.Adapter {
	if cfg == nil {
		panic("llm-embedding: config must not be nil")
	}
	if cfg.BaseURL == "" {
		panic("llm-embedding: BASE_URL must not be empty")
	}
	if cfg.Model == "" {
		panic("llm-embedding: MODEL must not be empty")
	}
	if cfg.Dimension <= 0 {
		panic("llm-embedding: DIMENSION must be greater than 0")
	}

	return &client{
		httpClient: exhttp.NewHTTPClient(),
		baseURL:    strings.TrimRight(cfg.BaseURL, "/"),
		model:      cfg.Model,
		apiKey:     cfg.APIKey,
		dimension:  cfg.Dimension,
	}
}
