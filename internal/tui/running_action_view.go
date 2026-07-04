package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) viewRunningActionModal() string {
	title := modalTitleStyle.Render(m.runningAction.label)

	viewOpt := "  View Output  "
	copyOpt := "  Copy Endpoint  "
	stopOpt := "  Stop  "
	if m.runningAction.selected == runningActionViewOutput {
		viewOpt = selectedProfileStyle.Render("[ View Output ]")
		stopOpt = profileStyle.Render(stopOpt)
		copyOpt = profileStyle.Render(copyOpt)
	} else if m.runningAction.selected == runningActionCopyEndpoint {
		viewOpt = profileStyle.Render(viewOpt)
		copyOpt = selectedProfileStyle.Render("[ Copy Endpoint ]")
		stopOpt = profileStyle.Render(stopOpt)
	} else {
		viewOpt = profileStyle.Render(viewOpt)
		copyOpt = profileStyle.Render(copyOpt)
		stopOpt = selectedProfileStyle.Render("[ Stop ]")
	}
	options := fmt.Sprintf("%s    %s    %s", viewOpt, copyOpt, stopOpt)

	help := helpStyle.Render("←/→ choose  enter confirm  esc cancel")

	body := lipgloss.JoinVertical(lipgloss.Center, title, "", options, "", help)
	return modalStyle.Render(body)
}
