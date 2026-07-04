/*
Copyright © 2026 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/sockheadrps/llmctl/internal/config"
	"github.com/sockheadrps/llmctl/internal/util"
)

// cfgFile holds the --config flag value, shared by all subcommands.
var cfgFile string

// rootCmd represents the base command when called without any subcommands.
// With no subcommand it launches the interactive TUI.
var rootCmd = &cobra.Command{
	Use:   "llmctl",
	Short: "A terminal UI for managing local llama.cpp models",
	Long: `llmctl manages local llama-server instances built from a config of
Models and reusable launch Profiles.

Run with no arguments to open the interactive TUI, or use one of the
subcommands (run, stop, ps, logs) to manage instances from scripts.`,
	// Once args parse successfully, a RunE failure is almost never a CLI
	// usage mistake (e.g. it's llama-server rejecting a flag) — don't dump
	// the full usage block on top of that error.
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runTUI()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default: ./config/config.yaml, then ~/.llmctl/config.yaml)")
}

// resolveConfigPath applies the --config flag, falling back to the default
// search path when unset.
func resolveConfigPath() (string, error) {
	if cfgFile != "" {
		return cfgFile, nil
	}
	return util.DefaultConfigPath()
}

// loadConfig resolves the --config flag (or the default search path) and
// loads it, shared by every subcommand.
func loadConfig() (*config.Config, error) {
	path, err := resolveConfigPath()
	if err != nil {
		return nil, err
	}

	cfg, err := config.Load(path)
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}
	return cfg, nil
}
