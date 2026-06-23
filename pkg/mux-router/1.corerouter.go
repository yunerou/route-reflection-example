package muxrouter

import (
	"fmt"
	"net/http"
	"strings"
)

type coreMuxRouter struct {
	adapter Adapter
	mainMux *http.ServeMux
	paths   map[string]*GroupRouter
}

func (c *coreMuxRouter) Create(pathPrefix string) *GroupRouter {
	trimmedPrefix := strings.Trim(pathPrefix, "/")

	if !isValidPathPrefix(trimmedPrefix) {
		panic(fmt.Sprintf("invalid path prefix %q: only letters, digits, '-', '_', '.' and '/' are allowed", pathPrefix))
	}

	if c.paths == nil {
		c.paths = make(map[string]*GroupRouter)
	}

	if existing, ok := c.paths[trimmedPrefix]; ok {
		return existing
	}

	c.paths[trimmedPrefix] = &GroupRouter{
		base:        c,
		prefixRoute: trimmedPrefix,
	}
	return c.paths[trimmedPrefix]
}

func (c *coreMuxRouter) ExtractHandler() http.Handler {
	for _, g := range c.paths {
		g.runLazyRegister()
	}

	return c.mainMux
}

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
