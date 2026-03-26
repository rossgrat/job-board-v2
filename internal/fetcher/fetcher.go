package fetcher

import (
	"context"
	"fmt"
	"time"

	"github.com/rossgrat/job-board-v2/plugin/greenhouse"
	"github.com/rossgrat/job-board-v2/plugin/runner"
)

type Fetcher struct {
	greenhouseClient *greenhouse.Client
	tickerTime       time.Duration
}

func New(options ...FetcherOption) *Fetcher {
	f := &Fetcher{
		tickerTime:       time.Hour * 1,
		greenhouseClient: greenhouse.New(),
	}

	for _, o := range options {
		o(f)
	}

	return f
}

func (f *Fetcher) NewFetcherRunner() runner.RunnerFunc {
	return func(ctx context.Context) func() error {
		return func() error {
			ticker := time.NewTicker(f.tickerTime)
			defer ticker.Stop()

			if err := f.execute(); err != nil {
				return err
			}

			for {
				select {
				case <-ticker.C:
					if err := f.execute(); err != nil {
						return err
					}
				case <-ctx.Done():
					return nil
				}
			}
		}
	}
}

func (f *Fetcher) execute() error {

	fmt.Println("Executing Fetcher")
	return nil
}
