package worker

import (
	"context"

	"github.com/rossgrat/job-board-v2/internal/fetcher"
	"github.com/rossgrat/job-board-v2/plugin/runner"
)

type Worker struct {
	r *runner.Runner
}

func New() *Worker {

	// Initialize fetcher
	f := fetcher.New()

	// Triage Worker Pool
	// Normalization Worker Pool
	// Scoring Worker Pool
	// Liveness Worker Pool

	r := runner.New(
		runner.WithProcess(f.NewFetcherRunner()),
	)
	return &Worker{
		r: r,
	}
}

func (w *Worker) Run(ctx context.Context) {
	w.r.Run()
}
