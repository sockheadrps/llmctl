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

	colorLines := strings.Split(fg, "\n")
	plainLines := strings.Split(stripANSI(fg), "\n")
	modalWidth := 0
	for _, line := range plainLines {
		if w := lipgloss.Width(line); w > modalWidth {
			modalWidth = w
		}
	}
	modalHeight := len(colorLines)

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

	dimStyle := lipgloss.NewStyle().Background(lipgloss.Color("236"))

	for i := range canvas {
		canvas[i] = dimStyle.Width(bgWidth).Render(canvas[i])
	}

	for rowIndex, colorLine := range colorLines {
		row := startRow + rowIndex
		if row < 0 || row >= bgHeight {
			continue
		}

		plainLine := ""
		if rowIndex < len(plainLines) {
			plainLine = plainLines[rowIndex]
		}
		plainW := lipgloss.Width(plainLine)
		padding := ""
		if plainW < maxModalWidth {
			padding = strings.Repeat(" ", maxModalWidth-plainW)
		}

		// Render the left and right dim strips independently so the modal's
		// ANSI reset sequences don't bleed into the surrounding background.
		rightCols := bgWidth - startCol - maxModalWidth
		if rightCols < 0 {
			rightCols = 0
		}
		leftStrip := dimStyle.Render(strings.Repeat(" ", startCol))
		rightStrip := dimStyle.Render(strings.Repeat(" ", rightCols))
		canvas[row] = leftStrip + colorLine + padding + rightStrip
	}

	return strings.Join(canvas, "\n")
}
