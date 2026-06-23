package main

import (
	"log/slog"

	"github.com/samber/do/v2"
	"github.com/yunerou/niarb/app"
	"github.com/yunerou/niarb/app/server"
	exampleDeli "github.com/yunerou/niarb/internal/example/delivery"
	configprovider "github.com/yunerou/niarb/provider/config-provider"
	fncollector "github.com/yunerou/niarb/provider/fn-collector"
	healthyprovider "github.com/yunerou/niarb/provider/healthy-provider"
	muxmiddleware "github.com/yunerou/niarb/provider/mux-middleware"
	observerprovider "github.com/yunerou/niarb/provider/observer-provider"
	otelprovider "github.com/yunerou/niarb/provider/otel-provider"
	validationprovider "github.com/yunerou/niarb/provider/validation-provider"
	workerpoolprovider "github.com/yunerou/niarb/provider/worker-pool-provider"
)

func registerProviders(i do.Injector) {
	// --- primitives & config ---
	do.ProvideValue(i, slog.Default())
	do.Provide(i, fncollector.NewDICleanupTask)
	do.Provide(i, fncollector.NewDIIntervalTask)
	do.Provide(i, validationprovider.NewDI)

	cfg := configprovider.FromYaml([]string{configPath})
	do.ProvideValue(i, cfg)

	// --- infrastructure (config-dependent) ---

	do.Provide(i, healthyprovider.NewDI)
	do.Provide(i, muxmiddleware.NewDI)
	do.Provide(i, workerpoolprovider.NewDI)
	do.Provide(i, otelprovider.NewDI)

	do.Provide(i, exampleDeli.NewDI)

	// --- observability & events ---
	do.Provide(i, observerprovider.NewDI)

	// --- app commands ---
	do.Provide(i, func(ix do.Injector) (*server.SvCmd, error) {
		return server.NewSvCmd(ix), nil
	})
	do.Provide(i, func(ix do.Injector) (*app.CmdApp, error) {
		return app.NewCmdApp(ix), nil
	})
}
