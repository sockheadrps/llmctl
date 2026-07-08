package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/sockheadrps/llmctl/internal/runtime"
)

// runCmd starts a model+profile pair as a detached llama-server instance.
var runCmd = &cobra.Command{
	Use:   "run <model> <profile>",
	Short: "Start a model with the given profile",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		modelKey, profileKey := args[0], args[1]

		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		mgr, err := runtime.NewManager()
		if err != nil {
			return err
		}

		entry, err := mgr.Start(cfg, modelKey, profileKey, "")
		if err != nil {
			return err
		}

		fmt.Printf("started %s (pid %d) on :%d\n", entry.Label(), entry.PID, entry.Port)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
}
