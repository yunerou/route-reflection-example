package observerprovider

import (
	"context"
	"log/slog"

	"github.com/samber/do/v2"
	"github.com/yunerou/niarb/pkg/observer"
	"github.com/yunerou/niarb/pkg/observer/obs"
	otelprv "github.com/yunerou/niarb/pkg/otel"
	configprovider "github.com/yunerou/niarb/provider/config-provider"
	"github.com/yunerou/niarb/shared/actx"
	"github.com/yunerou/niarb/shared/constants"
)

func NewDI(i do.Injector) (observer.Observability, error) {
	cfg := do.MustInvoke[configprovider.ConfigStore](i)
	otelProvider := do.MustInvoke[otelprv.OtelProvider](i)
	slogIns := do.MustInvoke[*slog.Logger](i)

	env := cfg.Env()
	logLevel := slog.Level(env.Otel.LogLevel)
	obsIns := obs.NewObservability(&obs.ObservabilityConfig{
		ServiceName:    constants.AppNameShort,
		ServiceVersion: env.Version,
		Environment:    env.Info.Env,
		GetTraceIdFn: func(ctx context.Context) string {
			traceId := actx.From(ctx).GetTraceID()
			return traceId
		},
		// GetAuthStringFn: func(ctx context.Context) string {
		// 	auth := actx.From(ctx).GetAuth()
		// 	if auth.IsAnonymous() {
		// 		return fmt.Sprintf("anon[%s]", auth.AnonymousID)
		// 	}
		// 	return fmt.Sprintf("%s@%s", auth.UserID, auth.AnonymousID)
		// },
		OtelTrace: otelProvider,
		Slogger:   slogIns,
		SlogLevel: logLevel,
	})

	return obsIns, nil
}
