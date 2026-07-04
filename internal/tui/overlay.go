package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// overlayCenter composites fg over bg as a centered modal. Rather than
// splicing ANSI-styled runes column-by-column (fragile, easy to corrupt
// escape sequences), it replaces whole background lines in fg's row band
// with fg's lines, each re-centered horizontally — cheap and always
// ANSI-safe since it never slices inside a styled bg line.
func overlayCenter(bg, fg string) string {
	bgLines := strings.Split(bg, "\n")
	fgLines := strings.Split(fg, "\n")

	width := lipgloss.Width(bg)
	if width <= 0 {
		width = 80
	}

	startRow := (len(bgLines) - len(fgLines)) / 2
	if startRow < 0 {
		startRow = 0
	}

	for i, line := range fgLines {
		row := startRow + i
		if row < 0 || row >= len(bgLines) {
			continue
		}
		bgLines[row] = lipgloss.PlaceHorizontal(width, lipgloss.Center, line)
	}

	return strings.Join(bgLines, "\n")
}
