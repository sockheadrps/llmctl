package tui

import (
	"fmt"
	"strings"
)

func (m Model) viewForm() string {
	mdl := m.cfg.Models[m.form.modelKey]

	title := "New Profile"
	if m.form.editing {
		title = "Edit Profile"
	}

	var b strings.Builder
	b.WriteString(titleStyle.Render(fmt.Sprintf("%s — %s", title, mdl.Name)))
	b.WriteString("\n\n")

	body := strings.Builder{}
	for i, f := range m.form.fields {
		label := formLabelStyle
		if m.form.focus == i {
			label = formFocusedLabelStyle
		}
		body.WriteString(fmt.Sprintf("%s %s\n", label.Render(f.label+":"), f.input.View()))
	}

	flashLabel := formLabelStyle
	if m.form.focus == len(m.form.fields) {
		flashLabel = formFocusedLabelStyle
	}
	flashValue := "false"
	if m.form.flash {
		flashValue = "true"
	}
	body.WriteString(fmt.Sprintf("%s %s\n", flashLabel.Render("Flash Attention:"), flashValue))

	saveStyle := profileStyle
	if m.form.focus == len(m.form.fields)+1 {
		saveStyle = selectedProfileStyle
	}
	body.WriteString("\n")
	body.WriteString(saveStyle.Render("[ Save ]"))
	body.WriteString("\n")

	b.WriteString(paneStyle.Render(body.String()))
	b.WriteString("\n")

	if m.form.err != "" {
		b.WriteString(errorStyle.Render("error: " + m.form.err))
		b.WriteString("\n")
	}
	b.WriteString(helpStyle.Render("tab/↓ next  shift+tab/↑ prev  space toggle  enter next/save  esc cancel"))
	return b.String()
}
