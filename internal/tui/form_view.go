package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
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
	sections := []struct {
		name   string
		fields []int
	}{
		{name: "Identity", fields: []int{fieldKey, fieldHost, fieldAlias, fieldPort, fieldCtxSize}},
		{name: "Sampling", fields: []int{fieldTemp, fieldTopP, fieldTopK, fieldMinP, fieldPresencePenalty, fieldRepetitionPenalty, fieldFrequencyPenalty, fieldSeed}},
		{name: "Cache & Compute", fields: []int{fieldBatchSize, fieldUBatchSize, fieldRepeatLastN, fieldGPULayers, fieldMMap, fieldKVOffload}},
		{name: "Server", fields: []int{fieldParallelSlots, fieldContBatching, fieldCachePrompt, fieldCacheRAM}},
		{name: "Reasoning", fields: []int{fieldReasoning, fieldReasoningBudget, fieldReasoningFormat}},
		{name: "Advanced", fields: []int{fieldCacheK, fieldCacheV, fieldExtraArgs, fieldNotes}},
	}

	visibleRows := m.formVisibleRows()
	selectedRows := make([]string, 0, len(m.form.fields)+3)
	focusedRow := 0
	rowIndex := 0
	for _, section := range sections {
		if section.name != "" {
			selectedRows = append(selectedRows, sectionTitleStyle.Render(section.name))
			rowIndex++
		}
		for _, idx := range section.fields {
			f := m.form.fields[idx]
			label := formLabelStyle
			if m.form.focus == idx {
				label = formFocusedLabelStyle
				focusedRow = rowIndex
			}
			selectedRows = append(selectedRows, fmt.Sprintf("%s %s", label.Render(f.label+":"), f.input.View()))
			rowIndex++
		}
	}
	flashLabel := formLabelStyle
	if m.form.focus == len(m.form.fields) {
		flashLabel = formFocusedLabelStyle
		focusedRow = rowIndex
	}
	flashValue := "false"
	if m.form.flash {
		flashValue = "true"
	}
	selectedRows = append(selectedRows, fmt.Sprintf("%s %s", flashLabel.Render("Flash Attention:"), flashValue))
	rowIndex++
	selectedRows = append(selectedRows, "")
	rowIndex++
	saveStyle := profileStyle
	if m.form.focus == len(m.form.fields)+1 {
		saveStyle = selectedProfileStyle
		focusedRow = rowIndex
	}
	selectedRows = append(selectedRows, saveStyle.Render("[ Save ]"))

	start := 0
	if len(selectedRows) > visibleRows {
		if focusedRow < start {
			start = focusedRow
		} else if focusedRow >= start+visibleRows {
			start = focusedRow - visibleRows + 1
		}
		maxStart := len(selectedRows) - visibleRows
		if start < 0 {
			start = 0
		} else if start > maxStart {
			start = maxStart
		}
	}
	for i := start; i < min(start+visibleRows, len(selectedRows)); i++ {
		body.WriteString(selectedRows[i])
		body.WriteString("\n")
	}
	if len(selectedRows) > visibleRows {
		body.WriteString(helpStyle.Render("↑/↓ scroll  tab/enter next"))
		body.WriteString("\n")
	}

	leftPane := paneStyle.Width(70).Render(body.String())
	rightPane := strings.Builder{}
	rightPane.WriteString(sectionTitleStyle.Render("Details"))
	rightPane.WriteString("\n\n")
	if m.form.focus < len(m.form.fields) {
		rightPane.WriteString(detailMutedStyle.Render(m.form.fields[m.form.focus].label))
		rightPane.WriteString("\n\n")
		rightPane.WriteString(strings.Join(strings.Split(formFieldDescription(m.form.focus), " "), " "))
	} else if m.form.focus == len(m.form.fields) {
		rightPane.WriteString(detailMutedStyle.Render("Flash Attention"))
		rightPane.WriteString("\n\n")
		rightPane.WriteString("Enable optimized attention kernels when your build and GPU support them.")
	} else {
		rightPane.WriteString(detailMutedStyle.Render("Save Profile"))
		rightPane.WriteString("\n\n")
		rightPane.WriteString("Persist the current profile settings to the model configuration and return to the main list.")
	}
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, leftPane, paneStyle.Width(36).Render(rightPane.String())))
	b.WriteString("\n")

	if m.form.err != "" {
		b.WriteString(errorStyle.Render("error: " + m.form.err))
		b.WriteString("\n")
	}
	b.WriteString(helpStyle.Render("tab/↓ next  shift+tab/↑ prev  space toggle  enter next/save  esc cancel"))
	return b.String()
}
