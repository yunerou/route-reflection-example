package muxrouter

import "sync"

type GroupRouter struct {
	base        *coreMuxRouter
	prefixRoute string

	lazyRegisters []func()
	serveOnce     sync.Once
}

func (m *GroupRouter) runLazyRegister() {
	m.serveOnce.Do(func() {
		for _, lazyRegister := range m.lazyRegisters {
			lazyRegister()
		}
		m.lazyRegisters = nil
	})
}
