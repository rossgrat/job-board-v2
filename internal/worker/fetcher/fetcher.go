package fetcher

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
	"github.com/rossgrat/job-board-v2/internal/model"
	"github.com/rossgrat/job-board-v2/internal/telemetry"
	"github.com/rossgrat/job-board-v2/internal/worker/constants"
	"github.com/rossgrat/job-board-v2/internal/worker/fetcher/atomfeed"
	"github.com/rossgrat/job-board-v2/internal/worker/fetcher/greenhouse"
	"github.com/rossgrat/job-board-v2/internal/worker/fetcher/workday"
	"github.com/rossgrat/job-board-v2/plugin/runner"
)

var (
	ErrEmptyDB               = errors.New("must configure a db")
	ErrFailedToLoadCompanies = errors.New("failed to load companies")
	ErrBeginTx               = errors.New("failed to begin transaction")
	ErrCreateRawJob          = errors.New("failed to create raw job")
	ErrCreateClassifiedJob   = errors.New("failed to create classified job")
	ErrCreateOutboxTask      = errors.New("failed to create outbox task")
	ErrCommitTx              = errors.New("failed to commit transaction")
)

type fetcherClient interface {
	GetJobs(context.Context, uuid.UUID, []byte, chan<- model.RawJob) error
}

type Fetcher struct {
	pool       *pgxpool.Pool
	tickerTime time.Duration
	clientsMap map[JobBoardName]fetcherClient
}

func New(pool *pgxpool.Pool, opts ...FetcherOption) (*Fetcher, error) {
	const fn = "Fetcher::New"

	f := &Fetcher{
		pool:       pool,
		tickerTime: time.Hour * 1,
		clientsMap: map[JobBoardName]fetcherClient{
			"atomfeed":   atomfeed.New(),
			"greenhouse": greenhouse.New(),
			"workday":    workday.New(),
		},
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

	// Load all active companies
	queries := db.New(f.pool)
	companies, err := queries.GetActiveCompanies(ctx)
	if err != nil {
		slog.Error("failed to load companies", slog.String("err", err.Error()))
		return fmt.Errorf("%s:%w:%w", fn, ErrFailedToLoadCompanies, err)
	}

	var wg sync.WaitGroup
	for _, company := range companies {
		fetcherClient := f.clientsMap[JobBoardName(company.FetchType)]
		wg.Go(func() {
			f.fetchCompany(ctx, queries, company, fetcherClient)
		})
	}

	wg.Wait()

	return nil
}

func (f *Fetcher) fetchCompany(ctx context.Context, queries *db.Queries, company db.Company, client fetcherClient) {
	jobs := make(chan model.RawJob)
	errCh := make(chan error, 1)
	go func() {
		defer close(jobs)
		errCh <- client.GetJobs(ctx, company.ID.Bytes, company.FetchConfig, jobs)
	}()

	count, seenSourceJobIDs := f.saveJobs(ctx, company.Name, jobs)

	fetchErr := <-errCh
	if fetchErr != nil {
		slog.Error("failed to load jobs for company",
			slog.String("err", fetchErr.Error()),
			slog.String("company", company.Name))
		telemetry.RecordFetchError(ctx, company.Name)
	}

	if fetchErr == nil && len(seenSourceJobIDs) > 0 {
		f.softDeleteMissing(ctx, queries, company, seenSourceJobIDs)
	}

	telemetry.RecordJobsFetched(ctx, company.Name, count)
	slog.Info(fmt.Sprintf("loaded %d jobs for %s", count, company.Name))
}

func (f *Fetcher) saveJobs(ctx context.Context, companyName string, jobs <-chan model.RawJob) (int64, []string) {
	var count int64
	var seenSourceJobIDs []string
	for job := range jobs {
		count++
		seenSourceJobIDs = append(seenSourceJobIDs, job.SourceJobID)
		if err := f.SaveJob(ctx, job); err != nil {
			slog.Error("failed to save job",
				slog.String("err", err.Error()),
				slog.String("jobID", job.SourceJobID),
				slog.String("company", companyName))
			telemetry.RecordFetchError(ctx, companyName)
		} else {
			telemetry.RecordJobSaved(ctx, companyName)
		}
	}
	return count, seenSourceJobIDs
}

func (f *Fetcher) softDeleteMissing(ctx context.Context, queries *db.Queries, company db.Company, seenSourceJobIDs []string) {
	deleted, err := queries.SoftDeleteMissingJobs(ctx, db.SoftDeleteMissingJobsParams{
		CompanyID:        company.ID,
		SeenSourceJobIds: seenSourceJobIDs,
	})
	if err != nil {
		slog.Error("failed to soft-delete missing jobs",
			slog.String("err", err.Error()),
			slog.String("company", company.Name))
	} else if deleted > 0 {
		slog.Info(fmt.Sprintf("soft-deleted %d jobs for %s", deleted, company.Name))
	}
}

func (f *Fetcher) SaveJob(ctx context.Context, rawJob model.RawJob) error {
	const fn = "Fetcher::SaveJob"

	tx, err := f.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("%s:%w:%w", fn, ErrBeginTx, err)
	}
	defer tx.Rollback(ctx)

	qtx := db.New(tx)

	newID := pgtype.UUID{Bytes: uuid.Must(uuid.NewV7()), Valid: true}
	newRawJob, err := qtx.CreateRawJob(ctx, db.CreateRawJobParams{
		ID:          newID,
		CompanyID:   pgtype.UUID{Bytes: rawJob.CompanyID, Valid: true},
		SourceJobID: rawJob.SourceJobID,
		Url:         rawJob.URL,
		RawData:     rawJob.RawData,
		CleanData:   rawJob.CleanData,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil // duplicate job, skip
		}
		return fmt.Errorf("%s:%w:%w", fn, ErrCreateRawJob, err)
	}

	// ON CONFLICT with WHERE deleted_at IS NOT NULL returns the existing row
	// when un-deleting. Commit to persist deleted_at = NULL, skip pipeline.
	if newRawJob.ID != newID {
		return tx.Commit(ctx)
	}

	classifiedJob, err := qtx.CreateClassifiedJob(ctx, db.CreateClassifiedJobParams{
		ID:       pgtype.UUID{Bytes: uuid.Must(uuid.NewV7()), Valid: true},
		RawJobID: newRawJob.ID,
	})
	if err != nil {
		return fmt.Errorf("%s:%w:%w", fn, ErrCreateClassifiedJob, err)
	}

	_, err = qtx.CreateOutboxTask(ctx, db.CreateOutboxTaskParams{
		ID:              pgtype.UUID{Bytes: uuid.Must(uuid.NewV7()), Valid: true},
		ClassifiedJobID: classifiedJob.ID,
		TaskName:        constants.PipelineTriage,
	})
	if err != nil {
		return fmt.Errorf("%s:%w:%w", fn, ErrCreateOutboxTask, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("%s:%w:%w", fn, ErrCommitTx, err)
	}

	return nil
}
