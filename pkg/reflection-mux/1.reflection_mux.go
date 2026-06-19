package reflectionmux

import (
	"sync"
)

type coreReflectionMux struct {
	lazyRegisters []func()
	serveOnce     sync.Once
	routeInfo     []RouteInfo
	routeHandlers []RoutHandler
}

type pathReflectionMux struct {
	*coreReflectionMux
	prefixRoute string
}

func (m *coreReflectionMux) getHandlers() []RoutHandler {
	m.runLazyRegister()
	return m.routeHandlers
}

func (m *coreReflectionMux) reflectionRouteInfo() []RouteInfo {
	m.runLazyRegister()
	return m.routeInfo
}

func (m *coreReflectionMux) runLazyRegister() {
	m.serveOnce.Do(func() {
		m.routeHandlers = make([]RoutHandler, 0)
		for _, lazyRegister := range m.lazyRegisters {
			lazyRegister()
		}
		m.lazyRegisters = nil
	})
}

func (c *coreReflectionMux) Create(pathPrefix string) PathReflectionMux {
	return &pathReflectionMux{
		coreReflectionMux: c,
		prefixRoute:       pathPrefix,
	}
}
