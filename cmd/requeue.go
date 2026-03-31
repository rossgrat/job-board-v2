package cmd

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rossgrat/job-board-v2/database/gen/db"
)

func requeueJobs(ctx context.Context, pool *pgxpool.Pool, ids []pgtype.UUID, taskName string) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}

	queries := db.New(pool)

	type requeue struct {
		oldID    pgtype.UUID
		rawJobID pgtype.UUID
	}
	jobs := make([]requeue, 0, len(ids))
	for _, id := range ids {
		cj, err := queries.GetClassifiedJobByID(ctx, id)
		if err != nil {
			return 0, fmt.Errorf("loading classified job: %w", err)
		}
		jobs = append(jobs, requeue{oldID: id, rawJobID: cj.RawJobID})
	}

	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := db.New(tx)
	for _, j := range jobs {
		err = qtx.ClearCurrentClassifiedJob(ctx, j.oldID)
		if err != nil {
			return 0, fmt.Errorf("clearing current flag: %w", err)
		}

		newJob, err := qtx.CreateClassifiedJob(ctx, db.CreateClassifiedJobParams{
			ID:       pgtype.UUID{Bytes: uuid.Must(uuid.NewV7()), Valid: true},
			RawJobID: j.rawJobID,
		})
		if err != nil {
			return 0, fmt.Errorf("creating classified job: %w", err)
		}

		_, err = qtx.CreateOutboxTask(ctx, db.CreateOutboxTaskParams{
			ID:              pgtype.UUID{Bytes: uuid.Must(uuid.NewV7()), Valid: true},
			ClassifiedJobID: newJob.ID,
			TaskName:        taskName,
		})
		if err != nil {
			return 0, fmt.Errorf("creating outbox task: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit tx: %w", err)
	}

	return len(jobs), nil
}
