package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/sockheadrps/llmctl/internal/process"
)

// Fallback dimensions used before the first WindowSizeMsg arrives (or if a
// terminal never reports one), and floors below which panes stop shrinking
// and start clipping/wrapping instead.
const (
	fallbackWidth  = 100
	fallbackHeight = 40
	minLeftWidth   = 32
	minRightWidth  = 34
	minBodyHeight  = 12
	chromeHeight   = 3 // header line + optional status/error line + help
)

func (m Model) View() string {
	switch m.screen {
	case screenPickModel:
		return m.viewPicker()
	case screenNewProfile:
		if m.form.importEditing {
			return overlayCenter(m.viewForm(), m.viewFormImportModal())
		}
		return m.viewForm()
	case screenFormExitConfirm:
		return overlayCenter(m.viewForm(), m.viewFormExitModal())
	case screenConfirmProfile:
		return overlayCenter(m.viewMain(), m.viewConfirmModal())
	case screenLogs:
		return m.viewLogs()
	case screenRunningAction:
		return overlayCenter(m.viewMain(), m.viewRunningActionModal())
	case screenStopConfirm:
		return overlayCenter(m.viewMain(), m.viewStopConfirmModal())
	case screenProfileTemplate:
		return m.viewTemplatePicker()
	case screenExportArgs:
		return overlayCenter(m.viewMain(), m.viewExportArgsModal())
	case screenNetworkSwitch:
		return overlayCenter(m.viewMain(), m.viewNetworkSwitchModal())
	case screenNetworkPicker:
		return overlayCenter(m.viewMain(), m.viewNetworkPickerModal())
	case screenRPCServerAction:
		return overlayCenter(m.viewMain(), m.viewRPCServerActionModal())
	case screenRPCLayerSplit:
		return overlayCenter(m.viewMain(), m.viewRPCLayerSplitModal())
	default:
		return m.viewMain()
	}
}

func (m Model) viewMain() string {
	if m.leftMode == modeOverview {
		return m.viewOverviewPage()
	}

	leftStyle := paneStyle
	if m.focus != focusRunning {
		leftStyle = focusedPaneStyle
	}

	leftW, rightW, targetLeftH := m.paneDimensions()
	// mirrors each box's Width+Padding (but not its border) so measuring
	// content through it produces the same word-wrapped line count the
	// real bordered box will render, instead of undercounting rows added
	// by wrapping long lines.
	leftMeasureStyle := lipgloss.NewStyle().Width(leftW).Padding(0, 1)
	rightMeasureStyle := lipgloss.NewStyle().Width(rightW).Padding(0, 1)

	leftContent := m.renderLeftPaneContent(leftW)

	// leftH starts from whichever is taller: the left pane's own natural
	// (wrapped) content, or the terminal-derived target — so panes fill
	// the terminal when content is short, without clipping it when it's
	// not. Measuring through leftMeasureStyle (not raw lipgloss.Height)
	// matters: a long model/profile name wraps at the real render width,
	// and undercounting that made the left box quietly grow taller than
	// what the right column's height was calculated from.
	leftH := max(lipgloss.Height(leftMeasureStyle.Render(leftContent)), targetLeftH)

	var right string
	if m.leftMode == modeRunning || m.leftMode == modeRPCServer {
		// These tabs manage their content entirely in the left pane; the
		// right column shows a single log-tail box rather than the usual
		// stacked Running+Details pair, which would repeat the same data.
		right = m.renderRunningOutputColumn(rightW, leftH)
	} else {
		runningBoxStyle := paneStyle
		if m.focus == focusRunning {
			runningBoxStyle = focusedPaneStyle
		}

		runningContent := m.renderRunning()
		detailsContent := m.renderDetails(rightW)
		runningH, detailsH := m.computeSplitHeights(leftH)

		// Content always wins over the computed floor — this keeps the layout
		// math consistent whether or not the user has dragged the split.
		if n := lipgloss.Height(rightMeasureStyle.Render(runningContent)); n > runningH {
			runningH = n
			// Shrink details to keep the total within budget so the body
			// height stays predictable and the hotkey line stays on screen.
			budget := leftH - 2
			if budget < 8 {
				budget = 8
			}
			detailsH = budget - runningH
			if detailsH < 3 {
				detailsH = 3
			}
		}
		rightAsLeftH := runningH + detailsH + 2 // right stacks 2 boxes' borders vs left's 1
		switch {
		case rightAsLeftH > leftH:
			leftH = rightAsLeftH
		case leftH > rightAsLeftH:
			detailsH += leftH - rightAsLeftH
		}

		runningBox := runningBoxStyle.Width(rightW).Height(runningH).Render(runningContent)
		detailsContent = m.renderDetailsWindow(detailsContent, rightW, detailsH)
		detailsBox := paneStyle.Width(rightW).Height(detailsH).Render(detailsContent)
		right = lipgloss.JoinVertical(lipgloss.Left, runningBox, detailsBox)
	}

	left := leftStyle.Width(leftW).Height(leftH).Render(leftContent)
	body := lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	headerLine := m.renderHeaderLine(lipgloss.Width(body))

	var b strings.Builder
	b.WriteString(headerLine)
	b.WriteString("\n")
	b.WriteString(body)
	b.WriteString("\n")
	switch {
	case m.starting:
		b.WriteString(loadingStyle.Render("starting " + m.startingLabel + "..."))
		b.WriteString("\n")
	case m.stopping:
		b.WriteString(loadingStyle.Render("stopping " + m.stoppingLabel + "..."))
		b.WriteString("\n")
	case m.netSwitching:
		b.WriteString(loadingStyle.Render("switching network..."))
		b.WriteString("\n")
	case m.err != nil:
		summary := "error: " + firstLine(m.err.Error())
		if m.errLogPath != "" {
			summary += "  (press e for logs)"
		}
		b.WriteString(errorStyle.Render(summary))
		b.WriteString("\n")
	}
	b.WriteString(helpStyle.Render(m.helpText()))
	return b.String()
}

// firstLine returns s up to (not including) its first newline, so a
// multi-line error (a log tail, a stack trace, …) collapses to a single
// summary line instead of blowing up the footer's height — the full text
// is still reachable via the log viewer.
func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

func (m Model) helpText() string {
	if m.searchEditing {
		return "type to filter · enter confirm · esc cancel"
	}
	if m.focus == focusSettingsContent && (m.settings.dirs.editing || m.settings.bin.editing) {
		return "enter save · esc cancel"
	}
	if m.leftMode == modeOverview {
		return "← (a) / → (d)  ·  q quit"
	}
	if m.focus == focusLeft && m.modelSubTabFocused {
		return "←→/ad switch view · ↓ enter list · q quit"
	}
	switch m.focus {
	case focusTabs:
		return "←→/ad tabs · ↑↓/ws select · q quit"
	case focusRunning:
		return "↑↓/wasd move · enter stop · e logs · q quit"
	case focusSettingsContent:
		return "↑↓/wasd move · enter edit · del delete · esc back"
	case focusLeft:
		switch m.leftMode {
		case modeRunning:
			return "↑↓/wasd move · enter/space stop · c copy endpoint · q quit"
		case modeRPCServer:
			return "enter start/stop · e view output · q quit"
		case modeRecents:
			return "↑↓/wasd move · enter/space run · q quit"
		case modeSettings:
			return "↑↓/wasd move · enter/space select · q quit"
		case modeNetwork:
			if m.netCursor >= netRowSetInternet {
				return "↑↓/wasd move · enter pick connection · q quit"
			}
			return "↑↓/wasd move · enter switch · q quit"
		default: // modeModels
			return "↑↓/wasd move · enter/space run · c copy · / search · del delete · q quit"
		}
	}
	return "←→/ad tabs · ↑↓/wasd move · q quit"
}

// renderLeftPaneContent renders whichever tab's content is active. The
// outer Models/Recents/Settings tab bar lives outside the boxes, in
// renderHeaderLine.
func (m Model) renderLeftPaneContent(leftW int) string {
	switch m.leftMode {
	case modeOverview:
		return "" // overview takes the full width; left pane is not used
	case modeModels, modeRecents:
		subHeader := m.renderModelsSubTab()
		var list string
		if m.leftMode == modeRecents {
			list = m.renderRecentsList(leftW)
		} else {
			list = m.renderModelsTree(leftW)
		}
		return subHeader + "\n" + list
	case modeSettings:
		return m.renderSettingsList(leftW)
	case modeRunning:
		return m.renderRunningTabList(leftW)
	case modeNetwork:
		if m.networkTabVisible() {
			return m.renderNetworkList(leftW)
		}
		return m.renderModelsTree(leftW)
	case modeRPCServer:
		return m.renderRPCServerTab()
	default:
		return m.renderModelsTree(leftW)
	}
}

// tailFittingHeight reads logPath's trailing output and drops the oldest
// lines until what remains word-wraps to at most maxLines rows at
// boxWidth — process output can contain very long or numerous lines (a
// chat template dump, say), which by raw line count alone could still
// wrap well past the available space.
func tailFittingHeight(logPath string, boxWidth, maxLines int) string {
	raw, err := process.TailLog(logPath, 500)
	if err != nil || raw == "" {
		return ""
	}

	lines := wrappedLogPreviewLines(raw, boxWidth)
	if len(lines) == 0 {
		return ""
	}
	if len(lines) > maxLines {
		lines = lines[len(lines)-maxLines:]
	}
	return strings.Join(lines, "\n")
}

func wrappedLogPreviewLines(raw string, boxWidth int) []string {
	innerWidth := formDescriptionTextWidth(boxWidth)
	if innerWidth <= 0 {
		innerWidth = formDescriptionTextWidth(minRightWidth)
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
	s = stripANSI(s)
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

