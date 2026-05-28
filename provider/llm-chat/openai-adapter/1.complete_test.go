package openaichatadapter

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	configprovider "github.com/yunerou/niarb/provider/config-provider"
	llmchat "github.com/yunerou/niarb/provider/llm-chat"
)

func ptrStr(s string) *string   { return &s }
func ptrF32(f float32) *float32 { return &f }
func ptrInt(i int) *int         { return &i }

func newTestClient(t *testing.T, handler http.HandlerFunc, apiKey *string) llmchat.Adapter {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return New(&configprovider.LLMChatT{
		BaseURL: srv.URL,
		Model:   "test-model",
		APIKey:  apiKey,
	})
}

func okHandler(content string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{"role": "assistant", "content": content}},
			},
		})
	}
}

func TestComplete_Success(t *testing.T) {
	client := newTestClient(t, okHandler(`{"doc_type":"api-docs"}`), nil)
	resp, aerr := client.Complete(context.Background(), llmchat.CompleteRequest{
		System: "sys", User: "usr",
	})
	if aerr != nil {
		t.Fatalf("unexpected error: %v", aerr)
	}
	if resp.Content != `{"doc_type":"api-docs"}` {
		t.Fatalf("content mismatch: %q", resp.Content)
	}
}

func TestComplete_WithAPIKey_SetsAuthHeader(t *testing.T) {
	var gotAuth string
	handler := func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		okHandler("ok")(w, r)
	}
	client := newTestClient(t, handler, ptrStr("sk-test"))
	if _, aerr := client.Complete(context.Background(), llmchat.CompleteRequest{System: "s", User: "u"}); aerr != nil {
		t.Fatalf("unexpected error: %v", aerr)
	}
	if gotAuth != "Bearer sk-test" {
		t.Fatalf("expected Bearer sk-test, got %q", gotAuth)
	}
}

func TestComplete_WithoutAPIKey_OmitsAuthHeader(t *testing.T) {
	var gotAuth string
	handler := func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		okHandler("ok")(w, r)
	}
	client := newTestClient(t, handler, nil)
	if _, aerr := client.Complete(context.Background(), llmchat.CompleteRequest{System: "s", User: "u"}); aerr != nil {
		t.Fatalf("unexpected error: %v", aerr)
	}
	if gotAuth != "" {
		t.Fatalf("expected no auth header, got %q", gotAuth)
	}
}

func TestComplete_SendsOptionalFields(t *testing.T) {
	var body map[string]any
	handler := func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &body)
		okHandler("ok")(w, r)
	}
	client := newTestClient(t, handler, nil)
	_, aerr := client.Complete(context.Background(), llmchat.CompleteRequest{
		System:         "sys",
		User:           "usr",
		ResponseFormat: &llmchat.ResponseFormat{Type: "json_object"},
		Temperature:    ptrF32(0),
		MaxTokens:      ptrInt(64),
	})
	if aerr != nil {
		t.Fatalf("unexpected error: %v", aerr)
	}
	if body["model"] != "test-model" {
		t.Fatalf("model field missing or wrong: %v", body["model"])
	}
	rf, _ := body["response_format"].(map[string]any)
	if rf == nil || rf["type"] != "json_object" {
		t.Fatalf("response_format not sent: %v", body["response_format"])
	}
	if _, ok := body["temperature"]; !ok {
		t.Fatalf("temperature not sent")
	}
	if _, ok := body["max_tokens"]; !ok {
		t.Fatalf("max_tokens not sent")
	}
	msgs, _ := body["messages"].([]any)
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
}

func TestComplete_OmitsOptionalFieldsWhenNil(t *testing.T) {
	var body map[string]any
	handler := func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &body)
		okHandler("ok")(w, r)
	}
	client := newTestClient(t, handler, nil)
	_, _ = client.Complete(context.Background(), llmchat.CompleteRequest{System: "s", User: "u"})
	if _, ok := body["response_format"]; ok {
		t.Fatalf("response_format should be omitted")
	}
	if _, ok := body["temperature"]; ok {
		t.Fatalf("temperature should be omitted")
	}
	if _, ok := body["max_tokens"]; ok {
		t.Fatalf("max_tokens should be omitted")
	}
}

func TestComplete_Non200ReturnsError(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("boom"))
	}
	client := newTestClient(t, handler, nil)
	_, aerr := client.Complete(context.Background(), llmchat.CompleteRequest{System: "s", User: "u"})
	if aerr == nil {
		t.Fatalf("expected error, got nil")
	}
	if aerr.Origin() == nil || !strings.Contains(aerr.Origin().Error(), "500") {
		t.Fatalf("expected status in origin, got %v", aerr.Origin())
	}
}

func TestComplete_MalformedJSONReturnsError(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("not json"))
	}
	client := newTestClient(t, handler, nil)
	_, aerr := client.Complete(context.Background(), llmchat.CompleteRequest{System: "s", User: "u"})
	if aerr == nil {
		t.Fatalf("expected error on malformed body")
	}
}

func TestComplete_EmptyChoicesReturnsError(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"choices": []any{}})
	}
	client := newTestClient(t, handler, nil)
	_, aerr := client.Complete(context.Background(), llmchat.CompleteRequest{System: "s", User: "u"})
	if aerr == nil {
		t.Fatalf("expected error on empty choices")
	}
}
