package tui

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/sockheadrps/llmctl/internal/models"
)

func (m Model) copyEndpoint(run models.Running) (tea.Model, tea.Cmd) {
	endpoint := fmt.Sprintf("http://localhost:%d/v1", run.Port)
	if err := writeClipboard(endpoint); err != nil {
		m.setError(fmt.Errorf("copy endpoint: %w", err), "")
		return m, nil
	}
	m.clearError()
	return m, nil
}
