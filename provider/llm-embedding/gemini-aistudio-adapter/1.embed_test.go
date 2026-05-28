package geminiaistudioadapter

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	configprovider "github.com/yunerou/niarb/provider/config-provider"
	llmembedding "github.com/yunerou/niarb/provider/llm-embedding"
)

func ptrStr(s string) *string { return &s }

func newTestClient(t *testing.T, handler http.HandlerFunc, taskType *string) (llmembedding.Adapter, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	apiKey := "test-key"
	adapter := New(&configprovider.LLMEmbeddingT{
		BaseURL:   srv.URL,
		Model:     "text-embedding-004",
		APIKey:    &apiKey,
		Dimension: 768,
		TaskType:  taskType,
	})
	return adapter, srv
}

func okHandler(t *testing.T, batchSize int) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		embs := make([]map[string]any, batchSize)
		for i := range embs {
			embs[i] = map[string]any{"values": []float32{float32(i), float32(i) + 0.5}}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"embeddings": embs})
	}
}

func TestEmbed_Success(t *testing.T) {
	adapter, _ := newTestClient(t, okHandler(t, 1), nil)
	vec, aerr := adapter.Embed(context.Background(), "hello")
	if aerr != nil {
		t.Fatalf("unexpected error: %v", aerr)
	}
	if len(vec) != 2 || vec[0] != 0 || vec[1] != 0.5 {
		t.Fatalf("unexpected vector: %v", vec)
	}
}

func TestEmbedBatch_Success(t *testing.T) {
	adapter, _ := newTestClient(t, okHandler(t, 3), nil)
	vecs, aerr := adapter.EmbedBatch(context.Background(), []string{"a", "b", "c"})
	if aerr != nil {
		t.Fatalf("unexpected error: %v", aerr)
	}
	if len(vecs) != 3 {
		t.Fatalf("expected 3 vectors, got %d", len(vecs))
	}
	if vecs[2][0] != 2 {
		t.Fatalf("order mismatch: %v", vecs[2])
	}
}

func TestEmbedBatch_EmptyReturnsInvalidInput(t *testing.T) {
	adapter, _ := newTestClient(t, okHandler(t, 0), nil)
	_, aerr := adapter.EmbedBatch(context.Background(), nil)
	if aerr == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestEmbed_TaskTypeSent(t *testing.T) {
	var body map[string]any
	handler := func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &body)
		okHandler(t, 1)(w, r)
	}
	adapter, _ := newTestClient(t, handler, ptrStr("RETRIEVAL_DOCUMENT"))
	if _, aerr := adapter.Embed(context.Background(), "x"); aerr != nil {
		t.Fatalf("unexpected error: %v", aerr)
	}
	reqs, _ := body["requests"].([]any)
	if len(reqs) != 1 {
		t.Fatalf("expected 1 request, got %d", len(reqs))
	}
	first, _ := reqs[0].(map[string]any)
	if first["taskType"] != "RETRIEVAL_DOCUMENT" {
		t.Fatalf("taskType missing or wrong: %v", first["taskType"])
	}
}

func TestEmbed_TaskTypeOmittedWhenNil(t *testing.T) {
	var body map[string]any
	handler := func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &body)
		okHandler(t, 1)(w, r)
	}
	adapter, _ := newTestClient(t, handler, nil)
	if _, aerr := adapter.Embed(context.Background(), "x"); aerr != nil {
		t.Fatalf("unexpected error: %v", aerr)
	}
	reqs, _ := body["requests"].([]any)
	first, _ := reqs[0].(map[string]any)
	if _, exists := first["taskType"]; exists {
		t.Fatalf("taskType should be omitted, got %v", first["taskType"])
	}
}

func TestEmbed_RequestShape(t *testing.T) {
	var body map[string]any
	var path string
	var apiKeyHdr string
	handler := func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		apiKeyHdr = r.Header.Get("x-goog-api-key")
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &body)
		okHandler(t, 1)(w, r)
	}
	adapter, _ := newTestClient(t, handler, nil)
	if _, aerr := adapter.Embed(context.Background(), "x"); aerr != nil {
		t.Fatalf("unexpected error: %v", aerr)
	}
	if !strings.HasSuffix(path, "/models/text-embedding-004:batchEmbedContents") {
		t.Fatalf("unexpected path: %s", path)
	}
	if apiKeyHdr != "test-key" {
		t.Fatalf("missing x-goog-api-key: %q", apiKeyHdr)
	}
	reqs, _ := body["requests"].([]any)
	first, _ := reqs[0].(map[string]any)
	if first["model"] != "models/text-embedding-004" {
		t.Fatalf("model field missing models/ prefix: %v", first["model"])
	}
	if v, _ := first["outputDimensionality"].(float64); int(v) != 768 {
		t.Fatalf("outputDimensionality wrong: %v", first["outputDimensionality"])
	}
}

func TestEmbed_Non200ReturnsError(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("boom"))
	}
	adapter, _ := newTestClient(t, handler, nil)
	_, aerr := adapter.Embed(context.Background(), "x")
	if aerr == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestEmbedBatch_CountMismatchReturnsError(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"embeddings": []map[string]any{
				{"values": []float32{1, 2}},
			},
		})
	}
	adapter, _ := newTestClient(t, handler, nil)
	_, aerr := adapter.EmbedBatch(context.Background(), []string{"a", "b"})
	if aerr == nil {
		t.Fatalf("expected error on count mismatch")
	}
}

func TestDimension(t *testing.T) {
	adapter, _ := newTestClient(t, okHandler(t, 0), nil)
	if d := adapter.Dimension(); d != 768 {
		t.Fatalf("expected 768, got %d", d)
	}
}
