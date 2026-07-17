package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	tui_form "github.com/sockheadrps/llmctl/internal/tui/form"
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

	leftWidth, detailsWidth := tui_form.FormPaneWidths(m.width)
	rowWidth := tui_form.FormRowTextWidth(leftWidth)
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
		{name: "RPC", fields: []int{fieldRPCEnabled}},
	}

	visibleRows := tui_form.FormVisibleRows(tui_form.FormPaneHeight(m.height))
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
			labelText := tui_form.TruncateText(f.label+":", labelWidth)
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
			selectedRows = append(selectedRows, tui_form.FitStyledLine(rowStr, rowWidth))
			rowIndex++

			if idx == fieldGPULayers {
				selectedRows = append(selectedRows, tui_form.FitStyledLine(m.renderLayerDistRow(labelWidth, rowWidth), rowWidth))
				if m.form.focus == len(m.form.fields)+3 {
					focusedRow = rowIndex
				}
				rowIndex++
			}
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
	selectedRows = append(selectedRows, tui_form.FitStyledLine(fmt.Sprintf("%s %s", flashLabel.Render(tui_form.TruncateText("Flash Attention:", labelWidth)), flashValue), rowWidth))
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
	selectedRows = append(selectedRows, tui_form.FitStyledLine(fmt.Sprintf("%s %s", cpuOnlyLabel.Render(tui_form.TruncateText("CPU Only:", labelWidth)), cpuOnlyValue), rowWidth))
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
	selectedRows = append(selectedRows, tui_form.FitStyledLine(fmt.Sprintf("%s %s", mlockLabel.Render(tui_form.TruncateText("MLock:", labelWidth)), mlockValue), rowWidth))
	rowIndex++

	selectedRows = append(selectedRows, "")
	rowIndex++
	saveStyle := profileStyle
	if m.form.focus == len(m.form.fields)+4 {
		saveStyle = selectedProfileStyle
		focusedRow = rowIndex
	}
	selectedRows = append(selectedRows, tui_form.FitStyledLine(saveStyle.Render("[ Save ]"), rowWidth))

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

	paneHeight := tui_form.FormPaneHeight(m.height)
	leftPane := paneStyle.Width(leftWidth).Height(paneHeight).Render(body.String())
	// lipgloss.Height includes the 2 border rows; subtract them so that the
	// right pane's .Height() sets content height, making total heights match.
	actualHeight := lipgloss.Height(leftPane) - 2
	rightPane := strings.Builder{}
	rightPane.WriteString(sectionTitleStyle.Render("Details"))
	rightPane.WriteString("\n\n")
	rightPane.WriteString(detailMutedStyle.Render(tui_form.TruncateText(tui_form.DescriptionTitle(m.form.focus, len(m.form.fields)), tui_form.FormDescriptionTextWidth(detailsWidth))))
	rightPane.WriteString("\n\n")
	descReserved := 4
	if currentFlag := tui_form.FocusedFlag(m.form.focus, len(m.form.fields)); currentFlag != "" {
		m.form.flagInput.Width = tui_form.FormDescriptionTextWidth(detailsWidth) - 7
		if m.form.flagFocus {
			rightPane.WriteString(formFocusedLabelStyle.Width(0).Render("Flag:"))
			rightPane.WriteString(" ")
			rightPane.WriteString(m.form.flagInput.View())
			rightPane.WriteString("\n")
			rightPane.WriteString(helpStyle.Render("← back  enter confirm"))
		} else {
			rightPane.WriteString(detailMutedStyle.Render("Flag:"))
			rightPane.WriteString(" ")
			rightPane.WriteString(m.form.flagInput.View())
			rightPane.WriteString("\n")
			rightPane.WriteString(helpStyle.Render("→/d to override"))
		}
		rightPane.WriteString("\n\n")
		descReserved = 7
	}
	if m.form.focus == fieldExtraArgs {
		if val := m.form.fields[fieldExtraArgs].input.Value(); val != "" {
			valLines := tui_form.WrapWords(val, tui_form.FormDescriptionTextWidth(detailsWidth))
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
	rightPane.WriteString(m.renderFormDescription(detailsWidth, actualHeight-descReserved))
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, leftPane, paneStyle.Width(detailsWidth).Height(actualHeight).Render(rightPane.String())))
	b.WriteString("\n")

	if m.form.err != "" {
		b.WriteString(errorStyle.Render("error: " + m.form.err))
		b.WriteString("\n")
	}
	b.WriteString(helpStyle.Render("↑↓/wasd navigate  enter edit/save  esc cancel"))
	b.WriteString("  ")
	b.WriteString(helpStyle.Render("x import args"))
	if tui_form.FocusedFlag(m.form.focus, len(m.form.fields)) != "" && !m.form.flagFocus {
		b.WriteString("  ")
		b.WriteString(helpStyle.Render("→/d override flag"))
	}
	return b.String()
}

func (m Model) viewFormImportModal() string {
	_, detailsWidth := tui_form.FormPaneWidths(m.width)
	inputWidth := max(24, tui_form.FormDescriptionTextWidth(detailsWidth)-2)
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

func (m Model) formDescriptionLines(width int) []string {
	return tui_form.WrapWords(tui_form.DescriptionText(m.form.focus, len(m.form.fields)), tui_form.FormDescriptionTextWidth(width))
}

func (m Model) formDescriptionLineCount() int {
	_, detailsWidth := tui_form.FormPaneWidths(m.width)
	return len(m.formDescriptionLines(detailsWidth))
}

func (m Model) formDescriptionVisibleLines() int {
	return max(2, tui_form.FormVisibleRows(tui_form.FormPaneHeight(m.height))-3)
}

func (m Model) renderFormDescription(width, visible int) string {
	return strings.Join(tui_form.DescriptionWindow(m.formDescriptionLines(width), visible, m.form.descScroll), "\n")
}

func (m Model) renderLayerDistRow(labelWidth, rowWidth int) string {
	focused := m.form.focus == len(m.form.fields)+3
	lbl := formLabelStyle
	if focused {
		lbl = formFocusedLabelStyle
	}
	lbl = lbl.Width(labelWidth)

	total := tui_form.FormSliderTotal(m.form.fields[fieldGPULayers].input.Value())
	client := m.form.rpcClientLayers
	if total > 0 && client > total {
		client = total
	}
	server := 0
	if total > 0 {
		server = total - client
	}
	active := tui_form.FormRPCActive(m.form.fields[fieldRPCEnabled].input.Value(), m.cfg.RPCEnabled) && total > 0 && !m.form.cpuOnly

	if !active {
		hint := "enable RPC + GPU Layers to adjust"
		return fmt.Sprintf("%s %s", lbl.Render(tui_form.TruncateText("Layer Split:", labelWidth)), detailMutedStyle.Render(hint))
	}

	const barWidth = 24
	filled := 0
	if total > 0 {
		filled = barWidth * client / total
	}

	localStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
	remoteStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	arrowStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	if focused {
		arrowStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true)
	}

	bar := localStyle.Render(strings.Repeat("█", filled)) +
		detailMutedStyle.Render(strings.Repeat("░", barWidth-filled))

	left := arrowStyle.Render("◄")
	right := arrowStyle.Render("►")

	counts := localStyle.Render(fmt.Sprintf("%d", client)) +
		detailMutedStyle.Render("↔") +
		remoteStyle.Render(fmt.Sprintf("%d", server)) +
		detailMutedStyle.Render(fmt.Sprintf("/%d", total))

	return fmt.Sprintf("%s %s%s%s %s", lbl.Render(tui_form.TruncateText("Layer Split:", labelWidth)), left, bar, right, counts)
}
