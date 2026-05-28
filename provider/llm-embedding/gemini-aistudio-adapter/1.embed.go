package geminiaistudioadapter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/yunerou/niarb/shared/aerror"
)

type contentPart struct {
	Text string `json:"text"`
}

type embedContent struct {
	Parts []contentPart `json:"parts"`
}

type embedRequest struct {
	Model                string       `json:"model"`
	Content              embedContent `json:"content"`
	TaskType             *string      `json:"taskType,omitempty"`
	OutputDimensionality int          `json:"outputDimensionality"`
}

type batchEmbedRequest struct {
	Requests []embedRequest `json:"requests"`
}

type embeddingValues struct {
	Values []float32 `json:"values"`
}

type batchEmbedResponse struct {
	Embeddings []embeddingValues `json:"embeddings"`
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
		return nil, aerror.New(ctx, aerror.ErrInvalidInput, fmt.Errorf("gemini-aistudio: texts must not be empty"))
	}

	modelField := "models/" + c.model
	reqs := make([]embedRequest, len(texts))
	for i, t := range texts {
		reqs[i] = embedRequest{
			Model:                modelField,
			Content:              embedContent{Parts: []contentPart{{Text: t}}},
			TaskType:             c.taskType,
			OutputDimensionality: c.dimension,
		}
	}

	bodyBytes, jsonErr := json.Marshal(batchEmbedRequest{Requests: reqs})
	if jsonErr != nil {
		return nil, aerror.New(ctx, aerror.ErrUnexpectedNetwork, fmt.Errorf("gemini-aistudio: failed to marshal request: %w", jsonErr))
	}

	url := c.baseURL + "/models/" + c.model + ":batchEmbedContents"
	req, reqErr := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if reqErr != nil {
		return nil, aerror.New(ctx, aerror.ErrUnexpectedNetwork, fmt.Errorf("gemini-aistudio: failed to create request: %w", reqErr))
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", c.apiKey)

	resp, httpErr := c.httpClient.Do(req)
	if httpErr != nil {
		return nil, aerror.New(ctx, aerror.ErrUnexpectedNetwork, fmt.Errorf("gemini-aistudio: request failed: %w", httpErr))
	}
	defer resp.Body.Close()

	respBody, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, aerror.New(ctx, aerror.ErrUnexpectedNetwork, fmt.Errorf("gemini-aistudio: failed to read response: %w", readErr))
	}

	if resp.StatusCode != http.StatusOK {
		return nil, aerror.New(ctx, aerror.ErrUnexpectedNetwork, fmt.Errorf("gemini-aistudio: unexpected status %d: %s", resp.StatusCode, string(respBody)))
	}

	var parsed batchEmbedResponse
	if decErr := json.Unmarshal(respBody, &parsed); decErr != nil {
		return nil, aerror.New(ctx, aerror.ErrUnexpectedNetwork, fmt.Errorf("gemini-aistudio: failed to decode response: %w", decErr))
	}

	if len(parsed.Embeddings) != len(texts) {
		return nil, aerror.New(ctx, aerror.ErrUnexpectedNetwork, fmt.Errorf("gemini-aistudio: expected %d embeddings, got %d", len(texts), len(parsed.Embeddings)))
	}

	results := make([][]float32, len(texts))
	for i, e := range parsed.Embeddings {
		results[i] = e.Values
	}
	return results, nil
}

func (c *client) Dimension() int {
	return c.dimension
}
