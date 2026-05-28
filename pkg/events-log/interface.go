package eventslog

import (
	"context"
)

type EventsLog interface {
	Fire(ctx context.Context, eventName string, eventPayload any) error
}
