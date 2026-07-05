package tui

import tea "github.com/charmbracelet/bubbletea"

// confirmAction is which option is highlighted on the Run/Edit screen.
type confirmAction int

const (
	confirmRun confirmAction = iota
	confirmEdit
)

// confirmState backs the screen shown after pressing Enter on a profile:
// a Run/Edit choice, defaulting to Run.
type confirmState struct {
	modelKey   string
	profileKey string
	label      string
	selected   confirmAction
}

// openConfirm switches to the Run/Edit screen for the profile under r.
func (m Model) openConfirm(r row) (tea.Model, tea.Cmd) {
	m.confirm = confirmState{
		modelKey:   r.modelKey,
		profileKey: r.profileKey,
		label:      r.label,
		selected:   confirmRun,
	}
	m.screen = screenConfirmProfile
	m.err = nil
	return m, nil
}

func (m Model) updateConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.screen = screenMain
		return m, nil

	case "left", "h", "a":
		m.confirm.selected = confirmRun
		return m, nil

	case "right", "l", "d":
		m.confirm.selected = confirmEdit
		return m, nil

	case "enter", " ":
		m.screen = screenMain
		switch m.confirm.selected {
		case confirmEdit:
			return m.openEditForm(m.confirm.modelKey, m.confirm.profileKey)
		default:
			return m.runProfile(row{kind: rowProfile, modelKey: m.confirm.modelKey, profileKey: m.confirm.profileKey, label: m.confirm.label})
		}
	}
	return m, nil
}
