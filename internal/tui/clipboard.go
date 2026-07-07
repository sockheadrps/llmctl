package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/sockheadrps/llmctl/internal/models"
	"github.com/sockheadrps/llmctl/internal/util"
)

// copyStatusServerAddr copies the first LAN "ip:port" for the status server
// to the clipboard so the user can paste it into a remote llmctl's
// Remote Status Address field. Falls back to "0.0.0.0:port" if no LAN IP
// is found.
// copyStatusServerAddr copies the status server address(es) to clipboard.
// When multiple LAN IPs are found all are joined so the user can see them,
// but only the first is written to clipboard for easy pasting.
func (m Model) copyStatusServerAddr() (tea.Model, tea.Cmd) {
	port := m.cfg.StatusServerPort
	if port == 0 {
		port = 11435
	}
	addrs := util.StatusServerAddrs(port)
	var toCopy string
	if len(addrs) > 0 {
		toCopy = addrs[0]
	} else {
		toCopy = fmt.Sprintf("0.0.0.0:%d", port)
	}
	if err := writeClipboard(toCopy); err != nil {
		m.setError(fmt.Errorf("copy: %w", err), "")
		return m, nil
	}
	m.rpcAddrCopied = true
	m.clearError()
	return m, nil
}

func (m Model) copyEndpoint(run models.Running) (tea.Model, tea.Cmd) {
	endpoint := fmt.Sprintf("http://localhost:%d/v1", run.Port)
	if err := writeClipboard(endpoint); err != nil {
		m.setError(fmt.Errorf("copy endpoint: %w", err), "")
		return m, nil
	}
	m.clearError()
	return m, nil
}
