package obs

import (
	"context"
	"log/slog"

	"github.com/yunerou/niarb/pkg/observer"
	"github.com/yunerou/niarb/pkg/otel"
)

type ObservabilityConfig struct {
	ServiceName    string
	ServiceVersion string
	Environment    string

	GetTraceIdFn    func(ctx context.Context) string
	GetAuthStringFn func(ctx context.Context) string

	// Tracing is disabled when OtelTrace is nil
	OtelTrace otel.OtelProvider

	// Use slog.Default() when nil
	Slogger   *slog.Logger
	SlogLevel slog.Level
}

type observability struct {
	cf *ObservabilityConfig
}

func NewObservability(cf *ObservabilityConfig) observer.Observability {
	return &observability{
		cf: cf,
	}
}
