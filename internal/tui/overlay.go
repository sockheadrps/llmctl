package tui

import (
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var ansiEscape = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string {
	return ansiEscape.ReplaceAllString(s, "")
}

// overlayCenter renders fg as a centered modal over bg with a dimmed
// backdrop, keeping the modal aligned inside the viewport even when the
// terminal is narrow.
func overlayCenter(bg, fg string) string {
	bgWidth := lipgloss.Width(bg)
	if bgWidth <= 0 {
		bgWidth = 80
	}

	bgHeight := strings.Count(bg, "\n") + 1
	if bgHeight <= 0 {
		bgHeight = 24
	}

	plainFG := stripANSI(fg)
	modalLines := strings.Split(plainFG, "\n")
	modalWidth := 0
	for _, line := range modalLines {
		if w := lipgloss.Width(line); w > modalWidth {
			modalWidth = w
		}
	}
	modalHeight := len(modalLines)

	if modalWidth < 24 {
		modalWidth = 24
	}
	if modalHeight < 6 {
		modalHeight = 6
	}

	maxModalWidth := max(30, min(bgWidth-6, modalWidth+4))
	maxModalHeight := max(7, min(bgHeight-4, modalHeight+2))

	startRow := (bgHeight - maxModalHeight) / 2
	startCol := (bgWidth - maxModalWidth) / 2

	canvas := make([]string, bgHeight)
	for i := range canvas {
		canvas[i] = strings.Repeat(" ", bgWidth)
	}

	for rowIndex, line := range modalLines {
		row := startRow + rowIndex
		if row < 0 || row >= bgHeight {
			continue
		}

		trimmed := []rune(line)
		if len(trimmed) > maxModalWidth {
			trimmed = trimmed[:maxModalWidth]
		}
		lineText := string(trimmed)
		if len(trimmed) < maxModalWidth {
			lineText += strings.Repeat(" ", maxModalWidth-len(trimmed))
		}
		canvas[row] = canvas[row][:startCol] + lineText + canvas[row][startCol+maxModalWidth:]
	}

	for i := range canvas {
		canvas[i] = lipgloss.NewStyle().
			Width(bgWidth).
			Background(lipgloss.Color("236")).
			Render(canvas[i])
	}

	return strings.Join(canvas, "\n")
}
