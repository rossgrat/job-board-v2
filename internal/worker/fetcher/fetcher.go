package fetcher

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rossgrat/job-board-v2/database/gen/db"
	"github.com/rossgrat/job-board-v2/internal/model"
	"github.com/rossgrat/job-board-v2/internal/worker/fetcher/greenhouse"
	"github.com/rossgrat/job-board-v2/plugin/runner"
)

var (
	ErrEmptyDB               = errors.New("must configure a db")
	ErrFailedToLoadCompanies = errors.New("failed to load companies")
)

type fetcherClient interface {
	GetJobs(context.Context, uuid.UUID, []byte) ([]model.RawJob, error)
}

type Fetcher struct {
	db         *db.Queries
	tickerTime time.Duration
	clientsMap map[JobBoardName]fetcherClient
}

func New(db *db.Queries, opts ...FetcherOption) (*Fetcher, error) {
	const fn = "Fetcher::New"

	f := &Fetcher{
		tickerTime: time.Hour * 1,
		clientsMap: map[JobBoardName]fetcherClient{
			"greenhouse": greenhouse.New(),
		},
		db: db,
	}

	for _, o := range opts {
		o(f)
	}

	return f, nil
}

func (f *Fetcher) NewFetcherRunner() runner.RunnerFunc {
	return func(ctx context.Context) func() error {
		return func() error {
			ticker := time.NewTicker(f.tickerTime)
			defer ticker.Stop()

			f.execute(ctx)

			for {
				select {
				case <-ticker.C:
					f.execute(ctx) // Intentionally not stopping worker with errors
				case <-ctx.Done():
					return nil
				}
			}
		}
	}
}

func (f *Fetcher) execute(ctx context.Context) error {
	const fn = "Fetcher::execute"

	companies, err := f.db.GetActiveCompanies(ctx)
	if err != nil {
		slog.Error("failed to load companies", slog.String("err", err.Error()))
		return fmt.Errorf("%s:%w:%w", fn, ErrFailedToLoadCompanies, err)
	}

	var wg sync.WaitGroup
	for _, company := range companies {
		fetcherClient := f.clientsMap[JobBoardName(company.FetchType)]
		wg.Go(func() {
			jobs, err := fetcherClient.GetJobs(ctx, company.ID.Bytes, company.FetchConfig)
			if err != nil {
				slog.Error("failed to load jobs for company", slog.String("err", err.Error()))
			}

			fmt.Println("loaded", len(jobs), "jobs")
		})
	}

	wg.Wait()

	// For each company
	// 	Dispatch Fetcher based on job board name

	fmt.Println("Executing Fetcher")
	return nil
}
