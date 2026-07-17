package tui

import (
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

// startResultMsg carries the outcome of an async profile start. Manager.Start
// blocks briefly to confirm the process didn't die immediately, so it must
// run as a tea.Cmd rather than inline in Update to avoid freezing the UI.
// logPath is set even on success, so the log viewer has somewhere to look.
type startResultMsg struct {
	modelKey   string
	profileKey string
	label      string
	logPath    string
	err        error
}

func (m Model) startProfileCmd(r row) tea.Cmd {
	ctrl, cfg := m.ctrl, m.cfg
	modelKey, profileKey, label := r.modelKey, r.profileKey, r.label
	rpcOverride := m.discoveredRPCEndpoint
	return func() tea.Msg {
		logPath, _ := ctrl.LogPath(modelKey, profileKey)
		_, err := ctrl.StartModel(cfg, modelKey, profileKey, rpcOverride)
		if err != nil && !logFileHasContent(logPath) {
			logPath = ""
		}
		return startResultMsg{modelKey: modelKey, profileKey: profileKey, label: label, logPath: logPath, err: err}
	}
}

func logFileHasContent(path string) bool {
	if path == "" {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && info.Size() > 0
}

// runProfile kicks off an async start for r and switches on the "starting…"
// status; the result arrives later as a startResultMsg.
func (m Model) runProfile(r row) (tea.Model, tea.Cmd) {
	m.starting = true
	m.startingLabel = r.label
	m.clearError()
	return m, m.startProfileCmd(r)
}
