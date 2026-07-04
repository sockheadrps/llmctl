package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/sockheadrps/llmctl/internal/runtime"
)

// stopCmd stops a running model+profile instance.
var stopCmd = &cobra.Command{
	Use:   "stop <model> <profile>",
	Short: "Stop a running model instance",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		modelKey, profileKey := args[0], args[1]

		mgr, err := runtime.NewManager()
		if err != nil {
			return err
		}

		if err := mgr.Stop(modelKey, profileKey); err != nil {
			return err
		}

		fmt.Printf("stopped %s / %s\n", modelKey, profileKey)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(stopCmd)
}
