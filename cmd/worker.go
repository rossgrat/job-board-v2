package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var workerCmd = &cobra.Command{
	Use:   "worker",
	Short: "Start the job pipeline worker",
	RunE:  RunWorker,
}

func init() {
	rootCmd.AddCommand(workerCmd)
}

func RunWorker(cmd *cobra.Command, args []string) error {
	fmt.Println("Hello world!")
	return nil
}
