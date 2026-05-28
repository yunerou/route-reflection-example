package main

import (
	"log/slog"
	"os"

	"github.com/samber/do/v2"

	"github.com/yunerou/niarb/app"
)

// Relative to CWD; pkg/koanf joins with "." which strips a leading slash.
// In the dev container CWD is /app, so this resolves to /app/config/config.yaml.
const configPath = "config/config.yaml"

func main() {
	i := do.New()
	defer func() { _ = i.Shutdown() }()

	registerProviders(i)

	cmdApp := do.MustInvoke[*app.CmdApp](i)
	if err := cmdApp.Run(os.Args); err != nil {
		slog.Error("app exited with error", "err", err)
		os.Exit(1)
	}
}
