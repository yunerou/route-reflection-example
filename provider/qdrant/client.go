package qdrant

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	configprovider "github.com/yunerou/niarb/provider/config-provider"
	exhttp "github.com/yunerou/niarb/shared/helper/ex-http"
)

type client struct {
	httpClient *http.Client
	baseURL    string
	apiKey     *string
}

type apiEnvelope struct {
	Status string          `json:"status"`
	Result json.RawMessage `json:"result"`
}

func New(cfg *configprovider.QdrantT) QdrantProvider {
	if cfg == nil {
		panic("qdrant: config must not be nil")
	}
	if cfg.URL == "" {
		panic("qdrant: URL must not be empty")
	}

	httpClient := exhttp.NewHTTPClient()
	if cfg.Timeout > 0 {
		httpClient.Timeout = cfg.Timeout
	} else {
		httpClient.Timeout = 10 * time.Second
	}

	return &client{
		httpClient: httpClient,
		baseURL:    strings.TrimRight(cfg.URL, "/"),
		apiKey:     cfg.APIKey,
	}
}

func (c *client) doJSON(
	ctx context.Context,
	method string,
	path string,
	query url.Values,
	reqBody any,
	result any,
) error {
	endpoint := c.baseURL + path
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}

	var bodyReader io.Reader
	if reqBody != nil {
		bodyBytes, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("qdrant: failed to marshal request %s %s: %w", method, path, err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint, bodyReader)
	if err != nil {
		return fmt.Errorf("qdrant: failed to create request %s %s: %w", method, path, err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != nil {
		req.Header.Set("api-key", *c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("qdrant: request failed %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("qdrant: failed to read response %s %s: %w", method, path, err)
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("qdrant: unexpected status %d for %s %s: %s", resp.StatusCode, method, path, string(respBody))
	}

	if result == nil {
		return nil
	}

	var envelope apiEnvelope
	if err := json.Unmarshal(respBody, &envelope); err != nil {
		return fmt.Errorf("qdrant: failed to decode response envelope %s %s: %w", method, path, err)
	}
	if envelope.Status != "" && envelope.Status != "ok" {
		return fmt.Errorf("qdrant: unexpected response status %q for %s %s", envelope.Status, method, path)
	}
	if len(envelope.Result) == 0 || string(envelope.Result) == "null" {
		return nil
	}
	if err := json.Unmarshal(envelope.Result, result); err != nil {
		return fmt.Errorf("qdrant: failed to decode response result %s %s: %w", method, path, err)
	}
	return nil
}
