package eventslog

import (
	"context"
)

type Adapter interface {
	Log(ctx context.Context, entry EventEntry) error
	Flush()
}

type EventEntry struct {
	TraceID      string
	UserID       string
	EventName    string
	EventPayload string // JSON-encoded payload
	FiredAt      int64  // Unix milliseconds
}
