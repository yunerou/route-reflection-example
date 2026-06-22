package humapvd

import (
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
)

type coreHuma struct {
	convertErrorToHumaSchema func(error) huma.StatusError

	humaConfig huma.Config
	humaAPI    huma.API

	mainMux    *http.ServeMux
	commonInfo CommonInfo

	paths map[string]*GroupHuma
}

func (c *coreHuma) ExtractHandler(enableDoc bool) http.Handler {
	return extractHuma(c, enableDoc)
}

func (c *coreHuma) Create(pathPrefix string) *GroupHuma {
	trimmedPrefix := strings.Trim(pathPrefix, "/")

	if !isValidPathPrefix(trimmedPrefix) {
		panic(fmt.Sprintf("invalid path prefix %q: only letters, digits, '-', '_', '.' and '/' are allowed", pathPrefix))
	}

	if c.paths == nil {
		c.paths = make(map[string]*GroupHuma)
	}

	if existing, ok := c.paths[trimmedPrefix]; ok {
		return existing
	}

	c.paths[trimmedPrefix] = &GroupHuma{
		coreHuma:    c,
		prefixRoute: trimmedPrefix,
	}
	return c.paths[trimmedPrefix]
}

type GroupHuma struct {
	*coreHuma
	prefixRoute string

	// Per-path registration state. Each path prefix owns its own lazy funcs and
	// route info so that ExtractHandler can iterate every path independently.
	lazyRegisters []func()
	serveOnce     sync.Once
	routeInfo     []RouteInfo
}

func (m *GroupHuma) runLazyRegister() {
	m.serveOnce.Do(func() {
		for _, lazyRegister := range m.lazyRegisters {
			lazyRegister()
		}
		m.lazyRegisters = nil
	})
}

// isValidPathPrefix reports whether prefix contains only path-safe characters:
// letters, digits, '-', '_', '.' and '/'. An empty prefix (root) is valid.
func isValidPathPrefix(prefix string) bool {
	for _, r := range prefix {
		switch {
		case r >= 'a' && r <= 'z',
			r >= 'A' && r <= 'Z',
			r >= '0' && r <= '9',
			r == '-', r == '_', r == '.', r == '/':
		default:
			return false
		}
	}
	return true
}

type humaMiddleware func(ctx huma.Context, next func(huma.Context))

func convertToHumaMiddleware(muxMiddleware Middleware) humaMiddleware {
	return func(ctx huma.Context, next func(huma.Context)) {
		r, w := humago.Unwrap(ctx)
		muxMiddleware(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
			next(ctx)
		})).ServeHTTP(w, r)
	}
}
