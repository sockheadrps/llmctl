package cmd

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/sockheadrps/llmctl/internal/runtime"
)

var follow bool

// logsCmd prints (and optionally follows) the log file for a running instance.
var logsCmd = &cobra.Command{
	Use:   "logs <model> <profile>",
	Short: "Show logs for a model instance",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		modelKey, profileKey := args[0], args[1]

		mgr, err := runtime.NewManager()
		if err != nil {
			return err
		}

		entry, ok, err := mgr.Find(modelKey, profileKey)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("%s / %s is not running", modelKey, profileKey)
		}

		f, err := os.Open(entry.LogFile)
		if err != nil {
			return fmt.Errorf("open log file: %w", err)
		}
		defer f.Close()

		if _, err := io.Copy(os.Stdout, f); err != nil {
			return err
		}

		if !follow {
			return nil
		}

		for {
			n, err := io.Copy(os.Stdout, f)
			if err != nil {
				return err
			}
			if n == 0 {
				time.Sleep(500 * time.Millisecond)
			}
		}
	},
}

func init() {
	logsCmd.Flags().BoolVarP(&follow, "follow", "f", false, "follow log output")
	rootCmd.AddCommand(logsCmd)
}
