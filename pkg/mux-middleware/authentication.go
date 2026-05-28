package muxmiddleware

import (
	"context"
	"net/http"
)

const defaultAuthHeader = "Authorization"

// Authentication reads the configured auth header from the request and hands its
// raw value to callbackFn, which returns the context to propagate downstream.
// The middleware is intentionally generic over the auth representation: the
// verifier and any actx injection live on the caller side.
func (mw *middlewareProvider) Authentication(
	callbackFn func(context.Context, string) context.Context,
) func(http.Handler) http.Handler {
	header := mw.config.AuthHeader
	if header == "" {
		header = defaultAuthHeader
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			if callbackFn != nil {
				ctx = callbackFn(ctx, r.Header.Get(header))
			}
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
