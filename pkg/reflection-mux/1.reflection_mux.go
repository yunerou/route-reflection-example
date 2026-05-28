package reflectionmux

import (
	"sync"
)

type reflectionMux struct {
	prefixRoute   string
	lazyRegisters []func()
	serveOnce     sync.Once

	routeInfo     []RouteInfo
	routeHandlers []RoutHandler
}

func (m *reflectionMux) SetPathPrefix(prefix string) {
	m.prefixRoute = prefix
}

func (m *reflectionMux) getHandlers() []RoutHandler {
	m.runLazyRegister()
	return m.routeHandlers
}

func (m *reflectionMux) reflectionRouteInfo() []RouteInfo {
	m.runLazyRegister()
	return m.routeInfo
}

func (m *reflectionMux) runLazyRegister() {
	m.serveOnce.Do(func() {
		m.routeHandlers = make([]RoutHandler, 0)
		for _, lazyRegister := range m.lazyRegisters {
			lazyRegister()
		}
		m.lazyRegisters = nil
	})
}
