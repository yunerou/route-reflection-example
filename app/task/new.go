package task

import (
	"strings"

	"github.com/samber/do/v2"
	"github.com/urfave/cli/v2"
)

func NewTaskCmd(i do.Injector) *TaskCmd {
	return &TaskCmd{}
}

type TaskCmd struct {
}

func (c *TaskCmd) Commands() []*cli.Command {
	return []*cli.Command{
		{
			Name:  "check_panic",
			Usage: "booom panic for testing",
			Action: func(ctx *cli.Context) error {
				args := ctx.Args().Slice()

				r := strings.Join(args, " ")
				panic(r)
			},
		},
	}
}
