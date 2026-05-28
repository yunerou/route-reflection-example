package otelprovider

import (
	"context"

	"github.com/samber/do/v2"
	"github.com/samber/lo"
	otelprv "github.com/yunerou/niarb/pkg/otel"
	configprovider "github.com/yunerou/niarb/provider/config-provider"
	fncollector "github.com/yunerou/niarb/provider/fn-collector"
	"github.com/yunerou/niarb/shared/actx"
	"github.com/yunerou/niarb/shared/constants"
)

func NewDI(i do.Injector) (otelprv.OtelProvider, error) {
	cfg := do.MustInvoke[configprovider.ConfigStore](i)
	cleanupTask := do.MustInvoke[fncollector.CleanupTask](i)
	env := cfg.Env()

	otelIns := otelprv.NewOtelProvider(&otelprv.OtelConfig{
		AppName:           constants.AppNameShort,
		ExporterType:      env.Otel.ExporterType,
		CollectorEndpoint: lo.ToPtr(env.Otel.CollectorEndpoint),
		Insecure:          env.Otel.CollectorInsecure,
		ExtractAttr: func(ctx context.Context) map[string]string {
			aCtx := actx.From(ctx)
			rid := aCtx.GetTraceID()
			return map[string]string{
				"traceID": rid,
			}
		},
	})

	// Register cleanup task
	cleanupTask.RegTask(func() {
		_ = otelIns.Shutdown(context.Background())
	}, fncollector.FnPriorityNormal)

	return otelIns, nil
}
