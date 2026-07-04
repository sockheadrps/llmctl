package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export all model profiles to YAML (stdout)",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		out := struct {
			Models interface{} `yaml:"models"`
		}{Models: cfg.Models}

		data, err := yaml.Marshal(out)
		if err != nil {
			return fmt.Errorf("marshal: %w", err)
		}
		fmt.Print(string(data))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(exportCmd)
}
