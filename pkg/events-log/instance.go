package eventslog

import (
	"context"
	"encoding/json"
	"time"

	"github.com/yunerou/niarb/shared/actx"
)

type eventsLog struct {
	adapter Adapter
}

func New(adapter Adapter) EventsLog {
	return &eventsLog{
		adapter: adapter,
	}
}

func (e *eventsLog) Fire(ctx context.Context, eventName string, eventPayload any) error {
	aCtx := actx.From(ctx)

	traceID := aCtx.GetTraceID()

	// var userID string
	// if auth := aCtx.GetAuth(); auth != nil {
	// 	userID = auth.UserID
	// }

	payloadBytes, err := json.Marshal(eventPayload)
	if err != nil {
		return err
	}

	entry := EventEntry{
		TraceID:      traceID,
		UserID:       "",
		EventName:    eventName,
		EventPayload: string(payloadBytes),
		FiredAt:      time.Now().UnixMilli(),
	}

	if err = e.adapter.Log(ctx, entry); err != nil {
		return err
	}

	return nil
}
