package geminivertexadapter

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"golang.org/x/oauth2"

	llmembedding "github.com/yunerou/niarb/provider/llm-embedding"
)

func ptrStr(s string) *string { return &s }

type staticTokenSource struct{ token string }

func (s *staticTokenSource) Token() (*oauth2.Token, error) {
	return &oauth2.Token{AccessToken: s.token, Expiry: time.Now().Add(time.Hour)}, nil
}

func okHandler(t *testing.T, batchSize int) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		preds := make([]map[string]any, batchSize)
		for i := range preds {
			preds[i] = map[string]any{
				"embeddings": map[string]any{
					"values": []float32{float32(i), float32(i) + 0.5},
				},
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"predictions": preds})
	}
}

func newTestAdapter(t *testing.T, handler http.HandlerFunc, taskType *string) llmembedding.Adapter {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return newClientWithDeps(
		"text-multilingual-embedding-002",
		"my-proj",
		"us-central1",
		768,
		taskType,
		&staticTokenSource{token: "test-token"},
		srv.URL,
	)
}

func TestEmbed_Success(t *testing.T) {
	adapter := newTestAdapter(t, okHandler(t, 1), nil)
	vec, aerr := adapter.Embed(context.Background(), "hello")
	if aerr != nil {
		t.Fatalf("unexpected error: %v", aerr)
	}
	if len(vec) != 2 || vec[0] != 0 || vec[1] != 0.5 {
		t.Fatalf("unexpected vector: %v", vec)
	}
}

func TestEmbedBatch_Success(t *testing.T) {
	adapter := newTestAdapter(t, okHandler(t, 3), nil)
	vecs, aerr := adapter.EmbedBatch(context.Background(), []string{"a", "b", "c"})
	if aerr != nil {
		t.Fatalf("unexpected error: %v", aerr)
	}
	if len(vecs) != 3 || vecs[2][0] != 2 {
		t.Fatalf("unexpected vectors: %v", vecs)
	}
}

func TestEmbedBatch_EmptyReturnsInvalidInput(t *testing.T) {
	adapter := newTestAdapter(t, okHandler(t, 0), nil)
	_, aerr := adapter.EmbedBatch(context.Background(), nil)
	if aerr == nil {
		t.Fatalf("expected error")
	}
}

func TestEmbedBatch_ExceedsLimitReturnsInvalidInput(t *testing.T) {
	adapter := newTestAdapter(t, okHandler(t, 0), nil)
	texts := make([]string, vertexMaxBatch+1)
	for i := range texts {
		texts[i] = "x"
	}
	_, aerr := adapter.EmbedBatch(context.Background(), texts)
	if aerr == nil {
		t.Fatalf("expected error on batch overflow")
	}
}

func TestEmbed_EndpointAndAuth(t *testing.T) {
	var gotPath, gotAuth string
	handler := func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		okHandler(t, 1)(w, r)
	}
	adapter := newTestAdapter(t, handler, nil)
	if _, aerr := adapter.Embed(context.Background(), "x"); aerr != nil {
		t.Fatalf("unexpected error: %v", aerr)
	}
	wantPath := "/v1/projects/my-proj/locations/us-central1/publishers/google/models/text-multilingual-embedding-002:predict"
	if gotPath != wantPath {
		t.Fatalf("path mismatch:\n  got:  %s\n  want: %s", gotPath, wantPath)
	}
	if gotAuth != "Bearer test-token" {
		t.Fatalf("auth header mismatch: %q", gotAuth)
	}
}

func TestEmbed_TaskTypeSnakeCase(t *testing.T) {
	var body map[string]any
	handler := func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &body)
		okHandler(t, 1)(w, r)
	}
	adapter := newTestAdapter(t, handler, ptrStr("RETRIEVAL_DOCUMENT"))
	if _, aerr := adapter.Embed(context.Background(), "x"); aerr != nil {
		t.Fatalf("unexpected error: %v", aerr)
	}
	instances, _ := body["instances"].([]any)
	first, _ := instances[0].(map[string]any)
	if first["task_type"] != "RETRIEVAL_DOCUMENT" {
		t.Fatalf("task_type missing or wrong: %v", first["task_type"])
	}
	if _, exists := first["taskType"]; exists {
		t.Fatalf("camelCase taskType should not be sent")
	}
}

func TestEmbed_TaskTypeOmittedWhenNil(t *testing.T) {
	var body map[string]any
	handler := func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &body)
		okHandler(t, 1)(w, r)
	}
	adapter := newTestAdapter(t, handler, nil)
	if _, aerr := adapter.Embed(context.Background(), "x"); aerr != nil {
		t.Fatalf("unexpected error: %v", aerr)
	}
	instances, _ := body["instances"].([]any)
	first, _ := instances[0].(map[string]any)
	if _, exists := first["task_type"]; exists {
		t.Fatalf("task_type should be omitted")
	}
}

func TestEmbed_ParametersBlock(t *testing.T) {
	var body map[string]any
	handler := func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &body)
		okHandler(t, 1)(w, r)
	}
	adapter := newTestAdapter(t, handler, nil)
	if _, aerr := adapter.Embed(context.Background(), "x"); aerr != nil {
		t.Fatalf("unexpected error: %v", aerr)
	}
	params, _ := body["parameters"].(map[string]any)
	if params == nil {
		t.Fatalf("parameters block missing")
	}
	if v, _ := params["outputDimensionality"].(float64); int(v) != 768 {
		t.Fatalf("outputDimensionality wrong: %v", params["outputDimensionality"])
	}
	instances, _ := body["instances"].([]any)
	first, _ := instances[0].(map[string]any)
	if _, ok := first["outputDimensionality"]; ok {
		t.Fatalf("outputDimensionality should not be per-instance")
	}
}

func TestEmbed_Non200ReturnsError(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("boom"))
	}
	adapter := newTestAdapter(t, handler, nil)
	_, aerr := adapter.Embed(context.Background(), "x")
	if aerr == nil {
		t.Fatalf("expected error")
	}
}

func TestEmbedBatch_CountMismatchReturnsError(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"predictions": []map[string]any{
				{"embeddings": map[string]any{"values": []float32{1, 2}}},
			},
		})
	}
	adapter := newTestAdapter(t, handler, nil)
	_, aerr := adapter.EmbedBatch(context.Background(), []string{"a", "b"})
	if aerr == nil {
		t.Fatalf("expected error on count mismatch")
	}
}

func TestDimension(t *testing.T) {
	adapter := newClientWithDeps(
		"m", "p", "us-central1", 768, nil,
		&staticTokenSource{token: "tk"}, "http://unused",
	)
	if d := adapter.Dimension(); d != 768 {
		t.Fatalf("expected 768, got %d", d)
	}
}
