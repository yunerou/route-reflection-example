package geminivertexchatadapter

import (
	"context"
	"fmt"
	"net/http"

	"golang.org/x/oauth2"

	configprovider "github.com/yunerou/niarb/provider/config-provider"
	llmchat "github.com/yunerou/niarb/provider/llm-chat"
	exhttp "github.com/yunerou/niarb/shared/helper/ex-http"
)

type client struct {
	httpClient  *http.Client
	baseURL     string // https://{location}-aiplatform.googleapis.com (overridable in tests)
	projectID   string
	location    string
	model       string
	tokenSource oauth2.TokenSource
}

// New constructs a Vertex AI chat Adapter. Panics if required config is missing.
func New(cfg *configprovider.LLMChatT) llmchat.Adapter {
	if cfg == nil {
		panic("gemini-vertex-chat: config must not be nil")
	}
	if cfg.Model == "" {
		panic("gemini-vertex-chat: MODEL must not be empty")
	}
	if cfg.ProjectID == nil || *cfg.ProjectID == "" {
		panic("gemini-vertex-chat: PROJECT_ID must not be empty")
	}
	if cfg.Location == nil || *cfg.Location == "" {
		panic("gemini-vertex-chat: LOCATION must not be empty")
	}

	ts, err := newTokenSource(context.Background(), cfg.CredentialsFile)
	if err != nil {
		panic(fmt.Sprintf("gemini-vertex-chat: init token source: %v", err))
	}

	return newClientWithDeps(
		cfg.Model,
		*cfg.ProjectID,
		*cfg.Location,
		ts,
		fmt.Sprintf("https://%s-aiplatform.googleapis.com", *cfg.Location),
	)
}

// newClientWithDeps is the lower-level constructor used by New and by tests
// (tests inject an httptest server URL + a static token source).
func newClientWithDeps(model, projectID, location string, ts oauth2.TokenSource, baseURL string) llmchat.Adapter {
	return &client{
		httpClient:  exhttp.NewHTTPClient(),
		baseURL:     baseURL,
		projectID:   projectID,
		location:    location,
		model:       model,
		tokenSource: ts,
	}
}
