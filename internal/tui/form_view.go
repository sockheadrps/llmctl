package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const (
	formDefaultLeftWidth    = 70
	formDefaultDetailsWidth = 36
	formMinLeftWidth        = 36
	formMinDetailsWidth     = 24
	formMinInputWidth       = 8
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

	leftWidth, detailsWidth := m.formPaneWidths()
	rowWidth := formRowTextWidth(leftWidth)
	labelWidth := min(30, max(14, rowWidth/2-2))
	inputWidth := max(formMinInputWidth, rowWidth-labelWidth-1)
	if labelWidth+1+inputWidth > rowWidth {
		labelWidth = max(8, rowWidth-formMinInputWidth-1)
		inputWidth = max(formMinInputWidth, rowWidth-labelWidth-1)
	}
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
			f.input.Width = inputWidth
			labelText := truncateText(f.label+":", labelWidth)
			label := formLabelStyle.Width(labelWidth)
			if m.form.focus == idx {
				if !m.form.navigating {
					label = formEditingLabelStyle.Width(labelWidth)
				} else {
					label = formFocusedLabelStyle.Width(labelWidth)
				}
				focusedRow = rowIndex
			}
			rowStr := fmt.Sprintf("%s %s", label.Render(labelText), f.input.View())
			if idx == fieldGPULayers && m.form.cpuOnly {
				rowStr += " " + detailMutedStyle.Render("(overridden)")
			}
			selectedRows = append(selectedRows, fitStyledLine(rowStr, rowWidth))
			rowIndex++
		}
	}
	flashLabel := formLabelStyle
	if m.form.focus == len(m.form.fields) {
		flashLabel = formFocusedLabelStyle
		focusedRow = rowIndex
	}
	flashLabel = flashLabel.Width(labelWidth)
	flashValue := "false"
	if m.form.flash {
		flashValue = "true"
	}
	selectedRows = append(selectedRows, fitStyledLine(fmt.Sprintf("%s %s", flashLabel.Render(truncateText("Flash Attention:", labelWidth)), flashValue), rowWidth))
	rowIndex++

	cpuOnlyLabel := formLabelStyle
	if m.form.focus == len(m.form.fields)+1 {
		cpuOnlyLabel = formFocusedLabelStyle
		focusedRow = rowIndex
	}
	cpuOnlyLabel = cpuOnlyLabel.Width(labelWidth)
	cpuOnlyValue := "false"
	if m.form.cpuOnly {
		cpuOnlyValue = "true"
	}
	selectedRows = append(selectedRows, fitStyledLine(fmt.Sprintf("%s %s", cpuOnlyLabel.Render(truncateText("CPU Only:", labelWidth)), cpuOnlyValue), rowWidth))
	rowIndex++

	mlockLabel := formLabelStyle
	if m.form.focus == len(m.form.fields)+2 {
		mlockLabel = formFocusedLabelStyle
		focusedRow = rowIndex
	}
	mlockLabel = mlockLabel.Width(labelWidth)
	mlockValue := "false"
	if m.form.mlock {
		mlockValue = "true"
	}
	selectedRows = append(selectedRows, fitStyledLine(fmt.Sprintf("%s %s", mlockLabel.Render(truncateText("MLock:", labelWidth)), mlockValue), rowWidth))
	rowIndex++

	selectedRows = append(selectedRows, "")
	rowIndex++
	saveStyle := profileStyle
	if m.form.focus == len(m.form.fields)+3 {
		saveStyle = selectedProfileStyle
		focusedRow = rowIndex
	}
	selectedRows = append(selectedRows, fitStyledLine(saveStyle.Render("[ Save ]"), rowWidth))

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
		body.WriteString(helpStyle.Render("↑↓/wasd scroll  enter activate"))
		body.WriteString("\n")
	}

	paneHeight := m.formPaneHeight()
	leftPane := paneStyle.Width(leftWidth).Height(paneHeight).Render(body.String())
	rightPane := strings.Builder{}
	rightPane.WriteString(sectionTitleStyle.Render("Details"))
	rightPane.WriteString("\n\n")
	rightPane.WriteString(detailMutedStyle.Render(truncateText(m.formDescriptionTitle(), formDescriptionTextWidth(detailsWidth))))
	rightPane.WriteString("\n\n")
	descReserved := 4
	if currentFlag := m.form.focusedFlag(); currentFlag != "" {
		m.form.flagInput.Width = formDescriptionTextWidth(detailsWidth) - 7
		if m.form.flagFocus {
			rightPane.WriteString(formFocusedLabelStyle.Width(0).Render("Flag:"));rightPane.WriteString(" ");rightPane.WriteString(m.form.flagInput.View())
			rightPane.WriteString("\n")
			rightPane.WriteString(helpStyle.Render("← back  enter confirm"))
		} else {
			rightPane.WriteString(detailMutedStyle.Render("Flag:"));rightPane.WriteString(" ");rightPane.WriteString(m.form.flagInput.View())
			rightPane.WriteString("\n")
			rightPane.WriteString(helpStyle.Render("→/d to override"))
		}
		rightPane.WriteString("\n\n")
		descReserved = 7
	}
	if m.form.focus == fieldExtraArgs {
		if val := m.form.fields[fieldExtraArgs].input.Value(); val != "" {
			valLines := wrapWords(val, formDescriptionTextWidth(detailsWidth))
			rightPane.WriteString(detailMutedStyle.Render("Args:"))
			rightPane.WriteString("\n")
			for _, line := range valLines {
				rightPane.WriteString(line)
				rightPane.WriteString("\n")
			}
			rightPane.WriteString("\n")
			descReserved += len(valLines) + 2
		}
	}
	rightPane.WriteString(m.renderFormDescription(detailsWidth, paneHeight-descReserved))
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, leftPane, paneStyle.Width(detailsWidth).Height(paneHeight).Render(rightPane.String())))
	b.WriteString("\n")

	if m.form.err != "" {
		b.WriteString(errorStyle.Render("error: " + m.form.err))
		b.WriteString("\n")
	}
	b.WriteString(helpStyle.Render("↑↓/wasd navigate  enter edit/save  esc cancel"))
	b.WriteString("  ")
	b.WriteString(helpStyle.Render("x import args"))
	return b.String()
}

func (m Model) viewFormImportModal() string {
	_, detailsWidth := m.formPaneWidths()
	inputWidth := max(24, formDescriptionTextWidth(detailsWidth)-2)
	m.form.importInput.Width = inputWidth

	var body strings.Builder
	body.WriteString(modalTitleStyle.Render("Import Args"))
	body.WriteString("\n\n")
	body.WriteString(detailMutedStyle.Render("Paste llama-server args copied from Export Args."))
	body.WriteString("\n\n")
	body.WriteString(m.form.importInput.View())
	body.WriteString("\n\n")
	if m.form.importErr != "" {
		body.WriteString(errorStyle.Render("error: " + m.form.importErr))
		body.WriteString("\n\n")
	}
	body.WriteString(helpStyle.Render("enter apply  esc cancel"))
	return modalStyle.Render(body.String())
}

func (m Model) formDescriptionTitle() string {
	if m.form.focus < len(m.form.fields) {
		return m.form.fields[m.form.focus].label
	}
	switch m.form.focus - len(m.form.fields) {
	case 0:
		return "Flash Attention"
	case 1:
		return "CPU Only"
	case 2:
		return "MLock"
	}
	return "Save Profile"
}

func (m Model) formDescriptionText() string {
	if m.form.focus < len(m.form.fields) {
		return formFieldDescription(m.form.focus)
	}
	switch m.form.focus - len(m.form.fields) {
	case 0:
		return formFieldDescription(len(formLabels))
	case 1:
		return formFieldDescription(len(formLabels) + 1)
	case 2:
		return formFieldDescription(len(formLabels) + 2)
	}
	return formFieldDescription(len(formLabels) + 3)
}

func (m Model) formDescriptionLines(width int) []string {
	return wrapWords(m.formDescriptionText(), formDescriptionTextWidth(width))
}

func (m Model) formDescriptionLineCount() int {
	_, detailsWidth := m.formPaneWidths()
	return len(m.formDescriptionLines(detailsWidth))
}

func (m Model) formDescriptionVisibleLines() int {
	return max(2, m.formVisibleRows()-3)
}

func (m Model) renderFormDescription(width, visible int) string {
	return strings.Join(descriptionWindow(m.formDescriptionLines(width), visible, m.form.descScroll), "\n")
}

func (m Model) formPaneWidths() (leftWidth, detailsWidth int) {
	termWidth := m.width
	if termWidth <= 0 {
		termWidth = fallbackWidth
	}

	available := termWidth - 4 // two bordered panes side-by-side
	if available < formMinLeftWidth+formMinDetailsWidth {
		available = formMinLeftWidth + formMinDetailsWidth
	}

	detailsWidth = formDefaultDetailsWidth
	leftWidth = formDefaultLeftWidth
	if leftWidth+detailsWidth > available {
		detailsWidth = max(formMinDetailsWidth, min(formDefaultDetailsWidth, available/3))
		leftWidth = available - detailsWidth
		if leftWidth < formMinLeftWidth {
			leftWidth = formMinLeftWidth
			detailsWidth = max(formMinDetailsWidth, available-leftWidth)
		}
	}
	return leftWidth, detailsWidth
}

func formDescriptionTextWidth(paneWidth int) int {
	return max(8, paneWidth-2)
}

func formRowTextWidth(paneWidth int) int {
	return max(8, paneWidth-2)
}

func truncateText(s string, width int) string {
	if width <= 0 || len(s) <= width {
		return s
	}
	if width <= 1 {
		return s[:width]
	}
	return s[:width-1] + "."
}

func fitStyledLine(s string, width int) string {
	if width <= 0 || lipgloss.Width(s) <= width {
		return s
	}
	return truncateText(s, width)
}

func descriptionWindow(lines []string, visible, offset int) []string {
	if visible <= 0 {
		return nil
	}
	if len(lines) == 0 {
		lines = []string{""}
	}

	maxOffset := max(0, len(lines)-visible)
	offset = max(0, min(offset, maxOffset))

	window := make([]string, 0, visible)
	for i := offset; i < min(offset+visible, len(lines)); i++ {
		window = append(window, lines[i])
	}
	for len(window) < visible {
		window = append(window, "")
	}
	return window
}

func wrapWords(text string, width int) []string {
	if width <= 0 {
		return []string{text}
	}

	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{""}
	}

	lines := []string{words[0]}
	for _, word := range words[1:] {
		last := len(lines) - 1
		if len(lines[last])+1+len(word) <= width {
			lines[last] += " " + word
			continue
		}
		lines = append(lines, word)
	}
	return lines
}
