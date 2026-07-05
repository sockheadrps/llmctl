package cmd

import "github.com/sockheadrps/llmctl/internal/build"

func init() {
	rootCmd.Version = build.Version
}
