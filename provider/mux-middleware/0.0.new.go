package muxmiddleware

import (
	"github.com/samber/do/v2"
	mm "github.com/yunerou/niarb/pkg/mux-middleware"
	configprovider "github.com/yunerou/niarb/provider/config-provider"
)

func NewDI(i do.Injector) (mm.MiddlewareProvider, error) {
	cfg := do.MustInvoke[configprovider.ConfigStore](i)
	config := cfg.Env()
	ins := mm.NewMiddlewareProvider(
		&mm.MWConfig{
			IgnoreAccessLogPath: []string{
				"/api/healthz",
				"/api/readyz",
				"/api/livez",
			},
			TraceIDHeader: config.ExtractHeader.TraceIDHeader,
			AuthHeader:    config.ExtractHeader.AuthHeader,
		},
	)

	return ins, nil
}
