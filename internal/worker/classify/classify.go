package classify

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rossgrat/job-board-v2/database/gen/db"
	"github.com/rossgrat/job-board-v2/database/pgutil"
	"github.com/rossgrat/job-board-v2/internal/llm"
	"github.com/rossgrat/job-board-v2/internal/worker/constants"
	"github.com/rossgrat/job-board-v2/internal/worker/outbox"
)

var (
	ErrLoadClassifiedJob    = errors.New("failed to load classified job")
	ErrLoadRawJob           = errors.New("failed to load raw job")
	ErrLLMCall              = errors.New("llm call failed")
	ErrUpdateClassification = errors.New("failed to update classification")
	ErrUpdateStatus         = errors.New("failed to update status")
)

type ClassifyResponse struct {
	Category  string  `json:"category"`
	Relevance *string `json:"relevance"`
	Reasoning string  `json:"reasoning"`
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
	const fn = "ClassifyHandler::Handle"

	queries := db.New(h.pool)

	classifiedJob, err := queries.GetClassifiedJobByID(ctx, task.ClassifiedJobID)
	if err != nil {
		return nil, fmt.Errorf("%s:%w:%w", fn, ErrLoadClassifiedJob, err)
	}

	rawJob, err := queries.GetRawJobByID(ctx, classifiedJob.RawJobID)
	if err != nil {
		return nil, fmt.Errorf("%s:%w:%w", fn, ErrLoadRawJob, err)
	}

	result, err := llm.Complete[ClassifyResponse](ctx, h.llm, rawJob.CleanData)
	if err != nil {
		return nil, fmt.Errorf("%s:%w:%w", fn, ErrLLMCall, err)
	}

	err = queries.UpdateClassifiedJobClassification(ctx, db.UpdateClassifiedJobClassificationParams{
		ID:                          task.ClassifiedJobID,
		Category:                    pgtype.Text{String: result.Category, Valid: true},
		Relevance:                   pgutil.ToPgText(result.Relevance),
		Reasoning:                   pgtype.Text{String: result.Reasoning, Valid: true},
		ClassificationPromptVersion: pgtype.Text{String: h.llm.PromptHash(), Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("%s:%w:%w", fn, ErrUpdateClassification, err)
	}

	if result.Category == "not_relevant" || (result.Relevance != nil && *result.Relevance == "weak_match") {
		err = queries.UpdateClassifiedJobStatus(ctx, db.UpdateClassifiedJobStatusParams{
			ID:     task.ClassifiedJobID,
			Status: constants.StatusFilteredRelevance,
		})
		if err != nil {
			return nil, fmt.Errorf("%s:%w:%w", fn, ErrUpdateStatus, err)
		}
		return &outbox.TaskResult{}, nil
	}

	err = queries.UpdateClassifiedJobStatus(ctx, db.UpdateClassifiedJobStatusParams{
		ID:     task.ClassifiedJobID,
		Status: constants.StatusAccepted,
	})
	if err != nil {
		return nil, fmt.Errorf("%s:%w:%w", fn, ErrUpdateStatus, err)
	}

	return &outbox.TaskResult{}, nil
}
