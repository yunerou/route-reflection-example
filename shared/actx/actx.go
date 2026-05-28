package actx

import (
	"context"
	"sync"

	"github.com/yunerou/niarb/shared/encdec"
)

type alterData struct {
	m sync.Mutex

	name string

	encoder encdec.Encoder
	decoder encdec.Decoder

	traceId       string
	parentTraceId []string

	fromBroadcast bool

	userIp    string
	userAgent string
}
type aContext struct {
	context.Context
	data *alterData
}

type AContext = *aContext

func From(ctx context.Context) AContext {
	if ctx == nil {
		ctx = context.Background()
	}

	aData, ok := ctx.Value(privKey).(*alterData)
	if ok {
		return &aContext{
			Context: ctx,
			data:    aData,
		}
	} else {
		newAData := alterData{
			m:       sync.Mutex{},
			userIp:  "",
			encoder: nil,
			decoder: nil,
		}
		ctx = context.WithValue(ctx, privKey, &newAData)
		return &aContext{
			Context: ctx,
			data:    &newAData,
		}
	}
}

func New() AContext {
	ctx := context.Background()

	return From(ctx)
}

type aCtxKey int

const (
	_ aCtxKey = iota + 1
	privKey
)
