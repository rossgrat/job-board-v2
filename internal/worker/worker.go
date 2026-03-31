package worker

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rossgrat/job-board-v2/internal/config"
	"github.com/rossgrat/job-board-v2/internal/llm"
	"github.com/rossgrat/job-board-v2/internal/worker/classify"
	"github.com/rossgrat/job-board-v2/internal/worker/constants"
	"github.com/rossgrat/job-board-v2/internal/worker/fetcher"
	"github.com/rossgrat/job-board-v2/internal/worker/filter"
	"github.com/rossgrat/job-board-v2/internal/worker/normalize"
	"github.com/rossgrat/job-board-v2/internal/worker/outbox"
	"github.com/rossgrat/job-board-v2/internal/worker/triage"
	"github.com/rossgrat/job-board-v2/plugin/runner"
	"golang.org/x/time/rate"
)

const (
	llmRateInterval        = 4 * time.Second
	llmRateBurst           = 1
	triageConcurrency      = 1
	normalizeConcurrency   = 1
	filterConcurrency      = 1
	classifyConcurrency    = 1
)

var (
	ErrInitFetcher       = errors.New("failed to init fetcher")
	ErrInitTriageLLM     = errors.New("failed to init triage llm")
	ErrInitNormalizeLLM  = errors.New("failed to init normalize llm")
	ErrInitClassifyLLM   = errors.New("failed to init classify llm")
)

type Worker struct {
	r *runner.Runner
}

func New(ctx context.Context, pool *pgxpool.Pool, cfg *config.Config) (*Worker, error) {
	const fn = "Worker::New"

	// Initialize fetcher
	f, err := fetcher.New(pool)
	if err != nil {
		return nil, fmt.Errorf("%s:%w:%w", fn, ErrInitFetcher, err)
	}

	llmLimiter := rate.NewLimiter(rate.Every(llmRateInterval), llmRateBurst)

	// Initialize triage LLM
	triageLLM, err := llm.New(cfg.Anthropic.APIKey,
		llm.WithRateLimiter(llmLimiter),
		llm.WithSchema(llm.TriageSchema),
	)
	if err != nil {
		return nil, fmt.Errorf("%s:%w:%w", fn, ErrInitTriageLLM, err)
	}

	if err := triageLLM.LoadPrompt("prompts/triage.txt"); err != nil {
		return nil, fmt.Errorf("%s:%w:%w", fn, ErrInitTriageLLM, err)
	}

	// Initialize triage handler + outbox worker
	triageHandler := triage.New(pool, triageLLM)
	triageWorker := outbox.New(pool, constants.PipelineTriage, triageHandler,
		outbox.WithConcurrency(triageConcurrency),
	)

	// Initialize normalize LLM
	normalizeLLM, err := llm.New(cfg.Anthropic.APIKey,
		llm.WithRateLimiter(llmLimiter),
		llm.WithSchema(llm.NormalizeSchema),
	)
	if err != nil {
		return nil, fmt.Errorf("%s:%w:%w", fn, ErrInitNormalizeLLM, err)
	}

	if err := normalizeLLM.LoadPrompt("prompts/normalize.txt"); err != nil {
		return nil, fmt.Errorf("%s:%w:%w", fn, ErrInitNormalizeLLM, err)
	}

	// Initialize normalize handler + outbox worker
	normalizeHandler := normalize.New(pool, normalizeLLM)
	normalizeWorker := outbox.New(pool, constants.PipelineNormalize, normalizeHandler,
		outbox.WithConcurrency(normalizeConcurrency),
	)

	// Initialize filter handler + outbox worker
	filterHandler := filter.New(pool)
	filterWorker := outbox.New(pool, constants.PipelineHardFilter, filterHandler,
		outbox.WithConcurrency(filterConcurrency),
	)

	// Initialize classify LLM
	classifyLLM, err := llm.New(cfg.Anthropic.APIKey,
		llm.WithRateLimiter(llmLimiter),
		llm.WithSchema(llm.ClassifySchema),
	)
	if err != nil {
		return nil, fmt.Errorf("%s:%w:%w", fn, ErrInitClassifyLLM, err)
	}

	if err := classifyLLM.LoadPrompt("prompts/classify.txt"); err != nil {
		return nil, fmt.Errorf("%s:%w:%w", fn, ErrInitClassifyLLM, err)
	}

	// Initialize classify handler + outbox worker
	classifyHandler := classify.New(pool, classifyLLM)
	classifyWorker := outbox.New(pool, constants.PipelineClassify, classifyHandler,
		outbox.WithConcurrency(classifyConcurrency),
	)

	r := runner.New(
		runner.WithProcess(f.NewFetcherRunner()),
		runner.WithProcess(triageWorker.NewOutboxRunner()),
		runner.WithProcess(normalizeWorker.NewOutboxRunner()),
		runner.WithProcess(filterWorker.NewOutboxRunner()),
		runner.WithProcess(classifyWorker.NewOutboxRunner()),
		runner.WithCloser(triageWorker.NewOutboxCloser()),
		runner.WithCloser(normalizeWorker.NewOutboxCloser()),
		runner.WithCloser(filterWorker.NewOutboxCloser()),
		runner.WithCloser(classifyWorker.NewOutboxCloser()),
	)

	return &Worker{r: r}, nil
}

func (w *Worker) Run(ctx context.Context) {
	w.r.Run()
}
