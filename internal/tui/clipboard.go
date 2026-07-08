package tui

import (
	"fmt"
	"net"
	"strconv"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/sockheadrps/llmctl/internal/models"
	"github.com/sockheadrps/llmctl/internal/util"
)

// copyStatusServerAddr copies the selected LAN status address and records it
// as the configured status server host/port shown in the RPC Server tab.
func (m Model) copyStatusServerAddr() (tea.Model, tea.Cmd) {
	addrs := m.statusServerAddrs()
	var toCopy string
	if len(addrs) > 0 {
		idx := m.rpcIPCursor - 1
		if idx < 0 || idx >= len(addrs) {
			idx = 0
		}
		toCopy = addrs[idx]
	} else {
		port := m.cfg.StatusServerPort
		if port == 0 {
			port = 11435
		}
		toCopy = fmt.Sprintf("0.0.0.0:%d", port)
	}
	if host, rawPort, err := net.SplitHostPort(toCopy); err == nil {
		m.cfg.StatusServerHost = host
		if p, err := strconv.Atoi(rawPort); err == nil {
			m.cfg.StatusServerPort = p
		}
		if err := m.saveConfig(); err != nil {
			m.setError(err, "")
			return m, nil
		}
		if err := m.reconcileStatusServer(); err != nil {
			m.setError(fmt.Errorf("status server: %w", err), "")
			return m, nil
		}
		m.pushStatusServer()
	}
	if err := writeClipboard(toCopy); err != nil {
		m.setError(fmt.Errorf("copy: %w", err), "")
		return m, nil
	}
	m.rpcAddrCopied = true
	m.clearError()
	return m, nil
}

func (m Model) statusServerAddrs() []string {
	port := m.cfg.StatusServerPort
	if port == 0 {
		port = 11435
	}
	addrs := util.StatusServerAddrs(port)
	selected := m.selectedStatusServerAddr()
	if selected == "" {
		return addrs
	}
	for _, addr := range addrs {
		if addr == selected {
			return addrs
		}
	}
	return append([]string{selected}, addrs...)
}

func (m Model) selectedStatusServerAddr() string {
	host := m.cfg.StatusServerHost
	if host == "" || host == "0.0.0.0" {
		return ""
	}
	port := m.cfg.StatusServerPort
	if port == 0 {
		port = 11435
	}
	return net.JoinHostPort(host, strconv.Itoa(port))
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
