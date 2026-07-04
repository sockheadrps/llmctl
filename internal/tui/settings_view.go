package tui

import (
	"fmt"
	"strings"
)

// renderSettingsDetail shows the given settings category's content in the
// Details pane — no separate screen, just like a model's profiles preview
// in place when it's focused. The header names the category itself instead
// of a generic "Details" label, same as the model/profile headers do.
func (m Model) renderSettingsDetail(categoryID string) string {
	label := categoryID
	for _, c := range settingsCategories {
		if c.id == categoryID {
			label = c.label
			break
		}
	}

	var b strings.Builder
	b.WriteString(modelStyle.Render(label))
	b.WriteString("\n\n")

	switch categoryID {
	case "model_dirs":
		b.WriteString(m.renderDirsContent())
	}

	if m.settings.dirs.err != "" {
		b.WriteString("\n")
		b.WriteString(errorStyle.Render("error: " + m.settings.dirs.err))
	}

	return b.String()
}

func (m Model) renderDirsContent() string {
	var b strings.Builder

	focused := m.focus == focusSettingsContent

	addCursor, addRowStyle := "  ", addStyle
	switch {
	case m.settings.dirs.cursor == 0 && focused:
		addCursor = cursorStyle.Render("> ")
		addRowStyle = selectedAddStyle
	case m.settings.dirs.cursor == 0:
		addCursor = profileStyle.Render("> ")
	}
	fmt.Fprintf(&b, "%s%s\n", addCursor, addRowStyle.Render("+ Add Directory"))

	if len(m.settings.dirs.list) == 0 {
		b.WriteString(profileStyle.Render("(no directories configured)"))
		b.WriteString("\n")
	}
	for i, d := range m.settings.dirs.list {
		cursor := "  "
		style := profileStyle
		label := d
		switch {
		case d == m.settings.dirs.pendingDel:
			style = pendingDeleteStyle
			label += " (del again to confirm)"
		case m.settings.dirs.cursor == i+1 && focused:
			cursor = cursorStyle.Render("> ")
			style = selectedProfileStyle
		case m.settings.dirs.cursor == i+1:
			cursor = profileStyle.Render("> ")
		}
		fmt.Fprintf(&b, "%s%s\n", cursor, style.Render(label))
	}

	if m.settings.dirs.editing {
		b.WriteString("\n")
		label := "New Directory:"
		if m.settings.dirs.editingIdx >= 0 {
			label = "Edit Directory:"
		}
		b.WriteString(formLabelStyle.Render(label) + " " + m.settings.dirs.input.View())
		b.WriteString("\n")
	}

	return b.String()
}
