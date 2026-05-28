package server

import (
	"github.com/samber/do/v2"
	"github.com/urfave/cli/v2"
	exampleDeli "github.com/yunerou/niarb/core/domain/example/delivery"
	muxmiddleware "github.com/yunerou/niarb/pkg/mux-middleware"
	configprovider "github.com/yunerou/niarb/provider/config-provider"
	fncollector "github.com/yunerou/niarb/provider/fn-collector"
)

func NewSvCmd(i do.Injector) *SvCmd {
	return &SvCmd{
		c:              do.MustInvoke[configprovider.ConfigStore](i),
		mw:             do.MustInvoke[muxmiddleware.MiddlewareProvider](i),
		exampleHandler: do.MustInvoke[exampleDeli.ExampleHandler](i),
		cleanupTask:    do.MustInvoke[fncollector.CleanupTask](i),
	}
}

type SvCmd struct {
	c  configprovider.ConfigStore
	mw muxmiddleware.MiddlewareProvider

	exampleHandler exampleDeli.ExampleHandler

	cleanupTask fncollector.CleanupTask
}

func (c *SvCmd) Commands() []*cli.Command {
	return []*cli.Command{
		{
			Name:  "public",
			Usage: "run public server at port (default: 8080)",
			Flags: []cli.Flag{
				&cli.IntFlag{
					Name:    "port",
					Aliases: []string{"p"},
					Usage:   "port to run server",
					Value:   8080,
				},
			},
			Action: func(ctx *cli.Context) error {
				return c.RunServerWithPort(ctx.Context, Public, ctx.Int("port"))
			},
		},
		{
			Name:  "private",
			Usage: "run private server at port (default: 8081)",
			Flags: []cli.Flag{
				&cli.IntFlag{
					Name:    "port",
					Aliases: []string{"p"},
					Usage:   "port to run server",
					Value:   8081,
				},
			},
			Action: func(ctx *cli.Context) error {
				return c.RunServerWithPort(ctx.Context, Private, ctx.Int("port"))
			},
		},
	}
}
