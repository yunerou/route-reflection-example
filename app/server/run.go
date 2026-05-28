package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type ServerType int

const (
	Public ServerType = iota + 1
	Private
)

func (c *SvCmd) RunServerWithPort(ctx context.Context, t ServerType, port int) error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return err
	}

	return c.runServerWithListener(ctx, t, listener)
}

func (c *SvCmd) RunServerWithDynamicPort(ctx context.Context, t ServerType, portCh chan<- int) error {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		if portCh != nil {
			close(portCh)
		}
		return err
	}

	if portCh != nil {
		var port int
		port, err = portFromListener(listener)
		if err != nil {
			close(portCh)
			_ = listener.Close()
			return err
		}

		portCh <- port
		close(portCh)
	}

	return c.runServerWithListener(ctx, t, listener)
}

const (
	gracefulShutdownTimeout = 10 * time.Second
)

func (c *SvCmd) runServerWithListener(ctx context.Context, t ServerType, listener net.Listener) error {

	defer func() {
		for _, fn := range c.cleanupTask.Get() {
			fn()
		}
	}()

	srv := &http.Server{
		Handler: c.router(t),
	}

	signalCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	serverErrCh := make(chan error, 1)
	go func() {
		serverErrCh <- srv.Serve(listener)
	}()

	slog.InfoContext(ctx, fmt.Sprintf("Serve at port %s", listener.Addr().String()))

	select {
	case err := <-serverErrCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case <-signalCtx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), gracefulShutdownTimeout)
		defer cancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			return err
		}

		if err := <-serverErrCh; err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	}

	return nil
}

func portFromListener(listener net.Listener) (int, error) {
	addr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		return 0, fmt.Errorf("listener is not tcp: %T", listener.Addr())
	}

	return addr.Port, nil
}
