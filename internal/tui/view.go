package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/sockheadrps/llmctl/internal/build"
	"github.com/sockheadrps/llmctl/internal/health"
	"github.com/sockheadrps/llmctl/internal/models"
	"github.com/sockheadrps/llmctl/internal/process"
	"github.com/sockheadrps/llmctl/internal/util"
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

// paneDimensions works out how big the main screen's panes should be so
// they fill the actual terminal instead of shrink-wrapping their content
// with empty space around them. leftWidth/rightWidth are content widths
// (Width() values — each box's border adds 2 more on render); targetLeftH
// is the content-height budget the left pane (and thus the whole layout)
// should fill, matching splitPaneHeight's units.
func (m Model) paneDimensions() (leftWidth, rightWidth, targetLeftH int) {
	termWidth, termHeight := m.width, m.height
	if termWidth <= 0 {
		termWidth = fallbackWidth
	}
	if termHeight <= 0 {
		termHeight = fallbackHeight
	}

	// The two boxes sit side by side with no gap; each has 2 columns of
	// border overhead not covered by its Width() value.
	avail := termWidth - 4
	if m.leftWidthOverride > 0 {
		leftWidth = m.leftWidthOverride
		if leftWidth < minLeftWidth {
			leftWidth = minLeftWidth
		}
		if leftWidth > avail-minRightWidth {
			leftWidth = avail - minRightWidth
		}
	} else {
		leftWidth = avail * 2 / 5
		if leftWidth < minLeftWidth {
			leftWidth = minLeftWidth
		}
	}
	rightWidth = avail - leftWidth
	if rightWidth < minRightWidth {
		rightWidth = minRightWidth
	}

	targetLeftH = termHeight - chromeHeight - 2 // -2 for the left box's own border
	if targetLeftH < minBodyHeight {
		targetLeftH = minBodyHeight
	}
	return leftWidth, rightWidth, targetLeftH
}

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
	default:
		return m.viewMain()
	}
}

func (m Model) viewMain() string {
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
	if m.leftMode == modeRunning {
		// The Running tab's own list already lives in the left pane, so
		// the right column collapses to a single box — the selected
		// instance's output — instead of the usual stacked Running+Details
		// boxes, which would just repeat the same list a second time.
		// Unlike the other panes, this content is unbounded (raw process
		// output), so it's fit *into* leftH rather than growing leftH to
		// match it — otherwise a large log tail blows both boxes past the
		// terminal's actual height.
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
		case modeRecents:
			return "↑↓/wasd move · enter/space run · q quit"
		case modeSettings:
			return "↑↓/wasd move · enter/space select · q quit"
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
	case modeRecents:
		return m.renderRecentsList(leftW)
	case modeSettings:
		return m.renderSettingsList(leftW)
	case modeRunning:
		return m.renderRunningTabList(leftW)
	default:
		return m.renderModelsTree(leftW)
	}
}

// renderHeaderLine renders the Models/Recents/Settings tab strip above the
// two boxes, centered over totalWidth (their combined rendered width) with
// dashes filling the rest, like a notebook tab strip sitting on the boxes'
// border.
func (m Model) renderHeaderLine(totalWidth int) string {
	return dashWrap(totalWidth, m.renderTabBarLabels(), m.focus == focusTabs)
}

// dashWrap centers content over totalWidth, padding both sides with a
// horizontal rule. The rule brightens when focused, matching how the
// boxes below signal focus via border color.
func dashWrap(totalWidth int, content string, focused bool) string {
	remaining := totalWidth - lipgloss.Width(content)
	if remaining < 2 {
		return content
	}
	left := remaining / 2
	right := remaining - left

	dashStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("30"))
	return dashStyle.Render(strings.Repeat("─", left)) + content + dashStyle.Render(strings.Repeat("─", right))
}

// renderTabBarLabels shows the Models/Recents/Settings/Running tabs, with
// the active tab styled as a filled chip.
func (m Model) renderTabBarLabels() string {
	tabs := []struct {
		mode  leftMode
		label string
	}{
		{modeModels, "Models"},
		{modeRecents, "Recents"},
		{modeSettings, "Settings"},
		{modeRunning, "Running"},
	}

	tabFocused := m.focus == focusTabs
	rendered := make([]string, len(tabs))
	for i, t := range tabs {
		if m.leftMode == t.mode {
			color := lipgloss.Color("39")
			if !tabFocused {
				color = lipgloss.Color("24")
			}
			rendered[i] = lipgloss.NewStyle().
				Foreground(color).
				Bold(true).
				Underline(true).
				Render(t.label)
		} else {
			rendered[i] = profileStyle.Render(t.label)
		}
	}

	return strings.Join(rendered, "  ")
}

// renderSettingsList shows the Settings tab's menu of configuration
// sub-pages (currently just Model Directories).
func (m Model) renderSettingsList(width int) string {
	var b strings.Builder
	rows := buildSettingsRows()
	textWidth := formRowTextWidth(width)
	inSettings := m.focus == focusLeft || m.focus == focusSettingsContent
	for i, r := range rows {
		selected := i == m.settingsCursor
		cursor := "  "
		style := profileStyle
		if selected && inSettings {
			if m.focus == focusLeft {
				cursor = cursorStyle.Render("> ")
			}
			style = activeModelStyle
		}
		label := truncateText(r.label, max(1, textWidth-lipgloss.Width(cursor)))
		b.WriteString(fmt.Sprintf("%s%s\n", cursor, style.Render(label)))
	}
	b.WriteString("\n")
	b.WriteString(detailMutedStyle.Render("llmctl " + build.Version))
	b.WriteString("\n")
	return b.String()
}

func (m Model) renderModelsTree(width int) string {
	var b strings.Builder
	textWidth := formRowTextWidth(width)

	if len(m.cfg.ModelsDirs) == 0 {
		b.WriteString(modelStyle.Render("No models folders set."))
		b.WriteString("\n")
		b.WriteString(profileStyle.Render("Please navigate to Settings > Model Directories"))
		b.WriteString("\n")
		b.WriteString(profileStyle.Render("to add a directory to load models from."))
		return b.String()
	}

	if m.modelSearch != "" || m.searchEditing {
		query := m.modelSearch
		if query == "" {
			query = "type to filter..."
		}
		prefix := "/ "
		if m.searchEditing {
			prefix = cursorStyle.Render("/ ")
		}
		b.WriteString(prefix + detailMutedStyle.Render(truncateText(query, max(1, textWidth-lipgloss.Width(prefix)))))
		b.WriteString("\n\n")
	}

	if len(m.rows) == 0 {
		empty := "(no models configured)"
		if m.modelSearch != "" {
			empty = "(no matches)"
		}
		b.WriteString(profileStyle.Render(empty))
		b.WriteString("\n")
		b.WriteString(helpStyle.Render("press A to add a model from your configured folders"))
		return b.String()
	}

	focused := m.focus == focusLeft
	for i, r := range m.rows {
		selected := i == m.cursor
		active := selected && focused
		cursor := "  "
		if active {
			cursor = cursorStyle.Render("> ")
		}

		switch r.kind {
		case rowModel:
			// A blank line ahead of every model but the first gives the
			// (now profile-less by default) list some breathing room
			// instead of a wall of names packed edge to edge.
			if i > 0 {
				b.WriteString("\n")
			}
			// The selected model stands out; every other one dims back
			// so it doesn't compete for attention.
			style := m.modelRowStyle(r, active)
			dot := ""
			if isRunning, status := m.modelRunningStatus(r.modelKey); isRunning {
				switch status {
				case health.StatusUp:
					dot = runningStyle.Render("● ")
				case health.StatusDown:
					dot = downStyle.Render("● ")
				default:
					dot = loadingStyle.Render("● ")
				}
			}
			label := truncateText(r.label, max(1, textWidth-lipgloss.Width(cursor)-lipgloss.Width(dot)))
			b.WriteString(fmt.Sprintf("%s%s%s\n", cursor, dot, style.Render(label)))

		case rowProfile:
			label := r.label
			style := profileStyle
			switch {
			case r.modelKey == m.pendingDeleteModel && r.profileKey == m.pendingDeleteProfile:
				style = pendingDeleteStyle
				label += " (del again to confirm)"
			case active:
				style = selectedProfileStyle
			}
			label = truncateText(label, max(1, textWidth-lipgloss.Width(cursor)-2))
			b.WriteString(fmt.Sprintf("%s  %s\n", cursor, style.Render(label)))

		case rowAddProfile:
			style := addStyle
			if active {
				style = selectedAddStyle
			}
			label := truncateText(r.label, max(1, textWidth-lipgloss.Width(cursor)-2))
			b.WriteString(fmt.Sprintf("%s  %s\n", cursor, style.Render(label)))

		case rowAddModel:
			b.WriteString("\n")
			style := addStyle
			if active {
				style = selectedAddStyle
			}
			label := truncateText(r.label, max(1, textWidth-lipgloss.Width(cursor)))
			b.WriteString(fmt.Sprintf("%s%s\n", cursor, style.Render(label)))
		}
	}
	return b.String()
}

func (m Model) modelRowStyle(r row, active bool) lipgloss.Style {
	switch {
	case active:
		return selectedProfileStyle
	case m.modelProfilesMode && r.modelKey == m.expandedModelKey:
		return activeModelStyle
	default:
		return profileStyle
	}
}

// renderRecentsList shows up to models.RecentLimit most recently run
// profiles, most recent first, for quick re-selection.
func (m Model) renderRecentsList(width int) string {
	var b strings.Builder

	if len(m.recentRows) == 0 {
		b.WriteString(profileStyle.Render("(nothing run yet)"))
		return b.String()
	}

	textWidth := formRowTextWidth(width)
	focused := m.focus == focusLeft
	for i, r := range m.recentRows {
		selected := i == m.recentCursor
		active := selected && focused
		cursor := "  "
		style := profileStyle
		if active {
			cursor = cursorStyle.Render("> ")
			style = selectedProfileStyle
		}
		label := truncateText(r.label, max(1, textWidth-lipgloss.Width(cursor)))
		b.WriteString(fmt.Sprintf("%s%s\n", cursor, style.Render(label)))
	}
	return b.String()
}

func (m Model) renderRunning() string {
	var b strings.Builder
	title := "Running"
	if m.gpuName != "" {
		title = m.gpuName
	}
	b.WriteString(modelStyle.Render(title))
	b.WriteString("\n")
	if vram := m.renderVRAMHeader(); vram != "" {
		b.WriteString(infoStyle.Render(vram))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	if len(m.running) == 0 {
		b.WriteString(profileStyle.Render("(nothing running)"))
		return b.String()
	}

	focused := m.focus == focusRunning
	for i, r := range m.running {
		b.WriteString(m.renderRunningRow(r, i == m.runningCursor, focused))
	}
	return b.String()
}

// renderRunningRow renders one running instance's list line — a status
// dot, its label/port, tok/s if actively generating, and VRAM usage if
// known. Shared between the persistent glance box (renderRunning) and the
// Running tab's left-pane list (renderRunningTabList), which differ only
// in which focus state highlights the selected row.
func (m Model) renderRunningRow(r models.Running, selected, focused bool) string {
	return m.renderRunningRowWithWidth(r, selected, focused, 0)
}

func (m Model) renderRunningRowWithWidth(r models.Running, selected, focused bool, width int) string {
	dot := loadingStyle.Render("●")
	badge := loadingStyle.Render("loading")
	switch m.health[r.ModelKey+"/"+r.ProfileKey] {
	case health.StatusUp:
		dot = runningStyle.Render("●")
		badge = runningStyle.Render("up")
	case health.StatusDown:
		dot = downStyle.Render("●")
		badge = downStyle.Render("down")
	}

	cursor := "  "
	labelStyle := profileStyle
	if selected {
		if focused {
			cursor = cursorStyle.Render("> ")
			labelStyle = selectedProfileStyle
		} else {
			cursor = profileStyle.Render("> ")
		}
	}

	key := r.ModelKey + "/" + r.ProfileKey
	text := fmt.Sprintf("%-24s :%d", r.Label(), r.Port)
	if rate, ok := m.tokRates[key]; ok {
		text += fmt.Sprintf("  %.1f tok/s", rate)
	}
	if mb, ok := m.gpuByPID[r.PID]; ok {
		text += fmt.Sprintf("  %.1fG", float64(mb)/1024)
	}
	if width > 0 {
		text = truncateText(text, max(1, formRowTextWidth(width)-lipgloss.Width(cursor)-2))
	}
	row := fmt.Sprintf("%s%s %s %s\n", cursor, dot, badge, labelStyle.Render(text))

	if rate, ok := m.tokRates[key]; ok {
		row += "   " + m.renderRateMeter(key, rate) + "\n"
	} else if peak := m.tokPeak[key]; peak > 0 {
		// model is idle but has a session history — show the meter at zero
		row += "   " + m.renderRateMeter(key, 0) + "\n"
	}
	return row
}

// renderRunningTabList shows every running instance in the Running tab's
// left pane — the same data as renderRunning's glance box, just styled as
// a selectable list like the other tabs' left-pane content.
func (m Model) renderRunningTabList(width int) string {
	var b strings.Builder

	if len(m.running) == 0 {
		b.WriteString(profileStyle.Render("(nothing running)"))
		return b.String()
	}

	focused := m.focus == focusLeft
	for i, r := range m.running {
		b.WriteString(m.renderRunningRowWithWidth(r, i == m.runningCursor, focused, width))
	}
	return b.String()
}

// renderRunningOutputColumn builds the Running tab's single right-hand
// box — a live-updating tail of the selected instance's output — in place
// of the usual stacked Running+Details boxes. Unlike other panes, this
// content is unbounded raw process output (llama.cpp can dump a multi-line
// chat template in one go), so it's fit *into* leftH rather than growing
// leftH to match it — otherwise a large tail would blow the box, and its
// own border, straight past the terminal's actual height.
func (m Model) renderRunningOutputColumn(rightW, leftH int) string {
	innerH := leftH
	if innerH < 1 {
		innerH = 1
	}
	content := m.renderRunningOutputPane(rightW, innerH)
	return paneStyle.Width(rightW).Height(innerH).Render(content)
}

// renderRunningOutputPane shows a live tail of the currently focused
// running instance's captured stdout/stderr (the same log file the crash
// detector and 'e' log viewer already read from), trimmed to fit innerH
// rows once word-wrapped at rightW.
func (m Model) renderRunningOutputPane(rightW, innerH int) string {
	var b strings.Builder

	if m.focus == focusTabs {
		b.WriteString(modelStyle.Render(tabTitle(m.leftMode)))
		b.WriteString("\n\n")
		b.WriteString(profileStyle.Render(tabInstructions(m.leftMode)))
		return b.String()
	}

	r, ok := m.currentRow()
	if !ok || r.kind != rowRunning {
		b.WriteString(modelStyle.Render("Output"))
		b.WriteString("\n\n")
		b.WriteString(profileStyle.Render("(nothing running)"))
		return b.String()
	}

	run, ok := m.findRunning(r.modelKey, r.profileKey)
	if !ok {
		b.WriteString(modelStyle.Render(r.label))
		b.WriteString("\n\n")
		b.WriteString(profileStyle.Render("(no longer running)"))
		return b.String()
	}

	header := modelStyle.Render(fmt.Sprintf("%s  :%d", run.Label(), run.Port))
	help := helpStyle.Render("enter to stop or view full output")

	key := run.ModelKey + "/" + run.ProfileKey
	hasMeter := m.tokPeak[key] > 0

	// header + (meter) + blank + tail + blank + help
	overhead := 4
	if hasMeter {
		overhead = 5
	}
	budget := innerH - overhead
	if budget < 1 {
		budget = 1
	}

	fmt.Fprintf(&b, "%s\n", header)
	if hasMeter {
		rate := m.tokRates[key]
		b.WriteString(m.renderRateMeter(key, rate) + "\n")
	}
	b.WriteString("\n")
	if tail := tailFittingHeight(run.LogFile, rightW, budget); tail != "" {
		b.WriteString(profileStyle.Render(tail))
	} else {
		b.WriteString(profileStyle.Render("(no output yet)"))
	}
	b.WriteString("\n\n")
	b.WriteString(help)
	return b.String()
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

// renderVRAMHeader renders a compact used/total VRAM bar for the "Running"
// header line. Returns "" when nvidia-smi isn't available or hasn't
// reported anything yet.
func (m Model) renderVRAMHeader() string {
	if !m.gpuAvailable || m.gpuUsage.TotalMiB <= 0 {
		return ""
	}

	const barWidth = 10
	frac := float64(m.gpuUsage.UsedMiB) / float64(m.gpuUsage.TotalMiB)
	filled := int(frac * barWidth)
	if filled > barWidth {
		filled = barWidth
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

	barStyle := runningStyle
	switch {
	case frac >= 0.9:
		barStyle = downStyle
	case frac >= 0.7:
		barStyle = loadingStyle
	}

	usedGB := float64(m.gpuUsage.UsedMiB) / 1024
	totalGB := float64(m.gpuUsage.TotalMiB) / 1024
	return fmt.Sprintf("%s %s", barStyle.Render(bar), profileStyle.Render(fmt.Sprintf("%.1f/%.1fG", usedGB, totalGB)))
}

// computeSplitHeights returns the running/details content heights for the
// right column, honouring a user-dragged rightSplitOverride when set.
func (m Model) computeSplitHeights(leftH int) (runningH, detailsH int) {
	if m.rightSplitOverride > 0 {
		budget := leftH - 2
		if budget < 8 {
			budget = 8
		}
		runningH = m.rightSplitOverride
		if runningH < 3 {
			runningH = 3
		}
		if runningH > budget-3 {
			runningH = budget - 3
		}
		detailsH = budget - runningH
		if detailsH < 3 {
			detailsH = 3
		}
		return
	}
	return splitPaneHeight(leftH, len(m.running))
}

// splitPaneHeight divides the right column so its total rendered height
// (including each box's own border) matches leftHeight, the left pane's
// content height. Running entries are one line each, so that box is sized
// to just fit runningCount (with a small minimum/cap), while the Details
// pane stays compact so a selected model preview does not stretch the full
// interface and push the top of the TUI out of view.
func splitPaneHeight(leftHeight, runningCount int) (running, details int) {
	// The right column stacks two bordered boxes instead of the left's
	// one, so it has two extra border lines to account for.
	budget := leftHeight - 2
	if budget < 8 {
		budget = 8
	}

	running = runningCount + 3 // header + blank + at least one row
	if running < 5 {
		running = 5
	}
	if max := budget * 2 / 5; running > max {
		running = max
	}

	details = budget - running
	if details < 5 {
		details = 5
	}
	if details > 12 {
		details = 12
	}
	return running, details
}

func (m Model) renderDetailsWindow(content string, width, height int) string {
	lines := wrappedContentLines(content, width)
	return strings.Join(contentWindow(lines, height, m.detailsScroll), "\n")
}

func wrappedContentLines(content string, width int) []string {
	innerWidth := formDescriptionTextWidth(width)
	if innerWidth <= 0 {
		innerWidth = formDescriptionTextWidth(minRightWidth)
	}
	rendered := lipgloss.NewStyle().Width(innerWidth).Render(strings.TrimRight(content, "\n"))
	lines := strings.Split(strings.TrimRight(rendered, "\n"), "\n")
	if len(lines) == 0 {
		return []string{""}
	}
	return lines
}

func contentWindow(lines []string, visible, offset int) []string {
	if visible <= 0 {
		return nil
	}
	if len(lines) == 0 {
		return []string{""}
	}

	maxOffset := max(0, len(lines)-visible)
	offset = max(0, min(offset, maxOffset))

	window := make([]string, 0, min(visible, len(lines)))
	for i := offset; i < min(offset+visible, len(lines)); i++ {
		window = append(window, lines[i])
	}
	return window
}

func (m Model) mainDetailsLineCount() int {
	rightW, _, detailsH := m.mainDetailsGeometry()
	if detailsH <= 0 {
		return 0
	}
	return len(wrappedContentLines(m.renderDetails(rightW), rightW))
}

func (m Model) mainDetailsVisibleLines() int {
	_, _, detailsH := m.mainDetailsGeometry()
	return detailsH
}

func (m Model) mainDetailsGeometry() (rightW, runningH, detailsH int) {
	leftW, rightW, targetLeftH := m.paneDimensions()
	leftMeasureStyle := lipgloss.NewStyle().Width(leftW).Padding(0, 1)
	rightMeasureStyle := lipgloss.NewStyle().Width(rightW).Padding(0, 1)

	leftH := max(lipgloss.Height(leftMeasureStyle.Render(m.renderLeftPaneContent(leftW))), targetLeftH)
	runningH, detailsH = m.computeSplitHeights(leftH)
	if n := lipgloss.Height(rightMeasureStyle.Render(m.renderRunning())); n > runningH {
		runningH = n
		budget := leftH - 2
		if budget < 8 {
			budget = 8
		}
		detailsH = budget - runningH
		if detailsH < 3 {
			detailsH = 3
		}
	}

	rightAsLeftH := runningH + detailsH + 2
	if leftH > rightAsLeftH {
		detailsH += leftH - rightAsLeftH
	}
	return rightW, runningH, detailsH
}

func (m *Model) resetDetailsScroll() {
	m.detailsScroll = 0
	m.detailsDir = 1
	m.detailsPause = scrollPauseTicks
}

func (m *Model) advanceDetailsScroll(lines, visible int) {
	m.detailsScroll, m.detailsDir, m.detailsPause = advanceAutoScroll(m.detailsScroll, m.detailsDir, m.detailsPause, lines, visible)
}

func advanceAutoScroll(offset, dir, pause, lines, visible int) (int, int, int) {
	if dir == 0 {
		dir = 1
	}
	maxScroll := max(0, lines-visible)
	if maxScroll == 0 {
		return 0, 1, 0
	}

	if pause > 0 {
		return offset, dir, pause - 1
	}

	offset += dir
	if offset >= maxScroll {
		return maxScroll, -1, scrollPauseTicks
	}
	if offset <= 0 {
		return 0, 1, scrollPauseTicks
	}
	return offset, dir, 0
}

// modelSourceLine describes where a model's weights come from: an on-disk
// GGUF file (with its size) or a HuggingFace repo fetched by llama-server.
func modelSourceLine(mdl models.Model) string {
	if mdl.IsRemote() {
		return "hf: " + mdl.HFRepo
	}
	size := "unknown size"
	if info, err := os.Stat(mdl.Path); err == nil {
		size = util.FormatBytes(info.Size())
	}
	return filepath.Base(mdl.Path) + " (" + size + ")"
}

// renderModelPreview shows a collapsed model's profiles as a quick preview
// — port and a couple of key settings each — so you can see what's there
// without expanding it. Enter expands the model into the tree for the full
// per-profile Details view.
func (m Model) renderModelPreview(modelKey string) string {
	var b strings.Builder

	mdl, ok := m.cfg.Models[modelKey]
	if !ok {
		return b.String()
	}

	fmt.Fprintf(&b, "%s\n\n", profileStyle.Render(modelSourceLine(mdl)))

	profileKeys := make([]string, 0, len(mdl.Profiles))
	for pk := range mdl.Profiles {
		profileKeys = append(profileKeys, pk)
	}
	sort.Strings(profileKeys)

	b.WriteString(modelStyle.Render("Profiles:"))
	b.WriteString("\n")
	for _, pk := range profileKeys {
		p := mdl.Profiles[pk]
		text := fmt.Sprintf("%-16s :%d", p.Name, p.Port)
		if p.Temp != nil {
			text += fmt.Sprintf("  temp %.2g", *p.Temp)
		}
		b.WriteString(detailMutedStyle.Render("• "+text) + "\n")
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("enter to expand and select a profile"))
	return b.String()
}

// tabTitle and tabInstructions describe each tab for the Details panel
// while focus is still at the outer tab bar — before arrowing down into a
// tab, there's no row selected yet to show details for, so explain what's
// in there instead of falling back to a generic empty-state message.
func tabTitle(mode leftMode) string {
	switch mode {
	case modeRecents:
		return "Recents"
	case modeSettings:
		return "Settings"
	case modeRunning:
		return "Running"
	default:
		return "Models"
	}
}

func tabInstructions(mode leftMode) string {
	switch mode {
	case modeRecents:
		return "Select from your most recently run profiles to quickly re-run one."
	case modeSettings:
		return "Select a settings category to configure, like where llmctl looks for model files."
	case modeRunning:
		return "Select a running instance to preview its output. Enter to stop it or view the full output."
	default:
		return "Select from saved model profiles, or add new model profiles."
	}
}

// renderRateMeter renders a horizontal bar showing current tok/s relative to
// the session peak, followed by the numeric rate.
func (m Model) renderRateMeter(key string, rate float64) string {
	const barWidth = 16
	peak := m.tokPeak[key]

	// Use the persisted all-time max as the bar ceiling — it survives restarts
	// and gives a stable scale from the first token of a new session.
	parts := strings.SplitN(key, "/", 2)
	if len(parts) == 2 {
		if mdl, ok := m.cfg.Models[parts[0]]; ok {
			if p, ok := mdl.Profiles[parts[1]]; ok && p.MaxTokPerSec > peak {
				peak = p.MaxTokPerSec
			}
		}
	}

	if peak <= 0 {
		peak = rate
	}
	filled := 0
	if peak > 0 {
		filled = int((rate / peak) * barWidth)
		if filled > barWidth {
			filled = barWidth
		}
	}
	bar := infoStyle.Render(strings.Repeat("█", filled)) +
		detailMutedStyle.Render(strings.Repeat("░", barWidth-filled))
	label := detailMutedStyle.Render(fmt.Sprintf("  %.1f tok/s", rate))
	if peak > rate && peak > 0 {
		label += detailMutedStyle.Render(fmt.Sprintf("  (peak %.1f)", peak))
	}
	return bar + label
}

// modelRunningStatus returns whether any profile for modelKey is running and
// the best health status among them (up beats loading beats down).
func (m Model) modelRunningStatus(modelKey string) (running bool, status health.Status) {
	for _, r := range m.running {
		if r.ModelKey != modelKey {
			continue
		}
		s := m.health[r.ModelKey+"/"+r.ProfileKey]
		if !running {
			running = true
			status = s
		}
		if s == health.StatusUp {
			status = s
			break
		}
	}
	return
}

// renderSparkline converts a slice of rate samples into a compact bar
// chart string using Unicode block elements, scaled to the slice max.
func renderSparkline(history []float64, width int) string {
	if len(history) == 0 || width <= 0 {
		return ""
	}
	peak := 0.0
	for _, v := range history {
		if v > peak {
			peak = v
		}
	}
	if peak == 0 {
		return ""
	}
	blocks := []rune("▁▂▃▄▅▆▇█")
	n := len(blocks)
	data := history
	if len(data) > width {
		data = data[len(data)-width:]
	}
	var b strings.Builder
	for _, v := range data {
		idx := int((v / peak) * float64(n-1))
		if idx < 0 {
			idx = 0
		} else if idx >= n {
			idx = n - 1
		}
		b.WriteRune(blocks[idx])
	}
	return b.String()
}

type detailPair struct {
	label string
	value string
}

func formatDetailPairs(pairs []detailPair, width int) []string {
	if width <= 0 {
		width = 36
	}

	if width < 56 {
		lines := make([]string, 0, len(pairs))
		for _, pair := range pairs {
			lines = append(lines, fmt.Sprintf("%s: %s", pair.label, pair.value))
		}
		return lines
	}

	lines := make([]string, 0, (len(pairs)+1)/2)
	for i := 0; i < len(pairs); i += 2 {
		left := fmt.Sprintf("%s: %s", pairs[i].label, pairs[i].value)
		if i+1 >= len(pairs) {
			lines = append(lines, left)
			continue
		}
		right := fmt.Sprintf("%s: %s", pairs[i+1].label, pairs[i+1].value)
		lines = append(lines, left+"    "+right)
	}
	return lines
}

// renderDetails shows the settings for the currently selected profile,
// including the backing model's on-disk file size and any notes. The
// header names whatever's actually focused instead of a static "Details"
// label, since that's more useful at a glance.
func (m Model) renderDetails(width int) string {
	var b strings.Builder

	// Still at the outer tab bar — nothing's selected yet within a tab,
	// so explain what arrowing down into it will show instead of an empty
	// "(select a profile...)" placeholder that doesn't fit Recents/Settings.
	if m.focus == focusTabs {
		b.WriteString(modelStyle.Render(tabTitle(m.leftMode)))
		b.WriteString("\n\n")
		b.WriteString(profileStyle.Render(tabInstructions(m.leftMode)))
		return b.String()
	}

	r, ok := m.currentRow()
	if !ok {
		b.WriteString(modelStyle.Render("Details"))
		b.WriteString("\n\n")
		b.WriteString(profileStyle.Render("(select a profile to see details)"))
		return b.String()
	}

	if r.kind == rowModel {
		mdl := m.cfg.Models[r.modelKey]
		b.WriteString(modelStyle.Render(mdl.Name))
		b.WriteString("\n")
		b.WriteString(m.renderModelPreview(r.modelKey))
		return b.String()
	}

	if r.kind == rowSettingsCategory {
		return m.renderSettingsDetail(r.modelKey)
	}

	if r.kind != rowProfile {
		b.WriteString(modelStyle.Render("Details"))
		b.WriteString("\n\n")
		b.WriteString(profileStyle.Render("(select a profile to see details)"))
		return b.String()
	}

	mdl, ok := m.cfg.Models[r.modelKey]
	if !ok {
		return b.String()
	}
	p, ok := mdl.Profiles[r.profileKey]
	if !ok {
		return b.String()
	}

	// Keep profile details compact so the preview doesn't expand the pane
	// enough to push the UI off-screen when a model is selected.
	fmt.Fprintf(&b, "%s\n", modelStyle.Render(mdl.Name+" / "+p.Name))
	fmt.Fprintf(&b, "%s\n", detailMutedStyle.Render(modelSourceLine(mdl)))
	if p.Notes != "" {
		b.WriteString(profileStyle.Render(p.Notes))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	dash := func(s string) string {
		if s == "" {
			return "-"
		}
		return s
	}

	boolDash := func(v *bool) string {
		if v == nil {
			return "-"
		}
		if *v {
			return "true"
		}
		return "false"
	}

	sections := []struct {
		name  string
		pairs []detailPair
	}{
		{
			name: "Profile",
			pairs: []detailPair{
				{label: "Port", value: fmt.Sprint(p.Port)},
				{label: "Ctx Size", value: dash(intOrEmpty(p.CtxSize))},
			},
		},
		{
			name: "Sampling",
			pairs: []detailPair{
				{label: "Temp", value: dash(floatPtrOrEmpty(p.Temp))},
				{label: "Top P", value: dash(floatPtrOrEmpty(p.TopP))},
				{label: "Top K", value: dash(intPtrOrEmpty(p.TopK))},
				{label: "Min P", value: dash(floatPtrOrEmpty(p.MinP))},
				{label: "Presence Pen", value: dash(floatPtrOrEmpty(p.PresencePenalty))},
				{label: "Repeat Pen", value: dash(floatPtrOrEmpty(p.RepetitionPenalty))},
				{label: "Freq Pen", value: dash(floatPtrOrEmpty(p.FrequencyPenalty))},
				{label: "Seed", value: dash(intPtrOrEmpty(p.Seed))},
			},
		},
		{
			name: "Runtime",
			pairs: []detailPair{
				{label: "Flash Attn", value: fmt.Sprint(p.FlashAttn)},
				{label: "GPU Layers", value: fmt.Sprint(p.GPULayers)},
				{label: "MMap", value: boolDash(p.MMap)},
				{label: "KV Offload", value: boolDash(p.KVOffload)},
				{label: "Parallel", value: dash(intPtrOrEmpty(p.Parallel))},
				{label: "Cont Batching", value: boolDash(p.ContBatching)},
			},
		},
		{
			name: "Cache",
			pairs: []detailPair{
				{label: "Cache K", value: dash(p.CacheTypeK)},
				{label: "Cache V", value: dash(p.CacheTypeV)},
				{label: "Cache Prompt", value: boolDash(p.CachePrompt)},
				{label: "Cache RAM", value: dash(intPtrOrEmpty(p.CacheRAM))},
			},
		},
		{
			name: "Reasoning",
			pairs: []detailPair{
				{label: "Reasoning", value: dash(p.Reasoning)},
				{label: "Budget", value: dash(intPtrOrEmpty(p.ReasoningBudget))},
				{label: "Format", value: dash(p.ReasoningFormat)},
			},
		},
	}

	// Always show all pairs — dash("-") for unset so the full picture is visible.

	if width >= 70 {
		leftSections := []struct {
			name  string
			pairs []detailPair
		}{sections[0], sections[1]}
		rightSections := []struct {
			name  string
			pairs []detailPair
		}{sections[2], sections[3], sections[4]}

		columnWidth := (width - 3) / 2
		if columnWidth < 24 {
			columnWidth = width
		}

		formatSection := func(section struct {
			name  string
			pairs []detailPair
		}) string {
			var sectionBuilder strings.Builder
			sectionBuilder.WriteString(modelStyle.Render(section.name))
			sectionBuilder.WriteString("\n")
			for _, line := range formatDetailPairs(section.pairs, columnWidth) {
				sectionBuilder.WriteString(profileStyle.Render(line))
				sectionBuilder.WriteString("\n")
			}
			return sectionBuilder.String()
		}

		leftColumn := strings.Builder{}
		for _, section := range leftSections {
			if section.name != "" {
				if leftColumn.Len() > 0 {
					leftColumn.WriteString("\n")
				}
				leftColumn.WriteString(formatSection(section))
			}
		}

		rightColumn := strings.Builder{}
		for _, section := range rightSections {
			if section.name != "" {
				if rightColumn.Len() > 0 {
					rightColumn.WriteString("\n")
				}
				rightColumn.WriteString(formatSection(section))
			}
		}

		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top,
			lipgloss.NewStyle().Width(columnWidth).Render(leftColumn.String()),
			lipgloss.NewStyle().Width(columnWidth).Render(rightColumn.String()),
		))
		b.WriteString("\n")
	} else {
		for _, section := range sections {
			if len(section.pairs) == 0 {
				continue
			}
			if section.name != "" {
				b.WriteString("\n")
				b.WriteString(modelStyle.Render(section.name))
				b.WriteString("\n")
			}
			for _, line := range formatDetailPairs(section.pairs, width) {
				b.WriteString(profileStyle.Render(line))
				b.WriteString("\n")
			}
		}
	}

	extraArgs := dash(strings.Join(p.ExtraArgs, " "))
	if extraArgs != "-" {
		b.WriteString("\n")
		b.WriteString(modelStyle.Render("Extra Args"))
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().Width(width).Render(profileStyle.Render(extraArgs)))
	} else {
		b.WriteString("\n")
		b.WriteString(modelStyle.Render("Extra Args"))
		b.WriteString("\n")
		b.WriteString(profileStyle.Render("-"))
	}

	return b.String()
}
