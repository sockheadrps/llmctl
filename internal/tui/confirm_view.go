package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) viewConfirmModal() string {
	title := modalTitleStyle.Render(m.confirm.label)

	runOpt := "  Run  "
	editOpt := "  Edit  "
	if m.confirm.selected == confirmRun {
		runOpt = selectedProfileStyle.Render("[ Run ]")
		editOpt = profileStyle.Render(editOpt)
	} else {
		runOpt = profileStyle.Render(runOpt)
		editOpt = selectedProfileStyle.Render("[ Edit ]")
	}
	options := fmt.Sprintf("%s    %s", runOpt, editOpt)

	help := helpStyle.Render("←/→ choose  enter confirm  esc cancel")

	body := lipgloss.JoinVertical(lipgloss.Center, title, "", options, "", help)
	return modalStyle.Render(body)
}
