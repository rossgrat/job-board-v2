package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rossgrat/job-board-v2/database/gen/db"
	"github.com/rossgrat/job-board-v2/internal/config"
	"github.com/rossgrat/job-board-v2/internal/model"
	"github.com/spf13/cobra"
)

var backfillCleanDataCmd = &cobra.Command{
	Use:   "backfill-clean-data",
	Short: "Backfill clean_data for raw jobs with empty clean_data",
	RunE:  runBackfillCleanData,
}

func init() {
	rootCmd.AddCommand(backfillCleanDataCmd)
}

func runBackfillCleanData(cmd *cobra.Command, args []string) error {
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
	jobs, err := queries.GetRawJobsWithEmptyCleanData(ctx)
	if err != nil {
		return fmt.Errorf("loading jobs: %w", err)
	}

	if len(jobs) == 0 {
		fmt.Println("no jobs with empty clean_data")
		return nil
	}

	fmt.Printf("backfilling %d jobs\n", len(jobs))

	for _, job := range jobs {
		cleanData := model.CleanContent(job.RawData)
		err := queries.UpdateRawJobCleanData(ctx, db.UpdateRawJobCleanDataParams{
			ID:        job.ID,
			CleanData: cleanData,
		})
		if err != nil {
			return fmt.Errorf("updating job %s: %w", job.SourceJobID, err)
		}
	}

	fmt.Printf("backfilled %d jobs\n", len(jobs))
	return nil
}
