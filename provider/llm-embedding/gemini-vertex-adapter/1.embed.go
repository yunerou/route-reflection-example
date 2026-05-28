package geminivertexadapter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/yunerou/niarb/shared/aerror"
)

const vertexMaxBatch = 250

type vertexInstance struct {
	Content  string  `json:"content"`
	TaskType *string `json:"task_type,omitempty"`
}

type vertexParameters struct {
	OutputDimensionality int `json:"outputDimensionality"`
}

type vertexRequest struct {
	Instances  []vertexInstance `json:"instances"`
	Parameters vertexParameters `json:"parameters"`
}

type vertexEmbeddings struct {
	Values []float32 `json:"values"`
}

type vertexPrediction struct {
	Embeddings vertexEmbeddings `json:"embeddings"`
}

type vertexResponse struct {
	Predictions []vertexPrediction `json:"predictions"`
}

func (c *client) Embed(ctx context.Context, text string) ([]float32, aerror.AError) {
	results, err := c.EmbedBatch(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	return results[0], nil
}

func (c *client) EmbedBatch(ctx context.Context, texts []string) ([][]float32, aerror.AError) {
	if len(texts) == 0 {
		return nil, aerror.New(ctx, aerror.ErrInvalidInput, fmt.Errorf("gemini-vertex: texts must not be empty"))
	}
	if len(texts) > vertexMaxBatch {
		return nil, aerror.New(ctx, aerror.ErrInvalidInput, fmt.Errorf("gemini-vertex: batch size %d exceeds Vertex limit %d; caller must chunk", len(texts), vertexMaxBatch))
	}

	instances := make([]vertexInstance, len(texts))
	for i, t := range texts {
		instances[i] = vertexInstance{Content: t, TaskType: c.taskType}
	}
	body := vertexRequest{
		Instances:  instances,
		Parameters: vertexParameters{OutputDimensionality: c.dimension},
	}

	bodyBytes, jsonErr := json.Marshal(body)
	if jsonErr != nil {
		return nil, aerror.New(ctx, aerror.ErrUnexpectedNetwork, fmt.Errorf("gemini-vertex: failed to marshal request: %w", jsonErr))
	}

	url := fmt.Sprintf("%s/v1/projects/%s/locations/%s/publishers/google/models/%s:predict",
		c.baseURL, c.projectID, c.location, c.model)

	req, reqErr := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if reqErr != nil {
		return nil, aerror.New(ctx, aerror.ErrUnexpectedNetwork, fmt.Errorf("gemini-vertex: failed to create request: %w", reqErr))
	}

	tok, tokErr := c.tokenSource.Token()
	if tokErr != nil {
		return nil, aerror.New(ctx, aerror.ErrUnexpectedNetwork, fmt.Errorf("gemini-vertex: failed to obtain token: %w", tokErr))
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok.AccessToken)

	resp, httpErr := c.httpClient.Do(req)
	if httpErr != nil {
		return nil, aerror.New(ctx, aerror.ErrUnexpectedNetwork, fmt.Errorf("gemini-vertex: request failed: %w", httpErr))
	}
	defer resp.Body.Close()

	respBody, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, aerror.New(ctx, aerror.ErrUnexpectedNetwork, fmt.Errorf("gemini-vertex: failed to read response: %w", readErr))
	}

	if resp.StatusCode != http.StatusOK {
		return nil, aerror.New(ctx, aerror.ErrUnexpectedNetwork, fmt.Errorf("gemini-vertex: unexpected status %d: %s", resp.StatusCode, string(respBody)))
	}

	var parsed vertexResponse
	if decErr := json.Unmarshal(respBody, &parsed); decErr != nil {
		return nil, aerror.New(ctx, aerror.ErrUnexpectedNetwork, fmt.Errorf("gemini-vertex: failed to decode response: %w", decErr))
	}

	if len(parsed.Predictions) != len(texts) {
		return nil, aerror.New(ctx, aerror.ErrUnexpectedNetwork, fmt.Errorf("gemini-vertex: expected %d predictions, got %d", len(texts), len(parsed.Predictions)))
	}

	results := make([][]float32, len(texts))
	for i, p := range parsed.Predictions {
		results[i] = p.Embeddings.Values
	}
	return results, nil
}

func (c *client) Dimension() int {
	return c.dimension
}
