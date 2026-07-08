package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/sockheadrps/llmctl/internal/health"
	"github.com/sockheadrps/llmctl/internal/models"
	"github.com/sockheadrps/llmctl/internal/process"
	"github.com/sockheadrps/llmctl/internal/runtime"
	"github.com/sockheadrps/llmctl/internal/statusserver"
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
	case modeRecents:
		return m.renderRecentsList(leftW)
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

	if m.cfg.RPCEnabled && m.cfg.RPCMode != "client" {
		b.WriteString(sectionTitleStyle.Render("RPC Server"))
		b.WriteString("\n")
		rpcSt := m.rpcServerHealthStatus()
		switch rpcSt {
		case health.StatusUp:
			b.WriteString(runningStyle.Render("● up") + "  " + detailMutedStyle.Render(m.cfg.RPCServerHost+":"+strconv.Itoa(m.cfg.RPCServerPort)))
		case health.StatusDown:
			b.WriteString(downStyle.Render("● down"))
		case health.StatusNotStarted:
			b.WriteString(detailMutedStyle.Render("● not started"))
		default:
			b.WriteString(loadingStyle.Render("● loading"))
		}
		b.WriteString("\n")
		if m.statusServer != nil && rpcSt == health.StatusUp {
			clients := m.statusServer.ClientStatuses(45 * time.Second)
			n := len(clients)
			if n == 1 {
				b.WriteString(profileStyle.Render("1 client connected"))
			} else if n > 1 {
				b.WriteString(profileStyle.Render(strconv.Itoa(n) + " clients connected"))
			} else {
				b.WriteString(detailMutedStyle.Render("no clients connected"))
			}
			b.WriteString("\n")
			for _, client := range clients {
				b.WriteString(m.renderClientStatusLines(client))
			}
		}
		b.WriteString("\n")
	}

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
func (m Model) renderClientStatusLines(client statusserver.ClientInfo) string {
	if len(client.Running) == 0 {
		return ""
	}
	var b strings.Builder
	for _, ri := range client.Running {
		dot := loadingStyle.Render("●")
		switch ri.Health {
		case "up":
			dot = runningStyle.Render("●")
		case "down":
			dot = downStyle.Render("●")
		}
		meta := m.clientModelSizeMeta(ri)
		if ri.TokS > 0 {
			meta += fmt.Sprintf("  %.0f tok/s", ri.TokS)
		}
		fmt.Fprintf(&b, "  %s %s%s\n", dot, profileStyle.Render(ri.Model+" / "+ri.Profile), detailMutedStyle.Render(meta))
	}
	return b.String()
}

func (m Model) clientModelSizeMeta(ri statusserver.RunningInfo) string {
	if ri.ModelSizeBytes <= 0 {
		return ""
	}
	loaded := m.rpcServerLoadedVRAMMiB()
	if loaded <= 0 {
		return "  " + util.FormatBytes(ri.ModelSizeBytes)
	}
	return fmt.Sprintf("  %s / %s server GPU", util.FormatBytes(ri.ModelSizeBytes), util.FormatBytes(loaded*1024*1024))
}

func (m Model) rpcServerLoadedVRAMMiB() int64 {
	if !m.rpcServerAlive || m.rpcServerState.PID == 0 {
		return 0
	}
	return m.gpuByPID[m.rpcServerState.PID]
}
func (m Model) renderRunningRow(r models.Running, selected, focused bool) string {
	return m.renderRunningRowWithWidth(r, selected, focused, 0)
}

func fmtLoadDur(d time.Duration) string {
	s := int(d.Seconds())
	if s < 60 {
		return fmt.Sprintf("%ds", s)
	}
	return fmt.Sprintf("%dm%ds", s/60, s%60)
}

func (m Model) renderRunningRowWithWidth(r models.Running, selected, focused bool, width int) string {
	hkey := r.ModelKey + "/" + r.ProfileKey
	dot := loadingStyle.Render("●")
	badge := loadingStyle.Render("loading")
	switch m.health[hkey] {
	case health.StatusUp:
		dot = runningStyle.Render("●")
		badge = runningStyle.Render("up")
		if dur, ok := m.loadDuration[hkey]; ok {
			badge += detailMutedStyle.Render("  (" + fmtLoadDur(dur) + ")")
		}
	case health.StatusDown:
		dot = downStyle.Render("●")
		badge = downStyle.Render("down")
	default:
		if startedAt, ok := m.loadStartedAt[hkey]; ok {
			elapsed := fmtLoadDur(time.Since(startedAt))
			timing := elapsed
			if avg, ok := m.loadHistory.average(hkey, m.loadWithRPC[hkey]); ok {
				timing += detailMutedStyle.Render(" / avg " + fmtLoadDur(time.Duration(avg*float64(time.Second))))
			}
			badge = loadingStyle.Render("loading") + "  " + loadingStyle.Render(timing)
		}
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
	if rate, ok := m.tokRates[hkey]; ok {
		text += fmt.Sprintf("  %.1f tok/s", rate)
	}
	if mb, ok := m.gpuByPID[r.PID]; ok {
		text += fmt.Sprintf("  %.1fG", float64(mb)/1024)
	}
	if width > 0 {
		text = truncateText(text, max(1, formRowTextWidth(width)-lipgloss.Width(cursor)-2))
	}
	row := fmt.Sprintf("%s%s %s %s\n", cursor, dot, badge, labelStyle.Render(text))

	if rate, ok := m.tokRates[hkey]; ok {
		row += "   " + m.renderRateMeter(hkey, rate) + "\n"
	} else if peak := m.tokPeak[hkey]; peak > 0 {
		// model is idle but has a session history — show the meter at zero
		row += "   " + m.renderRateMeter(hkey, 0) + "\n"
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

// renderRPCServerTab renders the RPC tab's left pane. Branches on RPCMode:
// server mode shows ggml-rpc-server status + enter to start/stop;
// client mode shows connection health and discovered endpoint.
func (m Model) renderRPCServerTab() string {
	if m.cfg.RPCMode == "client" {
		return m.renderRPCConnectionTab()
	}
	return m.renderRPCServerModeTab()
}

func (m Model) renderRPCServerModeTab() string {
	var b strings.Builder

	focused := m.focus == focusLeft
	rpcStatus := m.rpcServerHealthStatus()
	endpoint := m.cfg.RPCServerHost + ":" + strconv.Itoa(m.cfg.RPCServerPort)
	rpcCursor := "  "
	rpcStyle := profileStyle
	if focused && m.rpcIPCursor == 0 {
		rpcCursor = cursorStyle.Render("> ")
		rpcStyle = selectedProfileStyle
	}

	switch rpcStatus {
	case health.StatusUp:
		b.WriteString(rpcCursor + rpcStyle.Render("RPC Server") + "  " + runningStyle.Render("up"))
		b.WriteString("\n")
		b.WriteString("  " + profileStyle.Render(endpoint))
	case health.StatusDown:
		b.WriteString(rpcCursor + rpcStyle.Render("RPC Server") + "  " + downStyle.Render("down"))
		b.WriteString("\n")
		b.WriteString("  " + detailMutedStyle.Render("process exited - enter to view logs or clear"))
	default:
		b.WriteString(rpcCursor + rpcStyle.Render("RPC Server") + "  " + detailMutedStyle.Render("not started"))
		b.WriteString("\n")
		b.WriteString("  " + profileStyle.Render(endpoint))
	}
	b.WriteString("\n")
	if m.cfg.RPCServerBin != "" {
		b.WriteString("  " + detailMutedStyle.Render("Binary: "+m.cfg.RPCServerBin))
		b.WriteString("\n")
	}

	if m.statusServer != nil {
		b.WriteString("\n")
		addrs := m.statusServerAddrs()
		if len(addrs) > 0 {
			b.WriteString(sectionTitleStyle.Render("Status Server"))
			b.WriteString("\n")
			selected := m.selectedStatusServerAddr()
			for i, a := range addrs {
				cursor := "  "
				style := profileStyle
				if focused && m.rpcIPCursor == i+1 {
					cursor = cursorStyle.Render("> ")
					style = selectedProfileStyle
				} else if a == selected {
					style = runningStyle
				}
				fmt.Fprintf(&b, "%s%s\n", cursor, style.Render(a))
			}
			if m.rpcAddrCopied {
				b.WriteString(runningStyle.Render("  copied"))
			} else {
				b.WriteString(detailMutedStyle.Render("  enter/c to copy and select"))
			}
			b.WriteString("\n")
		}
	}

	return b.String()
}

// renderRPCConnectionTab renders the left pane for client mode: connection
// health, discovered endpoint, and a summary of what's running remotely.
func (m Model) renderRPCConnectionTab() string {
	var b strings.Builder

	focused := m.focus == focusLeft
	cursor := "  "
	style := profileStyle
	if focused {
		cursor = cursorStyle.Render("> ")
		style = selectedProfileStyle
	}

	// Connection status row
	if m.discoveredRPCEndpoint != "" {
		b.WriteString(cursor + style.Render("RPC Connection") + "  " + runningStyle.Render("● connected"))
		b.WriteString("\n")
		b.WriteString("  " + profileStyle.Render(m.discoveredRPCEndpoint))
	} else if m.remoteStatus != nil {
		b.WriteString(cursor + style.Render("RPC Connection") + "  " + runningStyle.Render("● reachable"))
		b.WriteString("\n")
		b.WriteString("  " + detailMutedStyle.Render(m.cfg.RemoteStatusAddr))
	} else if m.cfg.RemoteStatusAddr != "" {
		b.WriteString(cursor + style.Render("RPC Connection") + "  " + loadingStyle.Render("● polling…"))
		b.WriteString("\n")
		b.WriteString("  " + detailMutedStyle.Render(m.cfg.RemoteStatusAddr))
	} else {
		b.WriteString(cursor + style.Render("RPC Connection") + "  " + detailMutedStyle.Render("not configured"))
		b.WriteString("\n")
		b.WriteString("  " + detailMutedStyle.Render("set Remote Status Address in Settings → RPC"))
	}
	b.WriteString("\n\n")

	if m.cfg.RPCEndpoint != "" {
		b.WriteString(detailMutedStyle.Render("Manual endpoint: " + m.cfg.RPCEndpoint))
		b.WriteString("\n\n")
	}

	// Remote running models
	if m.remoteStatus != nil && len(m.remoteStatus.Running) > 0 {
		b.WriteString(sectionTitleStyle.Render("Remote"))
		b.WriteString("\n")
		for _, ri := range m.remoteStatus.Running {
			label := ri.Model + " / " + ri.Profile
			var meta string
			if ri.TokS > 0 {
				meta = fmt.Sprintf("  %.0f tok/s", ri.TokS)
			}
			if ri.VRAMMiB > 0 {
				meta += fmt.Sprintf("  %.1fG", float64(ri.VRAMMiB)/1024)
			}
			b.WriteString("  " + runningStyle.Render("●") + " " + profileStyle.Render(label) + detailMutedStyle.Render(meta))
			b.WriteString("\n")
		}
	} else if m.remoteStatus != nil {
		b.WriteString(detailMutedStyle.Render("Remote: no models running"))
		b.WriteString("\n")
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
		b.WriteString(modelStyle.Render(m.tabTitle(m.leftMode)))
		b.WriteString("\n\n")
		b.WriteString(profileStyle.Render(m.tabInstructions(m.leftMode)))
		return b.String()
	}

	if m.leftMode == modeRPCServer {
		return m.renderRPCServerOutputPane(rightW, innerH)
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
		b.WriteString(m.renderRateMeter(key, rate))
		b.WriteString("\n")
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

func (m Model) renderRPCServerOutputPane(rightW, innerH int) string {
	if m.cfg.RPCMode == "server" {
		return m.renderRPCServerModeOutputPane(rightW, innerH)
	}
	return m.renderRPCConnectionOutputPane(innerH)
}

func (m Model) renderRPCServerModeOutputPane(rightW, innerH int) string {
	var b strings.Builder

	header := modelStyle.Render("RPC Server")
	endpoint := m.cfg.RPCServerHost + ":" + strconv.Itoa(m.cfg.RPCServerPort)
	rpcStatus := m.rpcServerHealthStatus()
	switch rpcStatus {
	case health.StatusUp:
		fmt.Fprintf(&b, "%s  %s  %s\n", header, runningStyle.Render("up"), endpoint)
	case health.StatusDown:
		fmt.Fprintf(&b, "%s  %s  %s\n", header, downStyle.Render("down"), endpoint)
	case health.StatusNotStarted:
		fmt.Fprintf(&b, "%s  %s\n", header, detailMutedStyle.Render("not started"))
	default:
		fmt.Fprintf(&b, "%s  %s  %s\n", header, loadingStyle.Render("loading"), endpoint)
	}

	if m.statusServer != nil {
		clients := m.statusServer.ClientStatuses(45 * time.Second)
		n := len(clients)
		if n == 1 {
			fmt.Fprintf(&b, "%s\n", profileStyle.Render("1 client connected"))
		} else if n > 1 {
			fmt.Fprintf(&b, "%s\n", profileStyle.Render(strconv.Itoa(n)+" clients connected"))
		} else {
			fmt.Fprintf(&b, "%s\n", detailMutedStyle.Render("no clients connected"))
		}
		for _, client := range clients {
			if lines := m.renderClientStatusLines(client); lines != "" {
				b.WriteString(lines)
			}
		}
	}
	b.WriteString("\n")

	if rpcStatus == health.StatusNotStarted {
		b.WriteString(detailMutedStyle.Render("(no output — server has not been started)"))
	} else {
		logPath := m.rpcServerState.LogFile
		if logPath == "" {
			if p, err := runtime.RPCServerLogPath(); err == nil {
				logPath = p
			}
		}
		if tail := tailFittingHeightRPC(logPath, rightW, innerH-5); tail != "" {
			b.WriteString(profileStyle.Render(tail))
		} else {
			b.WriteString(profileStyle.Render("(no output yet)"))
		}
	}
	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render("enter start/stop · e view full output"))
	return b.String()
}

func (m Model) renderRPCConnectionOutputPane(innerH int) string {
	var b strings.Builder

	header := modelStyle.Render("RPC Connection")
	if m.discoveredRPCEndpoint != "" {
		fmt.Fprintf(&b, "%s  %s\n\n", header, runningStyle.Render("● active"))
	} else if m.remoteStatus != nil {
		fmt.Fprintf(&b, "%s  %s\n\n", header, runningStyle.Render("● reachable"))
	} else if m.cfg.RemoteStatusAddr != "" {
		fmt.Fprintf(&b, "%s  %s\n\n", header, loadingStyle.Render("● polling…"))
	} else {
		fmt.Fprintf(&b, "%s\n\n", header)
	}

	// Status server connection
	if m.cfg.RemoteStatusAddr == "" {
		b.WriteString(detailMutedStyle.Render("No remote status address configured."))
		b.WriteString("\n")
		b.WriteString(detailMutedStyle.Render("Set one in Settings → RPC → Remote Status Address."))
		return b.String()
	}

	if m.remoteStatus != nil {
		fmt.Fprintf(&b, "%s%s  %s\n", profileStyle.Render("Status server:  "), runningStyle.Render("● reachable"), detailMutedStyle.Render(m.cfg.RemoteStatusAddr))
	} else {
		fmt.Fprintf(&b, "%s%s  %s\n", profileStyle.Render("Status server:  "), loadingStyle.Render("● polling…"), detailMutedStyle.Render(m.cfg.RemoteStatusAddr))
	}

	if m.discoveredRPCEndpoint != "" {
		fmt.Fprintf(&b, "%s%s\n", profileStyle.Render("RPC endpoint:   "), runningStyle.Render("● "+m.discoveredRPCEndpoint))
	} else if m.cfg.RPCEndpoint != "" {
		fmt.Fprintf(&b, "%s%s\n", profileStyle.Render("RPC endpoint:   "), detailMutedStyle.Render(m.cfg.RPCEndpoint+" (manual, not verified)"))
	} else {
		fmt.Fprintf(&b, "%s%s\n", profileStyle.Render("RPC endpoint:   "), detailMutedStyle.Render("not discovered yet"))
	}
	b.WriteString("\n")

	// Remote version and GPU
	if m.remoteStatus != nil {
		if m.remoteStatus.Version != "" {
			b.WriteString(detailMutedStyle.Render("Remote llmctl " + m.remoteStatus.Version))
			b.WriteString("\n")
		}
		if m.remoteStatus.GPU != nil {
			g := m.remoteStatus.GPU
			usedGB := float64(g.UsedMiB) / 1024
			totalGB := float64(g.TotalMiB) / 1024
			b.WriteString(profileStyle.Render(fmt.Sprintf("GPU: %s  %.1f/%.1fG", g.Name, usedGB, totalGB)))
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	// Remote running models
	if m.remoteStatus != nil && len(m.remoteStatus.Running) > 0 {
		b.WriteString(sectionTitleStyle.Render("Running on remote"))
		b.WriteString("\n")
		for _, ri := range m.remoteStatus.Running {
			label := ri.Model + " / " + ri.Profile
			var meta string
			if ri.TokS > 0 {
				meta = fmt.Sprintf("  %.0f tok/s", ri.TokS)
			}
			if ri.VRAMMiB > 0 {
				meta += fmt.Sprintf("  %.1fG", float64(ri.VRAMMiB)/1024)
			}
			fmt.Fprintf(&b, "  %s %s%s\n", runningStyle.Render("●"), profileStyle.Render(label), detailMutedStyle.Render(meta))
		}
	} else if m.remoteStatus != nil {
		b.WriteString(detailMutedStyle.Render("No models running on remote."))
		b.WriteString("\n")
	}

	_ = innerH
	return b.String()
}

// rpcLogNoisyPrefixes lists high-frequency ggml-rpc-server log patterns that
// clutter the tail preview. Consecutive runs are collapsed to a single
// summary line; the full log viewer (e) still shows everything.
var rpcLogNoisyPrefixes = []string{
	"Accepted client connection",
	"Client connection closed",
	"ggml_backend_cuda_graph_compute:",
}

func isNoisyRPCLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	for _, prefix := range rpcLogNoisyPrefixes {
		if strings.HasPrefix(trimmed, prefix) {
			return true
		}
	}
	return false
}

// compressRPCLogLines collapses consecutive runs of known-noisy lines into a
// single summary entry, preserving all other lines in order.
func compressRPCLogLines(lines []string) []string {
	var result []string
	noiseCount := 0

	flush := func() {
		if noiseCount == 0 {
			return
		}
		if noiseCount == 1 {
			// single occurrence — already appended, nothing extra to do
		} else {
			result = append(result, fmt.Sprintf("  [%d repeated lines — press e for full log]", noiseCount))
		}
		noiseCount = 0
	}

	for _, line := range lines {
		if isNoisyRPCLine(line) {
			if noiseCount == 0 {
				result = append(result, line) // keep the first as context
			}
			noiseCount++
		} else {
			flush()
			result = append(result, line)
		}
	}
	flush()
	return result
}

// tailFittingHeightRPC is like tailFittingHeight but compresses known-noisy
// RPC server log patterns before fitting to the available height.
func tailFittingHeightRPC(logPath string, boxWidth, maxLines int) string {
	raw, err := process.TailLog(logPath, 500)
	if err != nil || raw == "" {
		return ""
	}

	lines := wrappedLogPreviewLines(raw, boxWidth)
	if len(lines) == 0 {
		return ""
	}
	lines = compressRPCLogLines(lines)
	if len(lines) > maxLines {
		lines = lines[len(lines)-maxLines:]
	}
	return strings.Join(lines, "\n")
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
	total := float64(m.gpuUsage.TotalMiB)
	frac := float64(m.gpuUsage.UsedMiB) / total

	// RPC server VRAM segment shown at the start of the bar in amber.
	var rpcBlocks int
	if m.cfg.RPCEnabled && m.rpcServerAlive && m.rpcServerState.PID > 0 {
		if rpcMiB, ok := m.gpuByPID[m.rpcServerState.PID]; ok && rpcMiB > 0 {
			rpcBlocks = int(float64(rpcMiB) / total * barWidth)
			if rpcBlocks > barWidth {
				rpcBlocks = barWidth
			}
		}
	}

	filled := int(frac * barWidth)
	if filled > barWidth {
		filled = barWidth
	}
	llamaBlocks := filled - rpcBlocks
	if llamaBlocks < 0 {
		llamaBlocks = 0
	}
	emptyBlocks := barWidth - rpcBlocks - llamaBlocks

	llamaStyle := runningStyle
	switch {
	case frac >= 0.9:
		llamaStyle = downStyle
	case frac >= 0.7:
		llamaStyle = loadingStyle
	}

	bar := loadingStyle.Render(strings.Repeat("█", rpcBlocks)) +
		llamaStyle.Render(strings.Repeat("█", llamaBlocks)) +
		strings.Repeat("░", emptyBlocks)

	usedGB := float64(m.gpuUsage.UsedMiB) / 1024
	totalGB := total / 1024
	return fmt.Sprintf("%s %s", bar, profileStyle.Render(fmt.Sprintf("%.1f/%.1fG", usedGB, totalGB)))
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
		b.WriteString(detailMutedStyle.Render("• " + text))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("enter to expand and select a profile"))
	return b.String()
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

	if m.leftMode == modeNetwork {
		return m.renderNetworkDetails()
	}

	// Still at the outer tab bar — nothing's selected yet within a tab,
	// so explain what arrowing down into it will show instead of an empty
	// "(select a profile...)" placeholder that doesn't fit Recents/Settings.
	if m.focus == focusTabs {
		b.WriteString(modelStyle.Render(m.tabTitle(m.leftMode)))
		b.WriteString("\n\n")
		b.WriteString(profileStyle.Render(m.tabInstructions(m.leftMode)))
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
