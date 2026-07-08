package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"

	"github.com/sockheadrps/llmctl/internal/health"
)

func (m Model) viewRPCServerActionModal() string {
	title := modalTitleStyle.Render("RPC Server")
	status := m.rpcServerHealthStatus()

	var options string
	var help string

	switch status {
	case health.StatusNotStarted:
		startOpt := "  Start  "
		cancelOpt := "  Cancel  "
		switch m.rpcServerActionState.selected {
		case rpcServerActionPrimary:
			startOpt = selectedProfileStyle.Render("[ Start ]")
			cancelOpt = profileStyle.Render(cancelOpt)
		default:
			startOpt = profileStyle.Render(startOpt)
			cancelOpt = selectedProfileStyle.Render("[ Cancel ]")
		}
		options = fmt.Sprintf("%s    %s", startOpt, cancelOpt)
		help = helpStyle.Render("←/→ choose  enter confirm  esc cancel")

	case health.StatusUp:
		viewOpt := "  View Output  "
		stopOpt := "  Stop  "
		cancelOpt := "  Cancel  "
		switch m.rpcServerActionState.selected {
		case rpcServerActionPrimary:
			viewOpt = selectedProfileStyle.Render("[ View Output ]")
			stopOpt = profileStyle.Render(stopOpt)
			cancelOpt = profileStyle.Render(cancelOpt)
		case rpcServerActionSecondary:
			viewOpt = profileStyle.Render(viewOpt)
			stopOpt = selectedProfileStyle.Render("[ Stop ]")
			cancelOpt = profileStyle.Render(cancelOpt)
		default:
			viewOpt = profileStyle.Render(viewOpt)
			stopOpt = profileStyle.Render(stopOpt)
			cancelOpt = selectedProfileStyle.Render("[ Cancel ]")
		}
		options = fmt.Sprintf("%s    %s    %s", viewOpt, stopOpt, cancelOpt)
		help = helpStyle.Render("←/→ choose  enter confirm  esc cancel")

	default: // StatusDown — crashed/errored
		viewOpt := "  View Output  "
		clearOpt := "  Clear  "
		cancelOpt := "  Cancel  "
		switch m.rpcServerActionState.selected {
		case rpcServerActionPrimary:
			viewOpt = selectedProfileStyle.Render("[ View Output ]")
			clearOpt = profileStyle.Render(clearOpt)
			cancelOpt = profileStyle.Render(cancelOpt)
		case rpcServerActionSecondary:
			viewOpt = profileStyle.Render(viewOpt)
			clearOpt = selectedProfileStyle.Render("[ Clear ]")
			cancelOpt = profileStyle.Render(cancelOpt)
		default:
			viewOpt = profileStyle.Render(viewOpt)
			clearOpt = profileStyle.Render(clearOpt)
			cancelOpt = selectedProfileStyle.Render("[ Cancel ]")
		}
		options = fmt.Sprintf("%s    %s    %s", viewOpt, clearOpt, cancelOpt)
		help = helpStyle.Render("←/→ choose  enter confirm  esc cancel")
	}

	body := lipgloss.JoinVertical(lipgloss.Center, title, "", options, "", help)
	return modalStyle.Render(body)
}
