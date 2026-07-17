package tui

import (
	"fmt"
	"strings"

	tui_logs "github.com/sockheadrps/llmctl/internal/tui/logs"
)

func (m Model) viewLogs() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Logs — " + m.logs.Label))
	b.WriteString("\n")
	b.WriteString(helpStyle.Render(m.logs.Path))
	b.WriteString("\n\n")

	visible := tui_logs.VisibleHeight(m.height)

	var body string
	switch {
	case m.logs.Err != nil:
		body = errorStyle.Render("could not read log: " + m.logs.Err.Error())
	case len(m.logs.Lines) == 0:
		body = profileStyle.Render("(empty log)")
	default:
		end := min(m.logs.Offset+visible, len(m.logs.Lines))
		body = strings.Join(m.logs.Lines[m.logs.Offset:end], "\n")
	}

	width := m.width - 4
	if width < 60 {
		width = 60
	}

	b.WriteString(paneStyle.Width(width).Height(visible).Render(body))
	b.WriteString("\n")

	if len(m.logs.Lines) > 0 {
		shown := min(m.logs.Offset+visible, len(m.logs.Lines))
		b.WriteString(helpStyle.Render(fmt.Sprintf(
			"line %d-%d/%d  ↑/k ↓/j scroll  pgup/pgdn page  g/G top/bottom  esc/q/e back",
			m.logs.Offset+1, shown, len(m.logs.Lines))))
	} else {
		b.WriteString(helpStyle.Render("esc/q/e back"))
	}

	return b.String()
}
