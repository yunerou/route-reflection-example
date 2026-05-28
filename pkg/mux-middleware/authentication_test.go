package muxmiddleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

type ctxAuthKey struct{}

func TestAuthentication_PassesEmptyHeaderToCallback(t *testing.T) {
	mw := NewMiddlewareProvider(&MWConfig{AuthHeader: "Authorization"})

	var got string
	called := false
	cb := func(ctx context.Context, headerValue string) context.Context {
		called = true
		got = headerValue
		return context.WithValue(ctx, ctxAuthKey{}, headerValue)
	}

	handler := mw.Authentication(cb)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		v, _ := r.Context().Value(ctxAuthKey{}).(string)
		if v != "" {
			t.Errorf("downstream ctx value = %q, want empty", v)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !called {
		t.Fatal("callback was not invoked")
	}
	if got != "" {
		t.Errorf("callback received %q, want empty string", got)
	}
}

func TestAuthentication_PassesHeaderValueToCallback(t *testing.T) {
	mw := NewMiddlewareProvider(&MWConfig{AuthHeader: "Authorization"})

	var got string
	cb := func(ctx context.Context, headerValue string) context.Context {
		got = headerValue
		return context.WithValue(ctx, ctxAuthKey{}, headerValue)
	}

	handler := mw.Authentication(cb)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		v, _ := r.Context().Value(ctxAuthKey{}).(string)
		if v != "Bearer abc" {
			t.Errorf("downstream ctx value = %q, want %q", v, "Bearer abc")
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer abc")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got != "Bearer abc" {
		t.Errorf("callback received %q, want %q", got, "Bearer abc")
	}
}

func TestAuthentication_UsesConfiguredHeaderName(t *testing.T) {
	mw := NewMiddlewareProvider(&MWConfig{AuthHeader: "X-Custom-Auth"})

	var got string
	cb := func(ctx context.Context, headerValue string) context.Context {
		got = headerValue
		return ctx
	}

	handler := mw.Authentication(cb)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Custom-Auth", "token-xyz")
	req.Header.Set("Authorization", "should-be-ignored")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got != "token-xyz" {
		t.Errorf("callback received %q, want %q", got, "token-xyz")
	}
}

func TestAuthentication_FallsBackToAuthorizationWhenConfigEmpty(t *testing.T) {
	mw := NewMiddlewareProvider(&MWConfig{AuthHeader: ""})

	var got string
	cb := func(ctx context.Context, headerValue string) context.Context {
		got = headerValue
		return ctx
	}

	handler := mw.Authentication(cb)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "fallback-token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got != "fallback-token" {
		t.Errorf("callback received %q, want %q", got, "fallback-token")
	}
}

func TestAuthentication_NilCallbackPassesThrough(t *testing.T) {
	mw := NewMiddlewareProvider(&MWConfig{AuthHeader: "Authorization"})

	called := false
	handler := mw.Authentication(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "irrelevant")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !called {
		t.Fatal("downstream handler not invoked when callback is nil")
	}
}
