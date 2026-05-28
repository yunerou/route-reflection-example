package geminivertexchatadapter

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"golang.org/x/oauth2"

	llmchat "github.com/yunerou/niarb/provider/llm-chat"
)

func ptrF32(f float32) *float32 { return &f }
func ptrInt(i int) *int         { return &i }

type staticTokenSource struct{ token string }

func (s *staticTokenSource) Token() (*oauth2.Token, error) {
	return &oauth2.Token{AccessToken: s.token, Expiry: time.Now().Add(time.Hour)}, nil
}

func newTestAdapter(t *testing.T, handler http.HandlerFunc) llmchat.Adapter {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return newClientWithDeps(
		"gemini-1.5-flash",
		"my-proj",
		"us-central1",
		&staticTokenSource{token: "test-token"},
		srv.URL,
	)
}

func okHandler(text string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"candidates": []map[string]any{
				{
					"content":      map[string]any{"parts": []map[string]any{{"text": text}}},
					"finishReason": "STOP",
				},
			},
		})
	}
}

func TestComplete_Success(t *testing.T) {
	adapter := newTestAdapter(t, okHandler(`{"doc_type":"api-docs"}`))
	resp, aerr := adapter.Complete(context.Background(), llmchat.CompleteRequest{User: "u"})
	if aerr != nil {
		t.Fatalf("unexpected error: %v", aerr)
	}
	if resp.Content != `{"doc_type":"api-docs"}` {
		t.Fatalf("content mismatch: %q", resp.Content)
	}
}

func TestComplete_SetsBearerToken(t *testing.T) {
	var gotAuth string
	handler := func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		okHandler("ok")(w, r)
	}
	adapter := newTestAdapter(t, handler)
	if _, aerr := adapter.Complete(context.Background(), llmchat.CompleteRequest{User: "u"}); aerr != nil {
		t.Fatalf("unexpected error: %v", aerr)
	}
	if gotAuth != "Bearer test-token" {
		t.Fatalf("expected Bearer test-token, got %q", gotAuth)
	}
}

func TestComplete_URLPath(t *testing.T) {
	var gotPath string
	handler := func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		okHandler("ok")(w, r)
	}
	adapter := newTestAdapter(t, handler)
	if _, aerr := adapter.Complete(context.Background(), llmchat.CompleteRequest{User: "u"}); aerr != nil {
		t.Fatalf("unexpected error: %v", aerr)
	}
	want := "/v1/projects/my-proj/locations/us-central1/publishers/google/models/gemini-1.5-flash:generateContent"
	if gotPath != want {
		t.Fatalf("path mismatch:\n  got:  %s\n  want: %s", gotPath, want)
	}
}

func TestComplete_SendsSystemInstruction(t *testing.T) {
	var body map[string]any
	handler := func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &body)
		okHandler("ok")(w, r)
	}
	adapter := newTestAdapter(t, handler)
	_, _ = adapter.Complete(context.Background(), llmchat.CompleteRequest{
		System: "you are a classifier",
		User:   "input",
	})
	si, _ := body["systemInstruction"].(map[string]any)
	if si == nil {
		t.Fatalf("systemInstruction missing")
	}
	parts, _ := si["parts"].([]any)
	first, _ := parts[0].(map[string]any)
	if first["text"] != "you are a classifier" {
		t.Fatalf("system text mismatch: %v", first["text"])
	}
}

func TestComplete_OmitsSystemWhenEmpty(t *testing.T) {
	var body map[string]any
	handler := func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &body)
		okHandler("ok")(w, r)
	}
	adapter := newTestAdapter(t, handler)
	_, _ = adapter.Complete(context.Background(), llmchat.CompleteRequest{User: "u"})
	if _, ok := body["systemInstruction"]; ok {
		t.Fatalf("systemInstruction should be omitted when System is empty")
	}
}

func TestComplete_SendsUserContent(t *testing.T) {
	var body map[string]any
	handler := func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &body)
		okHandler("ok")(w, r)
	}
	adapter := newTestAdapter(t, handler)
	_, _ = adapter.Complete(context.Background(), llmchat.CompleteRequest{User: "hello"})
	contents, _ := body["contents"].([]any)
	if len(contents) != 1 {
		t.Fatalf("expected 1 content, got %d", len(contents))
	}
	first, _ := contents[0].(map[string]any)
	if first["role"] != "user" {
		t.Fatalf("expected role=user, got %v", first["role"])
	}
	parts, _ := first["parts"].([]any)
	if len(parts) != 1 {
		t.Fatalf("expected 1 part, got %d", len(parts))
	}
	p, _ := parts[0].(map[string]any)
	if p["text"] != "hello" {
		t.Fatalf("user text mismatch: %v", p["text"])
	}
}

func TestComplete_JSONMode(t *testing.T) {
	var body map[string]any
	handler := func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &body)
		okHandler("ok")(w, r)
	}
	adapter := newTestAdapter(t, handler)
	_, _ = adapter.Complete(context.Background(), llmchat.CompleteRequest{
		User:           "u",
		ResponseFormat: &llmchat.ResponseFormat{Type: "json_object"},
	})
	gc, _ := body["generationConfig"].(map[string]any)
	if gc["responseMimeType"] != "application/json" {
		t.Fatalf("expected responseMimeType=application/json, got %v", gc["responseMimeType"])
	}
}

func TestComplete_OmitsOptionalFieldsWhenNil(t *testing.T) {
	var body map[string]any
	handler := func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &body)
		okHandler("ok")(w, r)
	}
	adapter := newTestAdapter(t, handler)
	_, _ = adapter.Complete(context.Background(), llmchat.CompleteRequest{User: "u"})
	gc, _ := body["generationConfig"].(map[string]any)
	if _, ok := gc["temperature"]; ok {
		t.Fatalf("temperature should be omitted")
	}
	if _, ok := gc["maxOutputTokens"]; ok {
		t.Fatalf("maxOutputTokens should be omitted")
	}
	if _, ok := gc["responseMimeType"]; ok {
		t.Fatalf("responseMimeType should be omitted")
	}
}

func TestComplete_SendsTempAndMaxTokens(t *testing.T) {
	var body map[string]any
	handler := func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &body)
		okHandler("ok")(w, r)
	}
	adapter := newTestAdapter(t, handler)
	_, _ = adapter.Complete(context.Background(), llmchat.CompleteRequest{
		User:        "u",
		Temperature: ptrF32(0),
		MaxTokens:   ptrInt(64),
	})
	gc, _ := body["generationConfig"].(map[string]any)
	if _, ok := gc["temperature"]; !ok {
		t.Fatalf("temperature not sent")
	}
	if _, ok := gc["maxOutputTokens"]; !ok {
		t.Fatalf("maxOutputTokens not sent")
	}
}

func TestComplete_HardcodedSafetyBlockNone(t *testing.T) {
	var body map[string]any
	handler := func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &body)
		okHandler("ok")(w, r)
	}
	adapter := newTestAdapter(t, handler)
	_, _ = adapter.Complete(context.Background(), llmchat.CompleteRequest{User: "u"})
	settings, _ := body["safetySettings"].([]any)
	if len(settings) != 4 {
		t.Fatalf("expected 4 safety settings, got %d", len(settings))
	}
	wantCats := map[string]bool{
		"HARM_CATEGORY_HARASSMENT":        false,
		"HARM_CATEGORY_HATE_SPEECH":       false,
		"HARM_CATEGORY_SEXUALLY_EXPLICIT": false,
		"HARM_CATEGORY_DANGEROUS_CONTENT": false,
	}
	for _, s := range settings {
		m, _ := s.(map[string]any)
		cat, _ := m["category"].(string)
		thr, _ := m["threshold"].(string)
		if thr != "BLOCK_NONE" {
			t.Fatalf("category %s threshold should be BLOCK_NONE, got %s", cat, thr)
		}
		if _, ok := wantCats[cat]; ok {
			wantCats[cat] = true
		}
	}
	for cat, seen := range wantCats {
		if !seen {
			t.Fatalf("missing safety category: %s", cat)
		}
	}
}

func TestComplete_Non200ReturnsError(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("boom"))
	}
	adapter := newTestAdapter(t, handler)
	_, aerr := adapter.Complete(context.Background(), llmchat.CompleteRequest{User: "u"})
	if aerr == nil {
		t.Fatalf("expected error")
	}
	if aerr.Origin() == nil || !strings.Contains(aerr.Origin().Error(), "500") {
		t.Fatalf("expected status in origin, got %v", aerr.Origin())
	}
}

func TestComplete_PromptBlockedReturnsError(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"promptFeedback": map[string]any{"blockReason": "SAFETY"},
		})
	}
	adapter := newTestAdapter(t, handler)
	_, aerr := adapter.Complete(context.Background(), llmchat.CompleteRequest{User: "u"})
	if aerr == nil {
		t.Fatalf("expected error on prompt blocked")
	}
	if !strings.Contains(aerr.Origin().Error(), "SAFETY") {
		t.Fatalf("expected SAFETY in error origin, got %v", aerr.Origin())
	}
}

func TestComplete_EmptyCandidatesReturnsError(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"candidates": []any{}})
	}
	adapter := newTestAdapter(t, handler)
	_, aerr := adapter.Complete(context.Background(), llmchat.CompleteRequest{User: "u"})
	if aerr == nil {
		t.Fatalf("expected error on empty candidates")
	}
}

func TestComplete_EmptyPartsReturnsError(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"candidates": []map[string]any{
				{
					"content":      map[string]any{"parts": []any{}},
					"finishReason": "SAFETY",
				},
			},
		})
	}
	adapter := newTestAdapter(t, handler)
	_, aerr := adapter.Complete(context.Background(), llmchat.CompleteRequest{User: "u"})
	if aerr == nil {
		t.Fatalf("expected error on empty parts")
	}
	if !strings.Contains(aerr.Origin().Error(), "SAFETY") {
		t.Fatalf("expected finishReason in error, got %v", aerr.Origin())
	}
}

func TestComplete_MalformedJSONReturnsError(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("not json"))
	}
	adapter := newTestAdapter(t, handler)
	_, aerr := adapter.Complete(context.Background(), llmchat.CompleteRequest{User: "u"})
	if aerr == nil {
		t.Fatalf("expected error on malformed body")
	}
}
