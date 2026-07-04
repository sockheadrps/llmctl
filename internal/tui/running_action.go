package tui

import tea "github.com/charmbracelet/bubbletea"

// runningActionChoice is which option is highlighted on the running-
// instance action modal (Enter on a row in the Running tab).
type runningActionChoice int

const (
	runningActionViewOutput runningActionChoice = iota
	runningActionStop
)

// runningActionState backs the modal shown after pressing Enter on a
// running instance in the Running tab: a View Output/Stop choice,
// defaulting to the non-destructive one.
type runningActionState struct {
	modelKey   string
	profileKey string
	label      string
	selected   runningActionChoice
}

// openRunningAction switches to the View Output/Stop modal for the running
// instance identified by modelKey/profileKey.
func (m Model) openRunningAction(modelKey, profileKey, label string) (tea.Model, tea.Cmd) {
	m.runningAction = runningActionState{
		modelKey:   modelKey,
		profileKey: profileKey,
		label:      label,
		selected:   runningActionViewOutput,
	}
	m.screen = screenRunningAction
	m.err = nil
	return m, nil
}

func (m Model) updateRunningAction(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.screen = screenMain
		return m, nil

	case "left", "h":
		m.runningAction.selected = runningActionViewOutput
		return m, nil

	case "right", "l":
		m.runningAction.selected = runningActionStop
		return m, nil

	case "enter":
		m.screen = screenMain
		switch m.runningAction.selected {
		case runningActionStop:
			return m.stopRunning(m.runningAction.modelKey, m.runningAction.profileKey, m.runningAction.label)
		default:
			run, ok := m.findRunning(m.runningAction.modelKey, m.runningAction.profileKey)
			if !ok {
				return m, nil
			}
			return m.openLogs(run.LogFile, m.runningAction.label)
		}
	}
	return m, nil
}
