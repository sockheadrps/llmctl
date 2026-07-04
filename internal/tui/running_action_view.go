package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) viewRunningActionModal() string {
	title := modalTitleStyle.Render(m.runningAction.label)

	viewOpt := "  View Output  "
	stopOpt := "  Stop  "
	if m.runningAction.selected == runningActionViewOutput {
		viewOpt = selectedProfileStyle.Render("[ View Output ]")
		stopOpt = profileStyle.Render(stopOpt)
	} else {
		viewOpt = profileStyle.Render(viewOpt)
		stopOpt = selectedProfileStyle.Render("[ Stop ]")
	}
	options := fmt.Sprintf("%s    %s", viewOpt, stopOpt)

	help := helpStyle.Render("←/→ choose  enter confirm  esc cancel")

	body := lipgloss.JoinVertical(lipgloss.Center, title, "", options, "", help)
	return modalStyle.Render(body)
}
