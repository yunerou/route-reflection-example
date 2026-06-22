package reflectionmux

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
)

type coreReflectionMux struct {
	encoderDecoder     EncoderDecoder
	convertErrorSchema func(error) (httpStatus int, body any)
	commonInfo         CommonInfo

	paths map[string]PathReflectionMux
}

type pathReflectionMux struct {
	*coreReflectionMux
	prefixRoute string

	// Per-path registration state. Each path prefix owns its own handlers and
	// route info so that ExtractReflectionMux can iterate every path without
	// re-registering another path's routes.
	lazyRegisters []func()
	serveOnce     sync.Once
	routeInfo     []RouteInfo
	routeHandlers []RoutHandler
}

func (m *pathReflectionMux) getHandlers() []RoutHandler {
	m.runLazyRegister()
	return m.routeHandlers
}

func (m *pathReflectionMux) reflectionRouteInfo() []RouteInfo {
	m.runLazyRegister()
	return m.routeInfo
}

func (m *pathReflectionMux) runLazyRegister() {
	m.serveOnce.Do(func() {
		m.routeHandlers = make([]RoutHandler, 0)
		for _, lazyRegister := range m.lazyRegisters {
			lazyRegister()
		}
		m.lazyRegisters = nil
	})
}
func (m *coreReflectionMux) ExtractHandler() http.Handler {
	return extractReflectionMux(m)
}

func (c *coreReflectionMux) Create(pathPrefix string) PathReflectionMux {
	trimmedPrefix := strings.Trim(pathPrefix, "/")

	// 1) Validate the prefix: only allow path-safe characters.
	if !isValidPathPrefix(trimmedPrefix) {
		panic(fmt.Sprintf("invalid path prefix %q: only letters, digits, '-', '_', '.' and '/' are allowed", pathPrefix))
	}

	if c.paths == nil {
		c.paths = make(map[string]PathReflectionMux)
	}

	// 2) Reuse the existing mux when the prefix was already created.
	if existing, ok := c.paths[trimmedPrefix]; ok {
		return existing
	}

	c.paths[trimmedPrefix] = &pathReflectionMux{
		coreReflectionMux: c,
		prefixRoute:       trimmedPrefix,
	}
	return c.paths[trimmedPrefix]
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
func (c *coreReflectionMux) GetAllPaths() map[string]PathReflectionMux {
	return c.paths
}
