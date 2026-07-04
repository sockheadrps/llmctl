package tui

import (
	"fmt"
	"strings"
)

func (m Model) viewPicker() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Add Model"))
	b.WriteString("\n\n")

	dirs, _ := m.cfg.ResolvedModelsDirs()
	b.WriteString(helpStyle.Render("scanning " + strings.Join(dirs, ", ")))
	b.WriteString("\n")
	if len(m.picker.unreadable) > 0 {
		b.WriteString(errorStyle.Render("could not scan: " + strings.Join(m.picker.unreadable, ", ")))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	body := strings.Builder{}
	switch {
	case m.picker.err != nil:
		body.WriteString(errorStyle.Render(m.picker.err.Error()))
	case len(m.picker.files) == 0:
		body.WriteString(profileStyle.Render("no new .gguf files found"))
	default:
		for i, f := range m.picker.files {
			cursor := "  "
			style := profileStyle
			if i == m.picker.cursor {
				cursor = cursorStyle.Render("> ")
				style = selectedProfileStyle
			}
			body.WriteString(fmt.Sprintf("%s%s\n", cursor, style.Render(f)))
		}
	}
	b.WriteString(paneStyle.Render(body.String()))
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("↑/k up  ↓/j down  enter import  esc cancel  (manage directories from Settings)"))
	return b.String()
}
