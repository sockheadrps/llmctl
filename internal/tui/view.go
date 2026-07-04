package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"

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
	leftWidth = avail * 2 / 5
	if leftWidth < minLeftWidth {
		leftWidth = minLeftWidth
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
		return m.viewForm()
	case screenConfirmProfile:
		return overlayCenter(m.viewMain(), m.viewConfirmModal())
	case screenLogs:
		return m.viewLogs()
	case screenRunningAction:
		return overlayCenter(m.viewMain(), m.viewRunningActionModal())
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
		detailsContent := m.renderDetails()
		runningH, detailsH := splitPaneHeight(leftH, len(m.running))

		// lipgloss's Height() is a floor, not a cap — content taller than
		// its preferred share (many settings + a long Notes field, lots of
		// running instances, …) just overflows past it. Let content win
		// instead of silently drifting out of sync: whichever box actually
		// needs more room gets it, then the other column stretches to
		// match so the two stay the same total height.
		if n := lipgloss.Height(rightMeasureStyle.Render(runningContent)); n > runningH {
			runningH = n
		}
		if n := lipgloss.Height(rightMeasureStyle.Render(detailsContent)); n > detailsH {
			detailsH = n
		}

		rightAsLeftH := runningH + detailsH + 2 // right stacks 2 boxes' borders vs left's 1
		switch {
		case rightAsLeftH > leftH:
			leftH = rightAsLeftH
		case leftH > rightAsLeftH:
			detailsH += leftH - rightAsLeftH
		}

		runningBox := runningBoxStyle.Width(rightW).Height(runningH).Render(runningContent)
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
	help := "←/→ switch  ↑/k up  ↓/j down  enter select/run  s stop  e logs  del delete (press twice)  q quit"
	if m.focus == focusSettingsContent && m.settings.dirs.editing {
		help = "enter save  esc cancel"
	}
	b.WriteString(helpStyle.Render(help))
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

// renderLeftPaneContent renders whichever tab's content is active. The
// outer Models/Recents/Settings tab bar lives outside the boxes, in
// renderHeaderLine.
func (m Model) renderLeftPaneContent(leftW int) string {
	switch m.leftMode {
	case modeRecents:
		return m.renderRecentsList()
	case modeSettings:
		return m.renderSettingsList()
	case modeRunning:
		return m.renderRunningTabList()
	default:
		return m.renderModelsTree()
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

	dashStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	if focused {
		dashStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	}
	return dashStyle.Render(strings.Repeat("─", left)) + content + dashStyle.Render(strings.Repeat("─", right))
}

// renderTabBarLabels shows the Models/Recents/Settings/Running tabs, with
// the active tab bold.
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

	rendered := make([]string, len(tabs))
	for i, t := range tabs {
		style := profileStyle
		if m.leftMode == t.mode {
			style = selectedProfileStyle
		}
		rendered[i] = style.Render("[ " + t.label + " ]")
	}

	return strings.Join(rendered, "  ")
}

// renderSettingsList shows the Settings tab's menu of configuration
// sub-pages (currently just Model Directories).
func (m Model) renderSettingsList() string {
	var b strings.Builder
	rows := buildSettingsRows()
	focused := m.focus == focusLeft
	for i, r := range rows {
		selected := i == m.settingsCursor
		cursor := "  "
		style := profileStyle
		if selected {
			if focused {
				cursor = cursorStyle.Render("> ")
				style = selectedProfileStyle
			} else {
				cursor = profileStyle.Render("> ")
			}
		}
		b.WriteString(fmt.Sprintf("%s%s\n", cursor, style.Render(r.label)))
	}
	return b.String()
}

func (m Model) renderModelsTree() string {
	var b strings.Builder

	if len(m.cfg.ModelsDirs) == 0 {
		b.WriteString(modelStyle.Render("No models folders set."))
		b.WriteString("\n")
		b.WriteString(profileStyle.Render("Please navigate to Settings > Model Directories"))
		b.WriteString("\n")
		b.WriteString(profileStyle.Render("to add a directory to load models from."))
		return b.String()
	}

	if len(m.rows) == 0 {
		b.WriteString(profileStyle.Render("(no models configured)"))
		b.WriteString("\n")
		b.WriteString(helpStyle.Render("press A to add a model from your configured folders"))
		return b.String()
	}

	focused := m.focus == focusLeft
	for i, r := range m.rows {
		selected := i == m.cursor
		cursor := "  "
		if selected {
			if focused {
				cursor = cursorStyle.Render("> ")
			} else {
				cursor = profileStyle.Render("> ")
			}
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
			style := profileStyle
			if selected {
				style = modelStyle
				if focused {
					style = selectedProfileStyle
				}
			}
			b.WriteString(fmt.Sprintf("%s%s\n", cursor, style.Render(r.label)))

		case rowProfile:
			label := r.label
			style := profileStyle
			switch {
			case r.modelKey == m.pendingDeleteModel && r.profileKey == m.pendingDeleteProfile:
				style = pendingDeleteStyle
				label += " (del again to confirm)"
			case selected && focused:
				style = selectedProfileStyle
			}
			b.WriteString(fmt.Sprintf("%s  %s\n", cursor, style.Render(label)))

		case rowAddProfile:
			style := addStyle
			if selected && focused {
				style = selectedAddStyle
			}
			b.WriteString(fmt.Sprintf("%s  %s\n", cursor, style.Render(r.label)))

		case rowAddModel:
			b.WriteString("\n")
			style := addStyle
			if selected && focused {
				style = selectedAddStyle
			}
			b.WriteString(fmt.Sprintf("%s%s\n", cursor, style.Render(r.label)))
		}
	}
	return b.String()
}

// renderRecentsList shows up to models.RecentLimit most recently run
// profiles, most recent first, for quick re-selection.
func (m Model) renderRecentsList() string {
	var b strings.Builder

	if len(m.recentRows) == 0 {
		b.WriteString(profileStyle.Render("(nothing run yet)"))
		return b.String()
	}

	focused := m.focus == focusLeft
	for i, r := range m.recentRows {
		selected := i == m.recentCursor
		cursor := "  "
		style := profileStyle
		if selected {
			if focused {
				cursor = cursorStyle.Render("> ")
				style = selectedProfileStyle
			} else {
				cursor = profileStyle.Render("> ")
			}
		}
		b.WriteString(fmt.Sprintf("%s%s\n", cursor, style.Render(r.label)))
	}
	return b.String()
}

func (m Model) renderRunning() string {
	var b strings.Builder
	b.WriteString(modelStyle.Render("Running"))
	if vram := m.renderVRAMHeader(); vram != "" {
		b.WriteString("  " + vram)
	}
	b.WriteString("\n\n")

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
	dot := loadingStyle.Render("●")
	switch m.health[r.ModelKey+"/"+r.ProfileKey] {
	case health.StatusUp:
		dot = runningStyle.Render("●")
	case health.StatusDown:
		dot = downStyle.Render("●")
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

	text := fmt.Sprintf("%-24s :%d", r.Label(), r.Port)
	if rate, ok := m.tokRates[r.ModelKey+"/"+r.ProfileKey]; ok {
		text += fmt.Sprintf("  %.1f tok/s", rate)
	}
	if mb, ok := m.gpuByPID[r.PID]; ok {
		text += fmt.Sprintf("  %.1fG", float64(mb)/1024)
	}
	return fmt.Sprintf("%s%s %s\n", cursor, dot, labelStyle.Render(text))
}

// renderRunningTabList shows every running instance in the Running tab's
// left pane — the same data as renderRunning's glance box, just styled as
// a selectable list like the other tabs' left-pane content.
func (m Model) renderRunningTabList() string {
	var b strings.Builder

	if len(m.running) == 0 {
		b.WriteString(profileStyle.Render("(nothing running)"))
		return b.String()
	}

	focused := m.focus == focusLeft
	for i, r := range m.running {
		b.WriteString(m.renderRunningRow(r, i == m.runningCursor, focused))
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
	innerH := leftH - 2
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

	// header + blank + tail + blank + help
	budget := innerH - 4
	if budget < 1 {
		budget = 1
	}

	fmt.Fprintf(&b, "%s\n\n", header)
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
	lines := strings.Split(raw, "\n")

	measure := lipgloss.NewStyle().Width(boxWidth).Padding(0, 1)
	for len(lines) > 0 {
		candidate := strings.Join(lines, "\n")
		if lipgloss.Height(measure.Render(candidate)) <= maxLines {
			return candidate
		}
		lines = lines[1:]
	}
	return ""
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

// splitPaneHeight divides the right column so its total rendered height
// (including each box's own border) matches leftHeight, the left pane's
// content height. Running entries are one line each, so that box is sized
// to just fit runningCount (with a small minimum/cap) and Details — which
// tends to have more to show — gets whatever height is left over.
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
	return running, details
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
		text := fmt.Sprintf("%-20s :%d", p.Name, p.Port)
		if p.Temp != nil {
			text += fmt.Sprintf("  temp %.2g", *p.Temp)
		}
		b.WriteString(profileStyle.Render("• "+text) + "\n")
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

// renderDetails shows the settings for the currently selected profile,
// including the backing model's on-disk file size and any notes. The
// header names whatever's actually focused instead of a static "Details"
// label, since that's more useful at a glance.
func (m Model) renderDetails() string {
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
		b.WriteString("\n\n")
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

	b.WriteString(modelStyle.Render(mdl.Name + " / " + p.Name))
	b.WriteString("\n\n")
	fmt.Fprintf(&b, "%s\n\n", profileStyle.Render(modelSourceLine(mdl)))

	dash := func(s string) string {
		if s == "" {
			return "-"
		}
		return s
	}
	pair := func(l1, v1, l2, v2 string) string {
		return fmt.Sprintf("%-24s%s: %s\n", l1+": "+v1, l2, v2)
	}

	var settings strings.Builder
	settings.WriteString(pair("Port", fmt.Sprint(p.Port), "Ctx Size", dash(intOrEmpty(p.CtxSize))))
	settings.WriteString(pair("Temp", dash(floatPtrOrEmpty(p.Temp)), "Top P", dash(floatPtrOrEmpty(p.TopP))))
	settings.WriteString(pair("Top K", dash(intPtrOrEmpty(p.TopK)), "Min P", dash(floatPtrOrEmpty(p.MinP))))
	settings.WriteString(pair("Presence Pen", dash(floatPtrOrEmpty(p.PresencePenalty)), "Repeat Pen", dash(floatPtrOrEmpty(p.RepetitionPenalty))))
	settings.WriteString(pair("Flash Attn", fmt.Sprint(p.FlashAttn), "GPU Layers", dash(intOrEmpty(p.GPULayers))))
	settings.WriteString(pair("Cache K", dash(p.CacheTypeK), "Cache V", dash(p.CacheTypeV)))
	fmt.Fprintf(&settings, "Extra Args: %s\n", dash(strings.Join(p.ExtraArgs, " ")))

	b.WriteString(profileStyle.Render(settings.String()))

	if p.Notes != "" {
		b.WriteString("\n")
		b.WriteString(modelStyle.Render("Notes:"))
		b.WriteString("\n")
		b.WriteString(profileStyle.Render(p.Notes))
		b.WriteString("\n")
	}

	return b.String()
}
