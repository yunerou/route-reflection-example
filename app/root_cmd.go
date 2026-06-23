package app

import (
	"github.com/samber/do/v2"
	"github.com/urfave/cli/v2"
	"github.com/yunerou/niarb/app/server"
)

type CmdApp struct {
	*cli.App

	s *server.SvCmd
}

func NewCmdApp(i do.Injector) *CmdApp {
	app := &CmdApp{
		s: do.MustInvoke[*server.SvCmd](i),
	}
	app.App = &cli.App{
		EnableBashCompletion: true,
		Name:                 "cbt",
		Usage:                "Common Bank Transfer",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "lang",
				Usage: "Language for display. If this flag not exist -> use LANG env var. Currently support [ja en]",
			},
		},
		Commands: app.subCommand(),
	}
	return app
}
