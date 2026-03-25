package worker

import (
	"context"

	"github.com/rossgrat/job-board-v2/plugin/runner"
)

type Worker struct {
	r *runner.Runner
}

func New() *Worker {
	// Scheduler
	// Fetcher Worker Pool
	// Triage Worker Pool
	// Normalization Worker Pool
	// Scoring Worker Pool
	// Liveness Worker Pool
	return &Worker{}
}

func (w *Worker) Run(ctx context.Context) {
	w.r.Run()
}
