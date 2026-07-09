package tui

import (
	"fmt"
	"net"
	"strconv"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/sockheadrps/llmctl/internal/models"
	"github.com/sockheadrps/llmctl/internal/util"
)

type clearOverviewCopiedMsg struct{}

func clearOverviewCopiedCmd() tea.Cmd {
	return tea.Tick(2*time.Second, func(_ time.Time) tea.Msg {
		return clearOverviewCopiedMsg{}
	})
}

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

// overviewClickedEntry maps a mouse click (x, y) to the running entry it falls
// on in the Overview ACTIVE SERVICES column, using the same layout math as
// viewOverviewPage / renderActiveServices.
func (m Model) overviewClickedEntry(x, y int) (models.Running, bool) {
	totalW := m.width
	if totalW <= 0 {
		totalW = fallbackWidth
	}
	avail := totalW - 6
	if avail < 24 {
		avail = 24
	}
	leftCW := avail * 3 / 5
	if leftCW < 18 {
		leftCW = 18
	}
	rightCW := avail - leftCW
	if rightCW < 14 {
		rightCW = 14
		leftCW = avail - rightCW
	}
	_ = rightCW

	// Left column X range: cols 1..leftCW+2 (outer space + content + trailing space)
	// Y layout: 0=top border 1=blank 2=ACTIVE SERVICES header 3=ALIAS header
	//           4=Local label 5+=entries
	// Entry stride: 3 lines (wide) or 5 lines (narrow, leftCW < 50).
	entryStride := 3
	if leftCW < 50 {
		entryStride = 5
	}
	const entryStartY = 5
	if x < 1 || x > leftCW+2 || y < entryStartY {
		return models.Running{}, false
	}
	idx := (y - entryStartY) / entryStride
	if idx < 0 || idx >= len(m.running) {
		return models.Running{}, false
	}
	return m.running[idx], true
}

// copyOverviewEntry copies "host:port" for the given running instance and sets
// overviewCopied for a brief visual acknowledgement.
func (m Model) copyOverviewEntry(run models.Running) (tea.Model, tea.Cmd) {
	host := run.Host
	if host == "" {
		host = "localhost"
	}
	addr := fmt.Sprintf("%s:%d", host, run.Port)
	if err := writeClipboard(addr); err != nil {
		m.setError(fmt.Errorf("copy: %w", err), "")
		return m, nil
	}
	m.clearError()
	m.overviewCopied = run.ModelKey + "/" + run.ProfileKey
	return m, clearOverviewCopiedCmd()
}
