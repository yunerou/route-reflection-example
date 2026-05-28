package valkeyadapter

import (
	"context"

	valkey "github.com/valkey-io/valkey-go"
	"github.com/yunerou/niarb/pkg/cache"
)

type ValkeyConfig struct {
	PrimaryAddress string
	ReplicaAddress string
	Password       string
	DatabaseIndex  int
	KeyPrefix      *string
}

type valkeyAdapter struct {
	primClient valkey.Client
	repClient  valkey.Client

	keyPrefix *string
}

func New(config *ValkeyConfig) cache.StoreAdapter {
	ins := &valkeyAdapter{
		keyPrefix: config.KeyPrefix,
	}

	primConf := valkey.ClientOption{
		InitAddress: []string{config.PrimaryAddress},
		SelectDB:    config.DatabaseIndex,
		Password:    config.Password,
	}

	primClient, err := valkey.NewClient(primConf)
	if err != nil {
		panic(err)
	}
	sendPingToValkey(primClient)

	ins.primClient = primClient

	if config.ReplicaAddress != "" {
		repConf := valkey.ClientOption{
			InitAddress: []string{config.ReplicaAddress},
			SelectDB:    config.DatabaseIndex,
			Password:    config.Password,
		}
		repClient, err := valkey.NewClient(repConf)
		if err != nil {
			panic(err)
		}
		sendPingToValkey(repClient)
		ins.repClient = repClient
	} else {
		ins.repClient = primClient
	}

	return ins
}

func sendPingToValkey(c valkey.Client) {
	err := c.Do(context.Background(),
		c.B().Ping().Build()).
		Error()
	if err != nil {
		panic(err)
	}
}

func (ris *valkeyAdapter) CacheAdapter() {}
