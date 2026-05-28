package openaichatadapter

import (
	"net/http"
	"strings"

	configprovider "github.com/yunerou/niarb/provider/config-provider"
	llmchat "github.com/yunerou/niarb/provider/llm-chat"
	exhttp "github.com/yunerou/niarb/shared/helper/ex-http"
)

type client struct {
	httpClient *http.Client
	baseURL    string
	model      string
	apiKey     *string
}

// New constructs an OpenAI-compatible chat Adapter.
// Panics if required config is missing — matches embedding adapter behavior.
func New(cfg *configprovider.LLMChatT) llmchat.Adapter {
	if cfg == nil {
		panic("llm-chat: config must not be nil")
	}
	if cfg.BaseURL == "" {
		panic("llm-chat: BASE_URL must not be empty")
	}
	if cfg.Model == "" {
		panic("llm-chat: MODEL must not be empty")
	}
	return &client{
		httpClient: exhttp.NewHTTPClient(),
		baseURL:    strings.TrimRight(cfg.BaseURL, "/"),
		model:      cfg.Model,
		apiKey:     cfg.APIKey,
	}
}
