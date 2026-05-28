package app

import (
	"github.com/samber/do/v2"
	"github.com/urfave/cli/v2"
	"github.com/yunerou/niarb/app/server"
	"github.com/yunerou/niarb/app/task"
)

type CmdApp struct {
	*cli.App

	s *server.SvCmd
	t *task.TaskCmd
}

func NewCmdApp(i do.Injector) *CmdApp {
	app := &CmdApp{
		s: do.MustInvoke[*server.SvCmd](i),
		t: do.MustInvoke[*task.TaskCmd](i),
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
