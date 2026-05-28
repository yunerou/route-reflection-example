package cacheprovider

import (
	"github.com/samber/do/v2"
	"github.com/samber/lo"
	"github.com/yunerou/niarb/pkg/cache"
	valkeyadapter "github.com/yunerou/niarb/pkg/cache/adapters/valkey"
	configprovider "github.com/yunerou/niarb/provider/config-provider"
	fncollector "github.com/yunerou/niarb/provider/fn-collector"
	"github.com/yunerou/niarb/shared/constants"
)

func NewDI(i do.Injector) (cache.CacheProvider, error) {
	cfg := do.MustInvoke[configprovider.ConfigStore](i)
	cleanupTask := do.MustInvoke[fncollector.CleanupTask](i)
	env := cfg.Env()

	cachePrv := cache.New(valkeyadapter.New(&valkeyadapter.ValkeyConfig{
		PrimaryAddress: env.Valkey.PrimaryAddress,
		ReplicaAddress: env.Valkey.ReplicaAddress,
		Password:       env.Valkey.Password,
		DatabaseIndex:  env.Valkey.DatabaseIdx,
		KeyPrefix:      lo.ToPtr(constants.AppNameShort + env.Info.Env),
	}))

	// Register cleanup task
	cleanupTask.RegTask(func() {
		cachePrv.Flush()
	}, fncollector.FnPriorityNormal)

	return cachePrv, nil
}
