package tui

import tea "github.com/charmbracelet/bubbletea"

// stopResultMsg carries the outcome of an async profile stop. Manager.Stop
// now waits to confirm the process actually exited (escalating to SIGKILL
// if needed), so it must run as a tea.Cmd rather than inline in Update to
// avoid freezing the UI.
type stopResultMsg struct {
	label string
	err   error
}

func (m Model) stopProfileCmd(modelKey, profileKey, label string) tea.Cmd {
	ctrl := m.ctrl
	return func() tea.Msg {
		err := ctrl.StopModel(modelKey, profileKey)
		return stopResultMsg{label: label, err: err}
	}
}

// stopRunning kicks off an async stop for modelKey/profileKey and switches
// on the "stopping…" status; the result arrives later as a stopResultMsg.
func (m Model) stopRunning(modelKey, profileKey, label string) (tea.Model, tea.Cmd) {
	m.stopping = true
	m.stoppingLabel = label
	m.clearError()
	return m, m.stopProfileCmd(modelKey, profileKey, label)
}
