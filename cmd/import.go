package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/sockheadrps/llmctl/internal/config"
	"github.com/sockheadrps/llmctl/internal/models"
)

var importMerge bool

var importCmd = &cobra.Command{
	Use:   "import <file>",
	Short: "Import model profiles from a YAML export",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		data, err := os.ReadFile(args[0])
		if err != nil {
			return fmt.Errorf("read %s: %w", args[0], err)
		}

		var src struct {
			Models map[string]models.Model `yaml:"models"`
		}
		if err := yaml.Unmarshal(data, &src); err != nil {
			return fmt.Errorf("parse %s: %w", args[0], err)
		}

		cfgPath, err := resolveConfigPath()
		if err != nil {
			return err
		}
		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		added, skipped := 0, 0
		for key, mdl := range src.Models {
			if _, exists := cfg.Models[key]; exists && !importMerge {
				skipped++
				continue
			}
			mdl.Key = key
			if mdl.Profiles == nil {
				mdl.Profiles = map[string]models.Profile{}
			}
			for pk, p := range mdl.Profiles {
				p.Name = pk
				mdl.Profiles[pk] = p
			}
			cfg.Models[key] = mdl
			added++
		}

		if err := config.Save(cfgPath, cfg); err != nil {
			return err
		}

		fmt.Printf("imported %d model(s)", added)
		if skipped > 0 {
			fmt.Printf(", skipped %d already present (use --merge to overwrite)", skipped)
		}
		fmt.Println()
		return nil
	},
}

func init() {
	importCmd.Flags().BoolVar(&importMerge, "merge", false, "overwrite existing models instead of skipping them")
	rootCmd.AddCommand(importCmd)
}
