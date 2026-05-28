package geminivertexadapter

import (
	"context"
	"fmt"
	"net/http"

	"golang.org/x/oauth2"

	configprovider "github.com/yunerou/niarb/provider/config-provider"
	llmembedding "github.com/yunerou/niarb/provider/llm-embedding"
	exhttp "github.com/yunerou/niarb/shared/helper/ex-http"
)

type client struct {
	httpClient  *http.Client
	baseURL     string // https://{location}-aiplatform.googleapis.com (overridable in tests)
	projectID   string
	location    string
	model       string
	dimension   int
	taskType    *string
	tokenSource oauth2.TokenSource
}

func New(cfg *configprovider.LLMEmbeddingT) llmembedding.Adapter {
	if cfg == nil {
		panic("gemini-vertex: config must not be nil")
	}
	if cfg.Model == "" {
		panic("gemini-vertex: MODEL must not be empty")
	}
	if cfg.ProjectID == nil || *cfg.ProjectID == "" {
		panic("gemini-vertex: PROJECT_ID must not be empty")
	}
	if cfg.Location == nil || *cfg.Location == "" {
		panic("gemini-vertex: LOCATION must not be empty")
	}
	if cfg.Dimension <= 0 {
		panic("gemini-vertex: DIMENSION must be greater than 0")
	}

	ts, err := newTokenSource(context.Background(), cfg.CredentialsFile)
	if err != nil {
		panic(fmt.Sprintf("gemini-vertex: init token source: %v", err))
	}

	return newClientWithDeps(
		cfg.Model,
		*cfg.ProjectID,
		*cfg.Location,
		cfg.Dimension,
		cfg.TaskType,
		ts,
		fmt.Sprintf("https://%s-aiplatform.googleapis.com", *cfg.Location),
	)
}

// newClientWithDeps is the lower-level constructor used by New and by tests
// (tests inject a httptest server URL + a static token source).
func newClientWithDeps(model, projectID, location string, dimension int, taskType *string, ts oauth2.TokenSource, baseURL string) llmembedding.Adapter {
	return &client{
		httpClient:  exhttp.NewHTTPClient(),
		baseURL:     baseURL,
		projectID:   projectID,
		location:    location,
		model:       model,
		dimension:   dimension,
		taskType:    taskType,
		tokenSource: ts,
	}
}
