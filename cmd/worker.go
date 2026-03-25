package cmd

import (
	"context"

	"github.com/rossgrat/job-board-v2/internal/application/worker"
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

	workerApp := worker.New()
	workerApp.Run(ctx)

	return nil
}
