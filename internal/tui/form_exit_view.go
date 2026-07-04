package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) viewFormExitModal() string {
	title := modalTitleStyle.Render("Unsaved Changes")
	message := profileStyle.Render("Exit this profile editor?")

	discardOpt := "  Exit without saving  "
	saveOpt := "  Save and exit  "
	if m.formExit.selected == formExitDiscard {
		discardOpt = selectedProfileStyle.Render("[ Exit without saving ]")
		saveOpt = profileStyle.Render(saveOpt)
	} else {
		discardOpt = profileStyle.Render(discardOpt)
		saveOpt = selectedProfileStyle.Render("[ Save and exit ]")
	}
	options := fmt.Sprintf("%s    %s", discardOpt, saveOpt)

	help := helpStyle.Render("←/→ choose  enter confirm  esc cancel")

	body := lipgloss.JoinVertical(lipgloss.Center, title, "", message, "", options, "", help)
	return modalStyle.Render(body)
}
