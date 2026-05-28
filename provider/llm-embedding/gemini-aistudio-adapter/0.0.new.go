package geminiaistudioadapter

import (
	"net/http"
	"strings"

	configprovider "github.com/yunerou/niarb/provider/config-provider"
	llmembedding "github.com/yunerou/niarb/provider/llm-embedding"
	exhttp "github.com/yunerou/niarb/shared/helper/ex-http"
)

const defaultBaseURL = "https://generativelanguage.googleapis.com/v1beta"

type client struct {
	httpClient *http.Client
	baseURL    string
	model      string
	apiKey     string
	dimension  int
	taskType   *string
}

func New(cfg *configprovider.LLMEmbeddingT) llmembedding.Adapter {
	if cfg == nil {
		panic("gemini-aistudio: config must not be nil")
	}
	if cfg.Model == "" {
		panic("gemini-aistudio: MODEL must not be empty")
	}
	if cfg.APIKey == nil || *cfg.APIKey == "" {
		panic("gemini-aistudio: API_KEY must not be empty")
	}
	if cfg.Dimension <= 0 {
		panic("gemini-aistudio: DIMENSION must be greater than 0")
	}

	baseURL := defaultBaseURL
	if cfg.BaseURL != "" {
		baseURL = strings.TrimRight(cfg.BaseURL, "/")
	}

	return &client{
		httpClient: exhttp.NewHTTPClient(),
		baseURL:    baseURL,
		model:      cfg.Model,
		apiKey:     *cfg.APIKey,
		dimension:  cfg.Dimension,
		taskType:   cfg.TaskType,
	}
}
