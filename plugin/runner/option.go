package runner

import (
	"time"
)

type RunnerOption func(*Runner)

func WithProcess(rf RunnerFunc) RunnerOption {
	return func(r *Runner) {
		r.processes = append(r.processes, rf)
	}
}

func WithCloser(rf RunnerFunc) RunnerOption {
	return func(r *Runner) {
		r.closers = append(r.closers, rf)
	}
}

func WithCloserTimeout(timeout time.Duration) RunnerOption {
	return func(r *Runner) {
		r.closerTimeout = timeout
	}
}
