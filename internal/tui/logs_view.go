package tui

import (
	"fmt"
	"strings"
)

func (m Model) viewLogs() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Logs — " + m.logs.label))
	b.WriteString("\n")
	b.WriteString(helpStyle.Render(m.logs.path))
	b.WriteString("\n\n")

	visible := logsVisibleHeight(m.height)

	var body string
	switch {
	case m.logs.err != nil:
		body = errorStyle.Render("could not read log: " + m.logs.err.Error())
	case len(m.logs.lines) == 0:
		body = profileStyle.Render("(empty log)")
	default:
		end := min(m.logs.offset+visible, len(m.logs.lines))
		body = strings.Join(m.logs.lines[m.logs.offset:end], "\n")
	}

	width := m.width - 4
	if width < 60 {
		width = 60
	}

	b.WriteString(paneStyle.Width(width).Height(visible).Render(body))
	b.WriteString("\n")

	if len(m.logs.lines) > 0 {
		shown := min(m.logs.offset+visible, len(m.logs.lines))
		b.WriteString(helpStyle.Render(fmt.Sprintf(
			"line %d-%d/%d  ↑/k ↓/j scroll  pgup/pgdn page  g/G top/bottom  esc/q/e back",
			m.logs.offset+1, shown, len(m.logs.lines))))
	} else {
		b.WriteString(helpStyle.Render("esc/q/e back"))
	}

	return b.String()
}
