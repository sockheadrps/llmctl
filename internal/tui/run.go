package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/sockheadrps/llmctl/internal/config"
	"github.com/sockheadrps/llmctl/internal/controller"
)

// Run starts the interactive TUI program and blocks until the user quits.
// cfgPath is where new models/profiles created in the TUI are persisted.
// netInternetConn, netRPCConn, and netIface configure the Network tab's
// nmcli profile names and the interface used for link-state polling.
func Run(cfg *config.Config, cfgPath string, ctrl *controller.Controller, netInternetConn, netRPCConn, netIface string) error {
	p := tea.NewProgram(New(cfg, cfgPath, ctrl, netInternetConn, netRPCConn, netIface), tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err := p.Run()
	return err
}
