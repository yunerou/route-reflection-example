package slogadapter

import (
	"context"
	"log/slog"

	eventslog "github.com/yunerou/niarb/pkg/events-log"
)

type slogAdapter struct {
	logger *slog.Logger
}

func New(logger *slog.Logger) eventslog.Adapter {
	return &slogAdapter{
		logger: logger,
	}
}

func (s *slogAdapter) Log(_ context.Context, entry eventslog.EventEntry) error {
	s.logger.Info("event_fired",
		slog.String("trace_id", entry.TraceID),
		slog.String("user_id", entry.UserID),
		slog.String("event_name", entry.EventName),
		slog.String("event_payload", entry.EventPayload),
		slog.Int64("fired_at", entry.FiredAt),
	)
	return nil
}

func (s *slogAdapter) Flush() {}
