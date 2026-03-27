package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rossgrat/job-board-v2/internal/config"
	"github.com/rossgrat/job-board-v2/internal/worker"
	"github.com/spf13/cobra"
)

var workerCmd = &cobra.Command{
	Use:   "worker",
	Short: "Start the job pipeline worker",
	RunE:  StartWorker,
}

func init() {
	rootCmd.AddCommand(workerCmd)
}

func StartWorker(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", slog.String("err", err.Error()))
		os.Exit(1)
	}

	dbPool, err := pgxpool.New(ctx, cfg.Postgres.DSN())
	if err != nil {
		slog.Error("failed to connect to DB", slog.String("err", err.Error()))
		os.Exit(1)
	}
	defer dbPool.Close()

	slog.Info("connected to postgres", slog.String("host", cfg.Postgres.Host))

	worker, err := worker.New(ctx, dbPool)
	if err != nil {
		fmt.Println("Failed to init worker")
		os.Exit(1)
	}
	worker.Run(ctx)

	return nil
}
