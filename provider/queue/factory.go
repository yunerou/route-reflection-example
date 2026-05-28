package queue

import (
	"github.com/samber/do/v2"
	"github.com/yunerou/niarb/pkg/queue"
)

type (
	QueueAdapter = func(i do.Injector) (queue.Adapter, error)
)
