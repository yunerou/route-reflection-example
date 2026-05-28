package eventslogprovider

import (
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/samber/do/v2"

	eventslog "github.com/yunerou/niarb/pkg/events-log"
	"github.com/yunerou/niarb/pkg/events-log/adapters/fileadapter"
	"github.com/yunerou/niarb/pkg/events-log/adapters/pgadapter"
	"github.com/yunerou/niarb/pkg/events-log/adapters/slogadapter"
	fncollector "github.com/yunerou/niarb/provider/fn-collector"
)

type Config struct {
	EnableSlog     bool
	EnablePostgres bool
	EnableFile     bool

	FilePath      string
	AutoMigrate   bool
}

func NewDI(cfg Config) func(i do.Injector) (eventslog.EventsLog, error) {
	return func(i do.Injector) (eventslog.EventsLog, error) {
		cleanupTask := do.MustInvoke[fncollector.CleanupTask](i)

		var adapter eventslog.Adapter

		// Priority: Postgre > File > Slog
		switch {
		case cfg.EnablePostgres:
			pool := do.MustInvoke[*pgxpool.Pool](i)
			adapter = pgadapter.New(pool, cfg.AutoMigrate)

		case cfg.EnableFile:
			fa := fileadapter.New(cfg.FilePath)
			adapter = fa
			cleanupTask.RegTask(func() {
				fa.Flush()
			}, fncollector.FnPriorityNormal)

		case cfg.EnableSlog:
			logger := do.MustInvoke[*slog.Logger](i)
			adapter = slogadapter.New(logger)

		default:
			logger := do.MustInvoke[*slog.Logger](i)
			adapter = slogadapter.New(logger)
		}

		return eventslog.New(adapter), nil
	}
}
