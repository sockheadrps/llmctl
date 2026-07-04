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
}

func (m Model) openStopConfirm(modelKey, profileKey, label string) (tea.Model, tea.Cmd) {
	m.stopConfirm = stopConfirmState{
		modelKey:   modelKey,
		profileKey: profileKey,
		label:      label,
	}
	m.screen = screenStopConfirm
	return m, nil
}

func (m Model) updateStopConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "n", "N":
		m.screen = screenMain
		return m, nil
	case "enter", "y", "Y":
		m.screen = screenMain
		return m.stopRunning(m.stopConfirm.modelKey, m.stopConfirm.profileKey, m.stopConfirm.label)
	}
	return m, nil
}

func (m Model) viewStopConfirmModal() string {
	title := modalTitleStyle.Render("Stop Server")
	msg := profileStyle.Render(fmt.Sprintf("Stop %s?", m.stopConfirm.label))
	help := helpStyle.Render("y / enter  confirm    n / esc  cancel")
	body := lipgloss.JoinVertical(lipgloss.Center, title, "", msg, "", help)
	return modalStyle.Render(body)
}
