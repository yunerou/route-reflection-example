package geminiaistudiochatadapter

import (
	"net/http"
	"strings"

	configprovider "github.com/yunerou/niarb/provider/config-provider"
	llmchat "github.com/yunerou/niarb/provider/llm-chat"
	exhttp "github.com/yunerou/niarb/shared/helper/ex-http"
)

const defaultBaseURL = "https://generativelanguage.googleapis.com/v1beta"

type client struct {
	httpClient *http.Client
	baseURL    string
	model      string
	apiKey     string
}

// New constructs a Google AI Studio chat Adapter.
// Panics if required config is missing — matches sibling adapters.
func New(cfg *configprovider.LLMChatT) llmchat.Adapter {
	if cfg == nil {
		panic("gemini-aistudio-chat: config must not be nil")
	}
	if cfg.Model == "" {
		panic("gemini-aistudio-chat: MODEL must not be empty")
	}
	if cfg.APIKey == nil || *cfg.APIKey == "" {
		panic("gemini-aistudio-chat: API_KEY must not be empty")
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
	}
}
