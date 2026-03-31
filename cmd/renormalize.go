package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rossgrat/job-board-v2/database/gen/db"
	"github.com/rossgrat/job-board-v2/internal/config"
	"github.com/rossgrat/job-board-v2/internal/worker/constants"
	"github.com/spf13/cobra"
)

var renormalizeStatusFlag string

var renormalizeCmd = &cobra.Command{
	Use:   "renormalize",
	Short: "Re-queue classified jobs for normalization",
	RunE:  runRenormalize,
}

func init() {
	renormalizeCmd.Flags().StringVar(&renormalizeStatusFlag, "status", "", "filter by classified job status, or \"all\" for all normalized jobs (required)")
	renormalizeCmd.MarkFlagRequired("status")
	rootCmd.AddCommand(renormalizeCmd)
}

func runRenormalize(cmd *cobra.Command, args []string) error {
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
	var ids []pgtype.UUID
	if renormalizeStatusFlag == "all" {
		ids, err = queries.ListNormalizedClassifiedJobIDs(ctx)
	} else {
		ids, err = queries.ListClassifiedJobIDsByStatus(ctx, renormalizeStatusFlag)
	}
	if err != nil {
		return fmt.Errorf("listing jobs: %w", err)
	}

	n, err := requeueJobs(ctx, pool, ids, constants.PipelineNormalize)
	if err != nil {
		return err
	}

	fmt.Printf("re-queued %d jobs for normalization\n", n)
	return nil
}
