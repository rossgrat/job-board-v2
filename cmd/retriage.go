package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rossgrat/job-board-v2/database/gen/db"
	"github.com/rossgrat/job-board-v2/internal/config"
	"github.com/rossgrat/job-board-v2/internal/worker/constants"
	"github.com/spf13/cobra"
)

var statusFlag string

var retriageCmd = &cobra.Command{
	Use:   "retriage",
	Short: "Re-queue classified jobs for triage",
	RunE:  runRetriage,
}

func init() {
	retriageCmd.Flags().StringVar(&statusFlag, "status", "", "filter by classified job status (required)")
	retriageCmd.MarkFlagRequired("status")
	rootCmd.AddCommand(retriageCmd)
}

func runRetriage(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", slog.String("err", err.Error()))
		os.Exit(1)
	}

	pool, err := pgxpool.New(ctx, cfg.Postgres.DSN())
	if err != nil {
		slog.Error("failed to connect to DB", slog.String("err", err.Error()))
		os.Exit(1)
	}
	defer pool.Close()

	queries := db.New(pool)
	ids, err := queries.ListClassifiedJobIDsByStatus(ctx, statusFlag)
	if err != nil {
		return fmt.Errorf("listing jobs: %w", err)
	}

	if len(ids) == 0 {
		fmt.Printf("no classified jobs with status %q\n", statusFlag)
		return nil
	}

	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	qtx := db.New(tx)
	for _, id := range ids {
		err = qtx.UpdateClassifiedJobStatus(ctx, db.UpdateClassifiedJobStatusParams{
			ID:     id,
			Status: constants.StatusPending,
		})
		if err != nil {
			return fmt.Errorf("resetting job status: %w", err)
		}

		_, err = qtx.CreateOutboxTask(ctx, db.CreateOutboxTaskParams{
			ID:              pgtype.UUID{Bytes: uuid.Must(uuid.NewV7()), Valid: true},
			ClassifiedJobID: id,
			TaskName:        constants.PipelineTriage,
		})
		if err != nil {
			return fmt.Errorf("creating outbox task: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	fmt.Printf("re-queued %d jobs for triage\n", len(ids))
	return nil
}
