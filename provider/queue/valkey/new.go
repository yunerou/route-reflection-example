package queue

import (
	"github.com/samber/do/v2"
	"github.com/yunerou/niarb/pkg/queue"
	"github.com/yunerou/niarb/pkg/queue/adapters/valkey"
	configprovider "github.com/yunerou/niarb/provider/config-provider"
	fncollector "github.com/yunerou/niarb/provider/fn-collector"
	"github.com/yunerou/niarb/shared/constants"
)

func QueueValkeyDI(i do.Injector) (queue.Adapter, error) {
	c := do.MustInvoke[configprovider.ConfigStore](i)
	cleanupTask := do.MustInvoke[fncollector.CleanupTask](i)

	return queueValkey(c, cleanupTask), nil
}

func queueValkey(c configprovider.ConfigStore, cleanupTask fncollector.CleanupTask) queue.Adapter {
	env := c.Env()
	ins := valkey.New(&valkey.Config{
		ValkeyEndpoint: env.Valkey.PrimaryAddress,
		Password:       env.Valkey.Password,
		DB:             env.Valkey.DatabaseIdx,
		Prefix:         constants.AppNameShort + env.Info.Env,
	})

	// Register cleanup task
	cleanupTask.RegTask(func() {
		ins.Close()
	}, fncollector.FnPriorityNormal)

	return ins
}
