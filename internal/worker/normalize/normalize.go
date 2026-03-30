package normalize

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rossgrat/job-board-v2/database/gen/db"
	"github.com/rossgrat/job-board-v2/internal/llm"
	"github.com/rossgrat/job-board-v2/internal/worker/constants"
	"github.com/rossgrat/job-board-v2/internal/worker/outbox"
)

var (
	ErrLoadClassifiedJob = errors.New("failed to load classified job")
	ErrLoadRawJob        = errors.New("failed to load raw job")
	ErrLLMCall           = errors.New("llm call failed")
	ErrBeginTx           = errors.New("failed to begin transaction")
	ErrCommitTx          = errors.New("failed to commit transaction")
	ErrUpdateNorm        = errors.New("failed to update normalization fields")
	ErrCreateLocation    = errors.New("failed to create location")
	ErrCreateTechnology  = errors.New("failed to create technology")
)

type NormalizeResponse struct {
	Title        string     `json:"title"`
	Locations    []Location `json:"locations"`
	Technologies []string   `json:"technologies"`
	SalaryMin    *int32     `json:"salary_min"`
	SalaryMax    *int32     `json:"salary_max"`
	Level        string     `json:"level"`
}

type Location struct {
	Country string  `json:"country"`
	City    *string `json:"city"`
	Setting string  `json:"setting"`
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
	const fn = "NormalizeHandler::Handle"

	queries := db.New(h.pool)

	classifiedJob, err := queries.GetClassifiedJobByID(ctx, task.ClassifiedJobID)
	if err != nil {
		return nil, fmt.Errorf("%s:%w:%w", fn, ErrLoadClassifiedJob, err)
	}

	rawJob, err := queries.GetRawJobByID(ctx, classifiedJob.RawJobID)
	if err != nil {
		return nil, fmt.Errorf("%s:%w:%w", fn, ErrLoadRawJob, err)
	}

	result, err := llm.Complete[NormalizeResponse](ctx, h.llm, rawJob.CleanData)
	if err != nil {
		return nil, fmt.Errorf("%s:%w:%w", fn, ErrLLMCall, err)
	}

	// Write normalization data in a transaction
	tx, err := h.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("%s:%w:%w", fn, ErrBeginTx, err)
	}
	defer tx.Rollback(ctx)

	qtx := db.New(tx)

	err = qtx.UpdateClassifiedJobNormalization(ctx, db.UpdateClassifiedJobNormalizationParams{
		ID:        task.ClassifiedJobID,
		Title:     pgtype.Text{String: result.Title, Valid: result.Title != ""},
		SalaryMin: toPgInt4(result.SalaryMin),
		SalaryMax: toPgInt4(result.SalaryMax),
		Level:     pgtype.Text{String: result.Level, Valid: result.Level != ""},
	})
	if err != nil {
		return nil, fmt.Errorf("%s:%w:%w", fn, ErrUpdateNorm, err)
	}

	for _, loc := range result.Locations {
		err = qtx.CreateClassifiedJobLocation(ctx, db.CreateClassifiedJobLocationParams{
			ID:              pgtype.UUID{Bytes: uuid.Must(uuid.NewV7()), Valid: true},
			ClassifiedJobID: task.ClassifiedJobID,
			Country:         loc.Country,
			City:            toPgText(loc.City),
			Setting:         loc.Setting,
		})
		if err != nil {
			return nil, fmt.Errorf("%s:%w:%w", fn, ErrCreateLocation, err)
		}
	}

	for _, tech := range result.Technologies {
		err = qtx.CreateClassifiedJobTechnology(ctx, db.CreateClassifiedJobTechnologyParams{
			ID:              pgtype.UUID{Bytes: uuid.Must(uuid.NewV7()), Valid: true},
			ClassifiedJobID: task.ClassifiedJobID,
			Name:            tech,
		})
		if err != nil {
			return nil, fmt.Errorf("%s:%w:%w", fn, ErrCreateTechnology, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("%s:%w:%w", fn, ErrCommitTx, err)
	}

	return &outbox.TaskResult{NextTaskName: constants.PipelineHardFilter}, nil
}

func toPgInt4(v *int32) pgtype.Int4 {
	if v == nil {
		return pgtype.Int4{}
	}
	return pgtype.Int4{Int32: *v, Valid: true}
}

func toPgText(v *string) pgtype.Text {
	if v == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *v, Valid: true}
}
