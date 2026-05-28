package pubsub

import (
	"github.com/samber/do/v2"
	"github.com/yunerou/niarb/pkg/pubsub"
	"github.com/yunerou/niarb/pkg/pubsub/adapters/valkey"
	configprovider "github.com/yunerou/niarb/provider/config-provider"
	fncollector "github.com/yunerou/niarb/provider/fn-collector"
	"github.com/yunerou/niarb/shared/constants"
)

func PubsubValkeyDI(i do.Injector) (pubsub.Adapter, error) {
	c := do.MustInvoke[configprovider.ConfigStore](i)
	cleanupTask := do.MustInvoke[fncollector.CleanupTask](i)

	return pubsubValkey(c, cleanupTask), nil
}

func pubsubValkey(c configprovider.ConfigStore, cleanupTask fncollector.CleanupTask) pubsub.Adapter {
	env := c.Env()
	ins := valkey.New(&valkey.Config{
		ValkeyEndpoint: env.Valkey.PrimaryAddress,
		Password:       env.Valkey.Password,
		DB:             env.Valkey.DatabaseIdx,
		Prefix:         constants.AppNameShort + env.Info.Env,
	})

	// Register cleanup task
	cleanupTask.RegTask(func() {
		ins.Flush()
	}, fncollector.FnPriorityNormal)

	return ins
}
