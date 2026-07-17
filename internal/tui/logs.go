package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	tui_logs "github.com/sockheadrps/llmctl/internal/tui/logs"
)

// openLogsForCurrent opens the log viewer for whatever's most relevant
// right now: the log behind the current error if there is one, otherwise
// whichever profile/instance is currently selected.
func (m Model) openLogsForCurrent() (tea.Model, tea.Cmd) {
	if m.errLogPath != "" {
		return m.openLogs(m.errLogPath, "error")
	}

	if m.leftMode == modeRPCServer {
		logPath := m.rpcServerState.LogFile
		if logPath == "" {
			var err error
			logPath, err = m.ctrl.RPCServerLogPath()
			if err != nil {
				m.setError(err, "")
				return m, nil
			}
		}
		return m.openLogs(logPath, "RPC Server")
	}

	if m.focus == focusRunning {
		if m.runningCursor < 0 || m.runningCursor >= len(m.running) {
			return m, nil
		}
		r := m.running[m.runningCursor]
		return m.openLogs(r.LogFile, r.Label())
	}

	r, ok := m.currentRow()
	if !ok || (r.kind != rowProfile && r.kind != rowRunning) {
		return m, nil
	}
	path, err := m.ctrl.LogPath(r.modelKey, r.profileKey)
	if err != nil {
		m.setError(err, "")
		return m, nil
	}
	return m.openLogs(path, r.label)
}

// openLogs reads path and switches to the log-viewer screen, scrolled to the
// end - the most recent output, and where a crash's error would be.
func (m Model) openLogs(path, label string) (tea.Model, tea.Cmd) {
	m.logs = tui_logs.Open(path, label, m.width, m.height)
	m.screen = screenLogs
	return m, nil
}

// refreshLogs re-reads the log file in place. If the viewer was scrolled to
// the bottom it stays there; otherwise the offset is preserved.
func (m *Model) refreshLogs() {
	m.logs.Refresh(m.width, m.height)
}

func (m Model) updateLogs(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.logs.Update(msg, m.height) {
		m.screen = screenMain
	}
	return m, nil
}
