package server

import (
	"context"
	"net/http"
	"regexp"
	"slices"

	"github.com/yunerou/niarb/shared/actx"
	"github.com/yunerou/niarb/shared/encdec"
)

type middleware = func(http.Handler) http.Handler

func (c *SvCmd) simpleMiddleware(h http.Handler) http.Handler {
	mw := []middleware{
		injectContext(),
		setDefaultEncoderDecoder(),
	}
	return chainMiddleware(h, mw...)
}

func (c *SvCmd) allMiddilewares(h http.Handler) http.Handler {
	e := c.c.Env()

	mw := []middleware{
		injectContext(),
		setDefaultEncoderDecoder(),
		c.mw.JSONLogFmt(nil),
		c.mw.RequestID(func(ctx context.Context, rId string) context.Context {
			aCtx := actx.From(ctx)
			aCtx.SetTraceID(rId)
			return aCtx
		}),
		// c.mw.Authentication(func(ctx context.Context, headerValue string) context.Context {
		// 	auth := verifyAuthHeader(ctx, headerValue)
		// 	actx.From(ctx).SetAuth(auth)
		// 	return ctx
		// }),
		c.mw.PanicRecover(),
		c.mw.EncoderDecoder(),
		// c.mw.InjectCmdQueue(),
	}
	if e.Otel.Enabled {
		mw = append(mw, c.mw.Otel())
	}
	if e.Cors.Enabled {
		mw = append(mw, c.corsMiddleware)
	}

	return chainMiddleware(h, mw...)
}

func chainMiddleware(h http.Handler, middlewares ...middleware) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}
	return h
}

func (c *SvCmd) corsMiddleware(next http.Handler) http.Handler {
	config := c.c.Env().Cors
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && isAllowedOrigin(origin, config.ExactlyOrigins, config.RegexOrigins) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func isAllowedOrigin(origin string, exactlyOrigins, regexOrigins []string) bool {
	if slices.Contains(exactlyOrigins, origin) {
		return true
	}
	for _, pattern := range regexOrigins {
		matched, err := regexp.MatchString(pattern, origin)
		if err == nil && matched {
			return true
		}
	}
	return false
}

func setDefaultEncoderDecoder() middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			aCtx := actx.From(r.Context())
			aCtx.SetEncoderDecoder(encdec.JSONEncoder(), encdec.JSONDecoder())
			next.ServeHTTP(w, r)
		})
	}
}

func injectContext() middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			aCtx := actx.From(r.Context())
			next.ServeHTTP(w, r.WithContext(aCtx))
		})
	}
}
