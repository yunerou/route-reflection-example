package pgadapter

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	eventslog "github.com/yunerou/niarb/pkg/events-log"
)

const createTableSQL = `
CREATE TABLE IF NOT EXISTS events_log (
	id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
	trace_id    TEXT NOT NULL DEFAULT '',
	user_id     TEXT NOT NULL DEFAULT '',
	event_name  TEXT NOT NULL,
	event_payload JSONB NOT NULL DEFAULT '{}',
	fired_at    BIGINT NOT NULL
);
`

type pgAdapter struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool, autoMigrate bool) eventslog.Adapter {
	a := &pgAdapter{pool: pool}

	if autoMigrate {
		if _, err := pool.Exec(context.Background(), createTableSQL); err != nil {
			panic(fmt.Sprintf("events-log pgadapter: failed to create table: %v", err))
		}
	}

	return a
}

func (pa *pgAdapter) Log(ctx context.Context, entry eventslog.EventEntry) error {
	const query = `
		INSERT INTO events_log (trace_id, user_id, event_name, event_payload, fired_at)
		VALUES ($1, $2, $3, $4::jsonb, $5)
	`
	_, err := pa.pool.Exec(ctx, query,
		entry.TraceID,
		entry.UserID,
		entry.EventName,
		entry.EventPayload,
		entry.FiredAt,
	)
	return err
}

func (pa *pgAdapter) Flush() {}
