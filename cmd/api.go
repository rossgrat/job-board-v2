package cmd

import (
	"context"
	"log/slog"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rossgrat/job-board-v2/internal/api"
	"github.com/rossgrat/job-board-v2/internal/config"
	"github.com/spf13/cobra"
)

var apiCmd = &cobra.Command{
	Use:   "api",
	Short: "Start the API server",
	RunE:  startAPI,
}

func init() {
	rootCmd.AddCommand(apiCmd)
}

func startAPI(cmd *cobra.Command, args []string) error {
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

	server := api.New(ctx, dbPool, cfg)
	server.Run()

	return nil
}
