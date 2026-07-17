package tui

import (
	"fmt"
	"strings"

	tui_picker "github.com/sockheadrps/llmctl/internal/tui/picker"
)

func pickerSpinnerFrame(step int) string {
	return tui_picker.SpinnerFrame(step)
}

func (m Model) viewPicker() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Add Model"))
	b.WriteString("\n\n")

	dirs, _ := m.cfg.ResolvedModelsDirs()
	if len(dirs) > 0 {
		b.WriteString(infoStyle.Render("scanned " + strings.Join(dirs, ", ")))
		b.WriteString("\n")
	}
	if len(m.picker.Unreadable) > 0 {
		b.WriteString(errorStyle.Render("could not scan: " + strings.Join(m.picker.Unreadable, ", ")))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	body := strings.Builder{}
	switch {
	case m.picker.Err != nil:
		body.WriteString(errorStyle.Render(m.picker.Err.Error()))
	case len(m.picker.Files) == 0:
		body.WriteString(profileStyle.Render("no new .gguf files found"))
	default:
		for i, f := range m.picker.Files {
			cursor := "  "
			style := profileStyle
			if i == m.picker.Cursor {
				cursor = cursorStyle.Render("> ")
				style = selectedProfileStyle
			}
			body.WriteString(fmt.Sprintf("%s%s\n", cursor, style.Render(f)))
			if meta := tui_picker.FileMetadata(f); meta != "" {
				body.WriteString(fmt.Sprintf("  %s\n", detailMutedStyle.Render(meta)))
			}
		}
	}
	b.WriteString(paneStyle.Render(body.String()))
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("↑/k up  ↓/j down  enter import  esc cancel  (manage directories from Settings)"))
	return b.String()
}
