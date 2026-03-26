package runner

// Runner Plugin is based on fetch-rewards/runner by @srall (Simon Rall)

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"
)

type RunnerFunc func(context.Context) func() error

type Runner struct {
	logger          *slog.Logger
	processes       []RunnerFunc
	closers         []RunnerFunc
	shutdownSignals []os.Signal
	closerTimeout   time.Duration
}

func New(options ...RunnerOption) *Runner {
	r := &Runner{
		logger:          slog.Default(),
		shutdownSignals: []os.Signal{os.Interrupt, syscall.SIGTERM},
		closerTimeout:   time.Second * 10,
	}

	for _, o := range options {
		o(r)
	}

	return r
}

func (r *Runner) Run() {
	r.logger.Info("runner starting up")

	// Stop all processes on any shutdown signal
	stopProcessesCtx, stopProcessesCtxCancel := signal.NotifyContext(
		context.Background(),
		r.shutdownSignals...)
	defer stopProcessesCtxCancel()

	// Start all runner processes
	rg, rgCtx := errgroup.WithContext(stopProcessesCtx)
	for _, p := range r.processes {
		rg.Go(p(rgCtx))
	}

	// Cancel all processes if one process errors
	err := rg.Wait()
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			r.logger.Error("app process crashed", slog.String("err", err.Error()))
		}
	}

	r.logger.Info("starting shutdown")

	// Closers must complete within time window
	closerCtx, closerCtxCancel := context.WithTimeout(
		context.Background(),
		r.closerTimeout)
	defer closerCtxCancel()
	cg, cgCtx := errgroup.WithContext(closerCtx)

	for _, c := range r.closers {
		cg.Go(c(cgCtx))
	}

	err = cg.Wait()
	if err != nil {
		r.logger.Error("closers did not complete correctly",
			slog.String("err", err.Error()))
	}

	r.logger.Info("goodbye")
}
