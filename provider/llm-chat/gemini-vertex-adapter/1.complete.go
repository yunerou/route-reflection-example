package geminivertexchatadapter

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

type genPart struct {
	Text string `json:"text"`
}

type genContent struct {
	Role  string    `json:"role,omitempty"`
	Parts []genPart `json:"parts"`
}

type genSafetySetting struct {
	Category  string `json:"category"`
	Threshold string `json:"threshold"`
}

type genGenerationConfig struct {
	Temperature      *float32 `json:"temperature,omitempty"`
	MaxOutputTokens  *int     `json:"maxOutputTokens,omitempty"`
	ResponseMimeType string   `json:"responseMimeType,omitempty"`
}

type genRequest struct {
	Contents          []genContent        `json:"contents"`
	SystemInstruction *genContent         `json:"systemInstruction,omitempty"`
	GenerationConfig  genGenerationConfig `json:"generationConfig"`
	SafetySettings    []genSafetySetting  `json:"safetySettings"`
}

type genCandidate struct {
	Content      genContent `json:"content"`
	FinishReason string     `json:"finishReason"`
}

type genPromptFeedback struct {
	BlockReason string `json:"blockReason"`
}

type genResponse struct {
	Candidates     []genCandidate     `json:"candidates"`
	PromptFeedback *genPromptFeedback `json:"promptFeedback"`
}

var safetyBlockNone = []genSafetySetting{
	{Category: "HARM_CATEGORY_HARASSMENT", Threshold: "BLOCK_NONE"},
	{Category: "HARM_CATEGORY_HATE_SPEECH", Threshold: "BLOCK_NONE"},
	{Category: "HARM_CATEGORY_SEXUALLY_EXPLICIT", Threshold: "BLOCK_NONE"},
	{Category: "HARM_CATEGORY_DANGEROUS_CONTENT", Threshold: "BLOCK_NONE"},
}

func (c *client) Complete(ctx context.Context, in llmchat.CompleteRequest) (*llmchat.CompleteResponse, aerror.AError) {
	const label = "gemini-vertex-chat"

	body := genRequest{
		Contents: []genContent{
			{Role: "user", Parts: []genPart{{Text: in.User}}},
		},
		GenerationConfig: genGenerationConfig{
			Temperature:     in.Temperature,
			MaxOutputTokens: in.MaxTokens,
		},
		SafetySettings: safetyBlockNone,
	}
	if in.System != "" {
		body.SystemInstruction = &genContent{Parts: []genPart{{Text: in.System}}}
	}
	if in.ResponseFormat != nil && in.ResponseFormat.Type == "json_object" {
		body.GenerationConfig.ResponseMimeType = "application/json"
	}

	bodyBytes, jsonErr := json.Marshal(body)
	if jsonErr != nil {
		return nil, aerror.New(ctx, aerror.ErrUnexpectedNetwork, fmt.Errorf("%s: marshal request: %w", label, jsonErr))
	}

	url := fmt.Sprintf("%s/v1/projects/%s/locations/%s/publishers/google/models/%s:generateContent",
		c.baseURL, c.projectID, c.location, c.model)

	req, reqErr := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if reqErr != nil {
		return nil, aerror.New(ctx, aerror.ErrUnexpectedNetwork, fmt.Errorf("%s: build request: %w", label, reqErr))
	}

	tok, tokErr := c.tokenSource.Token()
	if tokErr != nil {
		return nil, aerror.New(ctx, aerror.ErrUnexpectedNetwork, fmt.Errorf("%s: obtain token: %w", label, tokErr))
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok.AccessToken)

	resp, httpErr := c.httpClient.Do(req)
	if httpErr != nil {
		return nil, aerror.New(ctx, aerror.ErrUnexpectedNetwork, fmt.Errorf("%s: request failed: %w", label, httpErr))
	}
	defer resp.Body.Close()

	respBody, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, aerror.New(ctx, aerror.ErrUnexpectedNetwork, fmt.Errorf("%s: read response: %w", label, readErr))
	}

	if resp.StatusCode != http.StatusOK {
		return nil, aerror.New(ctx, aerror.ErrUnexpectedNetwork, fmt.Errorf("%s: unexpected status %d: %s", label, resp.StatusCode, string(respBody)))
	}

	var decoded genResponse
	if decErr := json.Unmarshal(respBody, &decoded); decErr != nil {
		return nil, aerror.New(ctx, aerror.ErrUnexpectedNetwork, fmt.Errorf("%s: decode response: %w", label, decErr))
	}
	if decoded.PromptFeedback != nil && decoded.PromptFeedback.BlockReason != "" {
		return nil, aerror.New(ctx, aerror.ErrUnexpectedNetwork, fmt.Errorf("%s: prompt blocked: %s", label, decoded.PromptFeedback.BlockReason))
	}
	if len(decoded.Candidates) == 0 {
		return nil, aerror.New(ctx, aerror.ErrUnexpectedNetwork, fmt.Errorf("%s: no candidates", label))
	}
	cand := decoded.Candidates[0]
	if len(cand.Content.Parts) == 0 {
		return nil, aerror.New(ctx, aerror.ErrUnexpectedNetwork, fmt.Errorf("%s: empty content (finishReason=%s)", label, cand.FinishReason))
	}
	return &llmchat.CompleteResponse{Content: cand.Content.Parts[0].Text}, nil
}
