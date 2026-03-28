package triage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rossgrat/job-board-v2/database/gen/db"
	"github.com/rossgrat/job-board-v2/internal/llm"
	"github.com/rossgrat/job-board-v2/internal/worker/constants"
	"github.com/rossgrat/job-board-v2/internal/worker/outbox"
)

var (
	ErrLoadClassifiedJob   = errors.New("failed to load classified job")
	ErrLoadRawJob          = errors.New("failed to load raw job")
	ErrParseRawData        = errors.New("failed to parse raw data")
	ErrLLMCall             = errors.New("llm call failed")
	ErrUpdateClassifiedJob = errors.New("failed to update classified job status")
)

type TriageResponse struct {
	IsTechnical bool `json:"is_technical"`
}

type Handler struct {
	pool *pgxpool.Pool
	llm  *llm.Client
}

func New(pool *pgxpool.Pool, llm *llm.Client) *Handler {
	return &Handler{
		pool: pool,
		llm:  llm,
	}
}

func (h *Handler) Handle(ctx context.Context, task db.OutboxTask) (*outbox.TaskResult, error) {
	const fn = "TriageHandler::Handle"

	queries := db.New(h.pool)

	classifiedJob, err := queries.GetClassifiedJobByID(ctx, task.ClassifiedJobID)
	if err != nil {
		return nil, fmt.Errorf("%s:%w:%w", fn, ErrLoadClassifiedJob, err)
	}

	rawJob, err := queries.GetRawJobByID(ctx, classifiedJob.RawJobID)
	if err != nil {
		return nil, fmt.Errorf("%s:%w:%w", fn, ErrLoadRawJob, err)
	}

	// Extract only title and content for triage — full raw data is too large
	var raw struct {
		Title   string `json:"title"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(rawJob.RawData, &raw); err != nil {
		return nil, fmt.Errorf("%s:%w:%w", fn, ErrParseRawData, err)
	}

	triageInput := fmt.Sprintf("Title: %s\n\nDescription: %s", raw.Title, raw.Content)
	result, err := llm.Complete[TriageResponse](ctx, h.llm, triageInput)
	if err != nil {
		return nil, fmt.Errorf("%s:%w:%w", fn, ErrLLMCall, err)
	}

	if result.IsTechnical {
		return &outbox.TaskResult{NextTaskName: constants.PipelineNormalize}, nil
	}

	err = queries.UpdateClassifiedJobStatus(ctx, db.UpdateClassifiedJobStatusParams{
		ID:     task.ClassifiedJobID,
		Status: constants.StatusNonTechnical,
	})
	if err != nil {
		return nil, fmt.Errorf("%s:%w:%w", fn, ErrUpdateClassifiedJob, err)
	}

	return &outbox.TaskResult{}, nil
}
