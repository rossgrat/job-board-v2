package cleanup

import (
	"context"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rossgrat/job-board-v2/database/gen/db"
	"github.com/rossgrat/job-board-v2/plugin/runner"
)

type Worker struct {
	pool      *pgxpool.Pool
	retention time.Duration
	interval  time.Duration
}

func New(pool *pgxpool.Pool) *Worker {
	return &Worker{
		pool:      pool,
		retention: 72 * time.Hour,
		interval:  1 * time.Hour,
	}
}

func (w *Worker) NewCleanupRunner() runner.RunnerFunc {
	return func(ctx context.Context) func() error {
		return func() error {
			ticker := time.NewTicker(w.interval)
			defer ticker.Stop()

			w.execute(ctx)

			for {
				select {
				case <-ticker.C:
					w.execute(ctx)
				case <-ctx.Done():
					return nil
				}
			}
		}
	}
}

func (w *Worker) execute(ctx context.Context) {
	cutoff := pgtype.Timestamptz{Time: time.Now().Add(-w.retention), Valid: true}
	queries := db.New(w.pool)

	jobs, err := queries.DeleteStaleClassifiedJobs(ctx, cutoff)
	if err != nil {
		slog.Error("cleanup: failed to delete stale classified jobs", slog.String("err", err.Error()))
	} else if jobs > 0 {
		slog.Info("cleanup: deleted stale classified jobs", slog.Int64("count", jobs))
	}

	tasks, err := queries.DeleteCompletedOutboxTasks(ctx, cutoff)
	if err != nil {
		slog.Error("cleanup: failed to delete completed outbox tasks", slog.String("err", err.Error()))
	} else if tasks > 0 {
		slog.Info("cleanup: deleted completed outbox tasks", slog.Int64("count", tasks))
	}
}
