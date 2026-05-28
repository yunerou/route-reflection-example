package openaichatadapter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	llmchat "github.com/yunerou/niarb/provider/llm-chat"
	"github.com/yunerou/niarb/shared/aerror"
)

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatResponseFormat struct {
	Type string `json:"type"`
}

type chatRequest struct {
	Model          string              `json:"model"`
	Messages       []chatMessage       `json:"messages"`
	ResponseFormat *chatResponseFormat `json:"response_format,omitempty"`
	Temperature    *float32            `json:"temperature,omitempty"`
	MaxTokens      *int                `json:"max_tokens,omitempty"`
}

type chatChoice struct {
	Message chatMessage `json:"message"`
}

type chatResponse struct {
	Choices []chatChoice `json:"choices"`
}

func (c *client) Complete(ctx context.Context, in llmchat.CompleteRequest) (*llmchat.CompleteResponse, aerror.AError) {
	reqBody := chatRequest{
		Model: c.model,
		Messages: []chatMessage{
			{Role: "system", Content: in.System},
			{Role: "user", Content: in.User},
		},
		Temperature: in.Temperature,
		MaxTokens:   in.MaxTokens,
	}
	if in.ResponseFormat != nil {
		reqBody.ResponseFormat = &chatResponseFormat{Type: in.ResponseFormat.Type}
	}

	bodyBytes, jsonErr := json.Marshal(reqBody)
	if jsonErr != nil {
		return nil, aerror.New(ctx, aerror.ErrUnexpectedNetwork, fmt.Errorf("llm-chat: marshal request: %w", jsonErr))
	}

	req, reqErr := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(bodyBytes))
	if reqErr != nil {
		return nil, aerror.New(ctx, aerror.ErrUnexpectedNetwork, fmt.Errorf("llm-chat: build request: %w", reqErr))
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != nil {
		req.Header.Set("Authorization", "Bearer "+*c.apiKey)
	}

	resp, httpErr := c.httpClient.Do(req)
	if httpErr != nil {
		return nil, aerror.New(ctx, aerror.ErrUnexpectedNetwork, fmt.Errorf("llm-chat: request failed: %w", httpErr))
	}
	defer resp.Body.Close()

	respBody, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, aerror.New(ctx, aerror.ErrUnexpectedNetwork, fmt.Errorf("llm-chat: read response: %w", readErr))
	}

	if resp.StatusCode != http.StatusOK {
		return nil, aerror.New(ctx, aerror.ErrUnexpectedNetwork, fmt.Errorf("llm-chat: unexpected status %d: %s", resp.StatusCode, string(respBody)))
	}

	var decoded chatResponse
	if decodeErr := json.Unmarshal(respBody, &decoded); decodeErr != nil {
		return nil, aerror.New(ctx, aerror.ErrUnexpectedNetwork, fmt.Errorf("llm-chat: decode response: %w", decodeErr))
	}
	if len(decoded.Choices) == 0 {
		return nil, aerror.New(ctx, aerror.ErrUnexpectedNetwork, fmt.Errorf("llm-chat: response has no choices"))
	}

	return &llmchat.CompleteResponse{Content: decoded.Choices[0].Message.Content}, nil
}
