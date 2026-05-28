package openaiadapter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/yunerou/niarb/shared/aerror"
)

type embeddingRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type embeddingDataItem struct {
	Embedding []float32 `json:"embedding"`
	Index     int       `json:"index"`
}

type embeddingResponse struct {
	Data []embeddingDataItem `json:"data"`
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
		return nil, aerror.New(ctx, aerror.ErrInvalidInput, fmt.Errorf("llm-embedding: texts must not be empty"))
	}

	reqBody := embeddingRequest{
		Model: c.model,
		Input: texts,
	}

	bodyBytes, jsonErr := json.Marshal(reqBody)
	if jsonErr != nil {
		return nil, aerror.New(ctx, aerror.ErrUnexpectedNetwork, fmt.Errorf("llm-embedding: failed to marshal request: %w", jsonErr))
	}

	req, reqErr := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/embeddings", bytes.NewReader(bodyBytes))
	if reqErr != nil {
		return nil, aerror.New(ctx, aerror.ErrUnexpectedNetwork, fmt.Errorf("llm-embedding: failed to create request: %w", reqErr))
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != nil {
		req.Header.Set("Authorization", "Bearer "+*c.apiKey)
	}

	resp, httpErr := c.httpClient.Do(req)
	if httpErr != nil {
		return nil, aerror.New(ctx, aerror.ErrUnexpectedNetwork, fmt.Errorf("llm-embedding: request failed: %w", httpErr))
	}
	defer resp.Body.Close()

	respBody, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, aerror.New(ctx, aerror.ErrUnexpectedNetwork, fmt.Errorf("llm-embedding: failed to read response: %w", readErr))
	}

	if resp.StatusCode != http.StatusOK {
		return nil, aerror.New(ctx, aerror.ErrUnexpectedNetwork, fmt.Errorf("llm-embedding: unexpected status %d: %s", resp.StatusCode, string(respBody)))
	}

	var embResp embeddingResponse
	if decodeErr := json.Unmarshal(respBody, &embResp); decodeErr != nil {
		return nil, aerror.New(ctx, aerror.ErrUnexpectedNetwork, fmt.Errorf("llm-embedding: failed to decode response: %w", decodeErr))
	}

	if len(embResp.Data) != len(texts) {
		return nil, aerror.New(ctx, aerror.ErrUnexpectedNetwork, fmt.Errorf("llm-embedding: expected %d embeddings, got %d", len(texts), len(embResp.Data)))
	}

	results := make([][]float32, len(texts))
	for _, item := range embResp.Data {
		results[item.Index] = item.Embedding
	}

	return results, nil
}

func (c *client) Dimension() int {
	return c.dimension
}
