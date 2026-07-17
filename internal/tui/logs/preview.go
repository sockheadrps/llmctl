package logs

import (
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"

	tui_form "github.com/sockheadrps/llmctl/internal/tui/form"
)

var ansiEscape = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// PreviewLines wraps raw log output into terminal rows for log previews.
func PreviewLines(raw string, boxWidth int) []string {
	innerWidth := tui_form.FormDescriptionTextWidth(boxWidth)
	if innerWidth <= 0 {
		innerWidth = tui_form.FormDescriptionTextWidth(34)
	}

	text := sanitizePreviewText(raw)
	rendered := lipgloss.NewStyle().Width(innerWidth).Render(strings.TrimRight(text, "\n"))
	rendered = strings.TrimRight(rendered, "\n")
	if rendered == "" {
		return nil
	}
	return strings.Split(rendered, "\n")
}

func sanitizePreviewText(s string) string {
	s = ansiEscape.ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return strings.Map(func(r rune) rune {
		switch r {
		case '\n', '\t':
			return r
		}
		if r < 32 || r == 127 {
			return -1
		}
		return r
	}, s)
}
