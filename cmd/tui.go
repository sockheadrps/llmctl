package cmd

import (
	"github.com/spf13/cobra"

	"github.com/sockheadrps/llmctl/internal/runtime"
	"github.com/sockheadrps/llmctl/internal/tui"
)

// tuiCmd explicitly launches the interactive TUI (rootCmd does the same
// when invoked with no subcommand).
var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Launch the interactive terminal UI",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runTUI()
	},
}

func init() {
	rootCmd.AddCommand(tuiCmd)
}

func runTUI() error {
	cfgPath, err := resolveConfigPath()
	if err != nil {
		return err
	}

	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	mgr, err := runtime.NewManager()
	if err != nil {
		return err
	}

	return tui.Run(cfg, cfgPath, mgr)
}
