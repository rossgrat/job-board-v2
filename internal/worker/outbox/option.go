package outbox

import "time"

type Option func(*OutboxWorker)

func WithConcurrency(n int) Option {
	return func(w *OutboxWorker) {
		w.concurrency = n
	}
}

func WithPollInterval(d time.Duration) Option {
	return func(w *OutboxWorker) {
		w.pollInterval = d
	}
}

func WithBaseBackoff(d time.Duration) Option {
	return func(w *OutboxWorker) {
		w.baseBackoff = d
	}
}
