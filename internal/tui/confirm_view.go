package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) viewConfirmModal() string {
	title := modalTitleStyle.Render(m.confirm.label)

	runOpt := "  Run  "
	editOpt := "  Edit  "
	exportOpt := "  Export Args  "
	if m.confirm.selected == confirmRun {
		runOpt = selectedProfileStyle.Render("[ Run ]")
		editOpt = profileStyle.Render(editOpt)
		exportOpt = profileStyle.Render(exportOpt)
	} else if m.confirm.selected == confirmEdit {
		runOpt = profileStyle.Render(runOpt)
		editOpt = selectedProfileStyle.Render("[ Edit ]")
		exportOpt = profileStyle.Render(exportOpt)
	} else {
		runOpt = profileStyle.Render(runOpt)
		editOpt = profileStyle.Render(editOpt)
		exportOpt = selectedProfileStyle.Render("[ Export Args ]")
	}
	options := fmt.Sprintf("%s    %s    %s", runOpt, editOpt, exportOpt)

	help := helpStyle.Render("←/→ choose  enter confirm  esc cancel")

	body := lipgloss.JoinVertical(lipgloss.Center, title, "", options, "", help)
	return modalStyle.Render(body)
}
