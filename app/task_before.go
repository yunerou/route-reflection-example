package app

import (
	"github.com/urfave/cli/v2"
	"github.com/yunerou/niarb/shared/actx"
	"github.com/yunerou/niarb/shared/utils/fn"
)

func taskBefore() cli.BeforeFunc {
	return func(c *cli.Context) error {
		ctx := c.Context
		aCtx := actx.From(ctx)

		// Set request trace id
		requestTraceID := fn.NewNanoID()
		aCtx.SetTraceID(requestTraceID)

		c.Context = aCtx

		return nil
	}
}
