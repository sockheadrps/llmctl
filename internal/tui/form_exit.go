package tui

import tea "github.com/charmbracelet/bubbletea"

type formExitChoice int

const (
	formExitDiscard formExitChoice = iota
	formExitSave
)

type formExitState struct {
	selected formExitChoice
}

func (m Model) updateFormExit(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.screen = screenNewProfile
		return m, nil

	case "left", "h":
		m.formExit.selected = formExitDiscard
		return m, nil

	case "right", "l":
		m.formExit.selected = formExitSave
		return m, nil

	case "enter":
		switch m.formExit.selected {
		case formExitSave:
			m.screen = screenNewProfile
			return m.submitForm()
		default:
			m.screen = screenMain
			m.clearError()
			return m, nil
		}
	}
	return m, nil
}
