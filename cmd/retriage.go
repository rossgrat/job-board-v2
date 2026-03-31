package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rossgrat/job-board-v2/database/gen/db"
	"github.com/rossgrat/job-board-v2/internal/config"
	"github.com/rossgrat/job-board-v2/internal/worker/constants"
	"github.com/spf13/cobra"
)

var retriageStatusFlag string

var retriageCmd = &cobra.Command{
	Use:   "retriage",
	Short: "Re-queue classified jobs for triage",
	RunE:  runRetriage,
}

func init() {
	retriageCmd.Flags().StringVar(&retriageStatusFlag, "status", "", "filter by classified job status (required)")
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

	ids, err := db.New(pool).ListClassifiedJobIDsByStatus(ctx, retriageStatusFlag)
	if err != nil {
		return fmt.Errorf("listing jobs: %w", err)
	}

	n, err := requeueJobs(ctx, pool, ids, constants.PipelineTriage)
	if err != nil {
		return err
	}

	fmt.Printf("re-queued %d jobs for triage\n", n)
	return nil
}
