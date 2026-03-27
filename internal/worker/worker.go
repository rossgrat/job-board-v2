package worker

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rossgrat/job-board-v2/internal/worker/fetcher"
	"github.com/rossgrat/job-board-v2/plugin/runner"
)

var (
	ErrInitFetcher = errors.New("failed to init fetcher")
)

type Worker struct {
	r *runner.Runner
}

func New(ctx context.Context, pool *pgxpool.Pool) (*Worker, error) {
	const fn = "Worker::New"

	// Initialize fetcher
	f, err := fetcher.New(
		pool,
	)
	if err != nil {
		return nil, fmt.Errorf("%s:%w:%w", fn, ErrInitFetcher, err)
	}

	// Triage Worker PoolI
	// Normalization Worker Pool
	// Scoring Worker Pool
	// Liveness Worker Pool

	r := runner.New(
		runner.WithProcess(f.NewFetcherRunner()),
	)
	return &Worker{
		r: r,
	}, nil
}

func (w *Worker) Run(ctx context.Context) {
	w.r.Run()
}
