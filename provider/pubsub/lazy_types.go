package pubsub

import (
	"github.com/samber/do/v2"
	"github.com/yunerou/niarb/pkg/pubsub"
)

type (
	PubsubAdapter = func(i do.Injector) (pubsub.Adapter, error)
)
