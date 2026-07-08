package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) viewRPCLayerSplitModal() string {
	s := m.rpcLayerSplit
	serverLayers := s.totalLayers - s.clientLayers

	title := modalTitleStyle.Render("RPC Layer Distribution")
	sub := profileStyle.Render(s.label)

	const barWidth = 40
	var filled int
	if s.totalLayers > 0 {
		filled = barWidth * s.clientLayers / s.totalLayers
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)
	barStyled := lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Render(bar)

	localLabel := lipgloss.NewStyle().Foreground(lipgloss.Color("42")).Bold(true).
		Render(fmt.Sprintf("Local  %3d", s.clientLayers))
	serverLabel := lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true).
		Render(fmt.Sprintf("%3d  Remote", serverLayers))
	splitLine := fmt.Sprintf("%s  %s  %s", localLabel, barStyled, serverLabel)

	hint := helpStyle.Render("← → adjust · shift+←→ jump 5 · enter start · esc cancel")

	body := lipgloss.JoinVertical(lipgloss.Center,
		title,
		"",
		sub,
		"",
		splitLine,
		"",
		hint,
	)
	return modalStyle.Render(body)
}
