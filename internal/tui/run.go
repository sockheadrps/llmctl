package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/sockheadrps/llmctl/internal/config"
	"github.com/sockheadrps/llmctl/internal/runtime"
)

// Run starts the interactive TUI program and blocks until the user quits.
// cfgPath is where new models/profiles created in the TUI are persisted.
func Run(cfg *config.Config, cfgPath string, mgr *runtime.Manager) error {
	p := tea.NewProgram(New(cfg, cfgPath, mgr), tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err := p.Run()
	return err
}
