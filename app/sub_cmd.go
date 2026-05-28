package app

import (
	"github.com/urfave/cli/v2"
)

func (app *CmdApp) subCommand() []*cli.Command {
	return []*cli.Command{
		{
			Name:        "server",
			Usage:       "run server",
			Subcommands: app.s.Commands(),
		},
		{
			Name:        "task",
			Usage:       "run task such as cronjob",
			Before:      taskBefore(),
			Subcommands: app.t.Commands(),
		},
	}
}
