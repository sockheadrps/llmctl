package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"
)

type stopConfirmState struct {
	modelKey   string
	profileKey string
	label      string
	cursor     int // 0 = Stop, 1 = Cancel
}

func (m Model) openStopConfirm(modelKey, profileKey, label string) (tea.Model, tea.Cmd) {
	m.stopConfirm = stopConfirmState{
		modelKey:   modelKey,
		profileKey: profileKey,
		label:      label,
		cursor:     0,
	}
	m.screen = screenStopConfirm
	return m, nil
}

func (m Model) updateStopConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "left", "h", "a":
		m.stopConfirm.cursor = 0
		return m, nil
	case "right", "l", "d":
		m.stopConfirm.cursor = 1
		return m, nil
	case "esc":
		m.screen = screenMain
		return m, nil
	case "enter", " ":
		m.screen = screenMain
		if m.stopConfirm.cursor == 0 {
			return m.stopRunning(m.stopConfirm.modelKey, m.stopConfirm.profileKey, m.stopConfirm.label)
		}
		return m, nil
	}
	return m, nil
}

func (m Model) viewStopConfirmModal() string {
	stopOpt := "  Stop  "
	cancelOpt := "  Cancel  "
	if m.stopConfirm.cursor == 0 {
		stopOpt = selectedProfileStyle.Render("[ Stop ]")
		cancelOpt = profileStyle.Render(cancelOpt)
	} else {
		stopOpt = profileStyle.Render(stopOpt)
		cancelOpt = selectedProfileStyle.Render("[ Cancel ]")
	}
	options := fmt.Sprintf("%s    %s", stopOpt, cancelOpt)

	title := modalTitleStyle.Render("Stop Server")
	label := profileStyle.Render(m.stopConfirm.label)
	help := helpStyle.Render("←/→ choose  enter confirm  esc cancel")
	body := lipgloss.JoinVertical(lipgloss.Center, title, "", label, "", options, "", help)
	return modalStyle.Render(body)
}
