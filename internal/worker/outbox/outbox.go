package outbox

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rossgrat/job-board-v2/database/gen/db"
	"github.com/rossgrat/job-board-v2/internal/worker/constants"
	"github.com/rossgrat/job-board-v2/plugin/runner"
)

var (
	ErrClaimTask     = errors.New("failed to claim outbox task")
	ErrSetTaskStatus = errors.New("failed to set task status")
	ErrMarkDone      = errors.New("failed to mark task done")
	ErrCreateNext    = errors.New("failed to create next outbox task")
	ErrBeginTx       = errors.New("failed to begin transaction")
	ErrCommitTx      = errors.New("failed to commit transaction")
	ErrUpdateRetry   = errors.New("failed to update task for retry")
	ErrCompleteTask  = errors.New("failed to complete task")
)

type TaskHandler interface {
	Handle(ctx context.Context, task db.OutboxTask) (*TaskResult, error)
}

type TaskResult struct {
	NextTaskName string
}

type OutboxWorker struct {
	pool         *pgxpool.Pool
	taskName     string
	handler      TaskHandler
	concurrency  int
	pollInterval time.Duration
	baseBackoff  time.Duration
}

func New(pool *pgxpool.Pool, taskName string, handler TaskHandler, opts ...Option) *OutboxWorker {
	w := &OutboxWorker{
		pool:         pool,
		taskName:     taskName,
		handler:      handler,
		concurrency:  1,
		pollInterval: 2 * time.Second,
		baseBackoff:  30 * time.Second,
	}

	for _, o := range opts {
		o(w)
	}

	return w
}

func (w *OutboxWorker) NewOutboxRunner() runner.RunnerFunc {
	return func(ctx context.Context) func() error {
		return func() error {
			var wg sync.WaitGroup
			for range w.concurrency {
				wg.Add(1)
				go func() {
					defer wg.Done()
					w.pollLoop(ctx)
				}()
			}
			wg.Wait()
			return nil
		}
	}
}

func (w *OutboxWorker) pollLoop(ctx context.Context) {
	for {
		err := w.processOne(ctx)
		if err == nil {
			continue
		}

		if !errors.Is(err, pgx.ErrNoRows) {
			slog.Error("outbox worker error",
				slog.String("task_name", w.taskName),
				slog.String("err", err.Error()))
		}

		select {
		case <-time.After(w.pollInterval):
		case <-ctx.Done():
			return
		}
	}
}

func (w *OutboxWorker) processOne(ctx context.Context) error {
	const fn = "OutboxWorker::processOne"

	task, err := w.claimTask(ctx)
	if err != nil {
		return fmt.Errorf("%s:%w:%w", fn, ErrClaimTask, err)
	}

	taskID := uuid.UUID(task.ID.Bytes).String()

	slog.Info("processing task",
		slog.String("task_name", w.taskName),
		slog.String("task_id", taskID))

	result, err := w.handler.Handle(ctx, task)
	if err != nil {
		w.handleRetry(ctx, task, err)
		return nil
	}

	if err := w.completeTask(ctx, task, result); err != nil {
		return fmt.Errorf("%s:%w:%w", fn, ErrCompleteTask, err)
	}

	return nil
}

func (w *OutboxWorker) claimTask(ctx context.Context) (db.OutboxTask, error) {
	const fn = "OutboxWorker::claimTask"

	tx, err := w.pool.Begin(ctx)
	if err != nil {
		return db.OutboxTask{}, fmt.Errorf("%s:%w:%w", fn, ErrBeginTx, err)
	}
	defer tx.Rollback(ctx)

	qtx := db.New(tx)

	task, err := qtx.ClaimOutboxTask(ctx, w.taskName)
	if err != nil {
		return db.OutboxTask{}, fmt.Errorf("%s:%w:%w", fn, ErrClaimTask, err)
	}

	err = qtx.SetOutboxTaskStatus(ctx, db.SetOutboxTaskStatusParams{
		ID:     task.ID,
		Status: constants.TaskProcessing,
	})
	if err != nil {
		return db.OutboxTask{}, fmt.Errorf("%s:%w:%w", fn, ErrSetTaskStatus, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return db.OutboxTask{}, fmt.Errorf("%s:%w:%w", fn, ErrCommitTx, err)
	}

	return task, nil
}

func (w *OutboxWorker) completeTask(ctx context.Context, task db.OutboxTask, result *TaskResult) error {
	const fn = "OutboxWorker::completeTask"

	tx, err := w.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("%s:%w:%w", fn, ErrBeginTx, err)
	}
	defer tx.Rollback(ctx)

	qtx := db.New(tx)

	err = qtx.SetOutboxTaskStatus(ctx, db.SetOutboxTaskStatusParams{
		ID:     task.ID,
		Status: constants.TaskDone,
	})
	if err != nil {
		return fmt.Errorf("%s:%w:%w", fn, ErrMarkDone, err)
	}

	if result.NextTaskName != "" {
		_, err = qtx.CreateOutboxTask(ctx, db.CreateOutboxTaskParams{
			ID:              pgtype.UUID{Bytes: uuid.Must(uuid.NewV7()), Valid: true},
			ClassifiedJobID: task.ClassifiedJobID,
			TaskName:        result.NextTaskName,
		})
		if err != nil {
			return fmt.Errorf("%s:%w:%w", fn, ErrCreateNext, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("%s:%w:%w", fn, ErrCommitTx, err)
	}

	return nil
}

func (w *OutboxWorker) handleRetry(ctx context.Context, task db.OutboxTask, handlerErr error) {
	const fn = "OutboxWorker::handleRetry"

	taskID := uuid.UUID(task.ID.Bytes).String()
	newRetryCount := task.RetryCount + 1
	status := constants.TaskWaiting
	var notBefore pgtype.Timestamptz

	if newRetryCount >= task.MaxRetries {
		status = constants.TaskFailed
		slog.Error("outbox task max retries exceeded",
			slog.String("task_name", w.taskName),
			slog.String("task_id", taskID),
			slog.String("err", handlerErr.Error()))
	} else {
		backoff := w.baseBackoff * (1 << (newRetryCount - 1))
		notBefore = pgtype.Timestamptz{Time: time.Now().Add(backoff), Valid: true}
		slog.Warn("outbox task retry",
			slog.String("task_name", w.taskName),
			slog.String("task_id", taskID),
			slog.Int("retry_count", int(newRetryCount)),
			slog.String("err", handlerErr.Error()))
	}

	queries := db.New(w.pool)
	err := queries.UpdateOutboxTask(ctx, db.UpdateOutboxTaskParams{
		ID:         task.ID,
		Status:     status,
		RetryCount: newRetryCount,
		NotBefore:  notBefore,
	})
	if err != nil {
		slog.Error(fmt.Errorf("%s:%w:%w", fn, ErrUpdateRetry, err).Error(),
			slog.String("task_name", w.taskName),
			slog.String("task_id", taskID))
	}
}

func (w *OutboxWorker) NewOutboxCloser() runner.RunnerFunc {
	return func(ctx context.Context) func() error {
		return func() error {
			slog.Info("resetting processing tasks", slog.String("task_name", w.taskName))
			queries := db.New(w.pool)
			return queries.ResetProcessingTasks(ctx)
		}
	}
}
