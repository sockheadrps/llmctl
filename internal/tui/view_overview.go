package tui

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/sockheadrps/llmctl/internal/build"
	"github.com/sockheadrps/llmctl/internal/health"
	"github.com/sockheadrps/llmctl/internal/models"
	"github.com/sockheadrps/llmctl/internal/statusserver"
)

// viewOverviewPage renders the complete Overview screen. The outer border's
// vertical lines appear only on the single spacer row between the top border
// and the inner boxes; below that the inner boxes sit flush, with nav text
// embedded in the left box's bottom border and the version in the right box's.
// The top border ╭╮ and spacer │ align with the inner box corners (col 1 and
// col totalW-4), matching the inner box edges rather than the terminal edges.
func (m Model) viewOverviewPage() string {
	totalW := m.width
	if totalW <= 0 {
		totalW = fallbackWidth
	}
	totalH := m.height
	if totalH <= 0 {
		totalH = fallbackHeight
	}

	// top border (1) + inner box (boxH+2) = totalH → boxH = totalH - 3
	// No spacer row: outer ╭╮ sits directly above inner box ╭╮ in the same
	// column so the corners appear as a continuous connected frame.
	boxH := totalH - 3
	if boxH < 4 {
		boxH = 4
	}

	// innerW is the available width for renderOverviewContent, which applies a
	// margin=1 on each side → boxes are totalW-4 wide, spanning col 1..totalW-4.
	innerW := totalW - 2

	topBorder := m.buildOverviewTopBorder(totalW)

	var sb strings.Builder
	sb.WriteString(topBorder)
	sb.WriteString("\n")
	// Inner boxes rendered directly below the top border — no spacer row.
	// The outer ╭╮ corners (col 1, col totalW-4) align with the inner box
	// corners, making the frame read as one connected structure.
	sb.WriteString(m.renderOverviewContent(innerW, boxH))
	return sb.String()
}

// buildOverviewTopBorder builds the top border line with the tab bar embedded
// near the left edge: " ╭─ <tabs> ──────╮   "
// The border is totalW-4 wide so that ╭ lands at col 1 and ╮ at col totalW-4,
// aligning with the inner box corners in renderOverviewContent.
func (m Model) buildOverviewTopBorder(totalW int) string {
	tabs := m.renderTabBarLabels()
	tabsW := lipgloss.Width(tabs)
	// Border spans col 1..totalW-4 → width = totalW-4; inner dash space = totalW-6.
	borderW := totalW - 4
	if borderW < 2 {
		borderW = 2
	}
	innerW := borderW - 2
	// 1 dash + space before tabs, space after, rest fills to ╮
	rightDash := innerW - 1 - 1 - tabsW - 1
	if rightDash < 0 {
		rightDash = 0
	}

	focused := m.focus == focusTabs
	dashColor := lipgloss.Color("38")
	if focused {
		dashColor = lipgloss.Color("39")
	}
	dashStyle := lipgloss.NewStyle().Foreground(dashColor)
	borderStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("38"))

	border := borderStyle.Render("╭") +
		dashStyle.Render("─") +
		" " + tabs + " " +
		dashStyle.Render(strings.Repeat("─", rightDash)) +
		borderStyle.Render("╮")
	// Pad to full terminal width: 1 space before, 3 spaces after.
	return " " + border + "   "
}

// renderOverviewContent builds the two-column body: ACTIVE SERVICES on the
// left, SYSTEM TELEMETRY on the right. Nav text is embedded in the left box's
// bottom border; version is centered in the right box's bottom border.
func (m Model) renderOverviewContent(innerW, boxH int) string {
	margin := 1
	available := innerW - margin*2

	leftBoxW := available * 3 / 5
	if leftBoxW < 34 {
		leftBoxW = 34
	}
	rightBoxW := available - leftBoxW
	if rightBoxW < 26 {
		rightBoxW = 26
		leftBoxW = available - rightBoxW
	}

	leftContentW := leftBoxW - 2
	rightContentW := rightBoxW - 2

	leftContent := m.renderActiveServices(leftContentW, boxH)
	rightContent := m.renderSystemTelemetry(rightContentW, boxH)

	// Build status text for the left box bottom border.
	var statusStr string
	switch {
	case m.starting:
		statusStr = "  " + loadingStyle.Render("starting "+m.startingLabel+"…")
	case m.stopping:
		statusStr = "  " + loadingStyle.Render("stopping "+m.stoppingLabel+"…")
	case m.netSwitching:
		statusStr = "  " + loadingStyle.Render("switching network…")
	case m.err != nil:
		msg := firstLine(m.err.Error())
		if m.errLogPath != "" {
			msg += "  (e for logs)"
		}
		statusStr = "  " + errorStyle.Render(msg)
	}

	// Progressive nav text: full → medium → minimal → hidden as left box narrows.
	leftBoxInnerW := leftBoxW - 2
	navFull := helpStyle.Render("click to copy addr  ·  ←→/ad tabs  ·  q quit") + statusStr
	navMid := helpStyle.Render("←→/ad tabs  ·  q quit")
	navMin := helpStyle.Render("q quit")
	var navText string
	switch {
	case leftBoxInnerW >= lipgloss.Width(navFull)+4:
		navText = navFull
	case leftBoxInnerW >= lipgloss.Width(navMid)+4:
		navText = navMid
	case leftBoxInnerW >= lipgloss.Width(navMin)+4:
		navText = navMin
	}

	// Progressive version text: full → short → hidden as right box narrows.
	rightBoxInnerW := rightBoxW - 2
	verFull := detailMutedStyle.Render("llmctl " + build.Version)
	verShort := detailMutedStyle.Render(build.Version)
	var versionText string
	switch {
	case rightBoxInnerW >= lipgloss.Width(verFull)+4:
		versionText = verFull
	case rightBoxInnerW >= lipgloss.Width(verShort)+4:
		versionText = verShort
	}

	leftBox := renderTitledInnerBox("ACTIVE SERVICES", leftContent, navText, false, leftBoxW, boxH)
	rightBox := renderTitledInnerBox("SYSTEM TELEMETRY", rightContent, versionText, true, rightBoxW, boxH)

	joined := lipgloss.JoinHorizontal(lipgloss.Top, leftBox, rightBox)
	joinedLines := strings.Split(joined, "\n")
	pad := strings.Repeat(" ", margin)
	for i, l := range joinedLines {
		joinedLines[i] = pad + l
	}
	return strings.Join(joinedLines, "\n")
}

// renderTitledInnerBox builds a rounded box with the title centered in the top
// border and bottomText embedded in the bottom border. When centerBottom is
// true the text is centered; otherwise it sits at the left edge after ╰─.
func renderTitledInnerBox(title, content, bottomText string, centerBottom bool, boxW, boxH int) string {
	borderColor := lipgloss.Color("38")
	bs := lipgloss.NewStyle().Foreground(borderColor)
	innerW := boxW - 2

	styledTitle := detailMutedStyle.Render(title)
	titleW := lipgloss.Width(styledTitle)
	remaining := innerW - titleW
	if remaining < 0 {
		remaining = 0
	}
	leftDash := remaining / 2
	rightDash := remaining - leftDash

	topBorder := bs.Render("╭") +
		bs.Render(strings.Repeat("─", leftDash)) +
		styledTitle +
		bs.Render(strings.Repeat("─", rightDash)) +
		bs.Render("╮")

	// Bottom border with embedded text — hide text independently if the box
	// is too narrow to fit it with at least a couple of dashes on each side.
	plainBottom := func() string {
		return bs.Render("╰") + bs.Render(strings.Repeat("─", innerW)) + bs.Render("╯")
	}
	var bottomBorder string
	if bottomText == "" {
		bottomBorder = plainBottom()
	} else {
		textW := lipgloss.Width(bottomText)
		dashTotal := innerW - textW - 2 // 1 space on each side of the text
		if dashTotal < 2 {
			// Not enough room for this box's label — fall back to plain dashes.
			bottomBorder = plainBottom()
		} else {
			var leftD, rightD int
			if centerBottom {
				leftD = dashTotal / 2
				rightD = dashTotal - leftD
			} else {
				leftD = 1
				rightD = dashTotal - 1
			}
			bottomBorder = bs.Render("╰") +
				bs.Render(strings.Repeat("─", leftD)) +
				" " + bottomText + " " +
				bs.Render(strings.Repeat("─", rightD)) +
				bs.Render("╯")
		}
	}

	rawLines := strings.Split(strings.TrimRight(content, "\n"), "\n")

	var sb strings.Builder
	sb.WriteString(topBorder)
	sb.WriteString("\n")
	for i := 0; i < boxH; i++ {
		var line string
		if i < len(rawLines) {
			line = rawLines[i]
		}
		pad := innerW - lipgloss.Width(line)
		if pad < 0 {
			pad = 0
		}
		sb.WriteString(bs.Render("│") + line + strings.Repeat(" ", pad) + bs.Render("│") + "\n")
	}
	sb.WriteString(bottomBorder)
	return sb.String()
}

// ─── ACTIVE SERVICES ─────────────────────────────────────────────────────────

func (m Model) renderActiveServices(contentW, contentH int) string {
	var b strings.Builder

	// Header — just the ALIAS label, no port/spd columns.
	b.WriteString(detailMutedStyle.Render("  ALIAS"))
	b.WriteString("\n")

	// ── Local ──────────────────────────────────────────────────────────────
	b.WriteString(sectionTitleStyle.Render("  Local"))
	b.WriteString("\n")
	if len(m.running) == 0 {
		b.WriteString(detailMutedStyle.Render("    (nothing running)"))
		b.WriteString("\n")
		b.WriteString(detailMutedStyle.Render("    → Models tab to start one"))
		b.WriteString("\n")
	} else {
		for _, r := range m.running {
			b.WriteString(m.renderServiceEntry(r, contentW))
		}
	}

	// ── Remote (server mode only, when clients have active models) ─────────
	if m.cfg.RPCEnabled && m.cfg.RPCMode == "server" && m.statusServer != nil {
		clients := m.statusServer.ClientStatuses(45 * time.Second)
		var activeClients []statusserver.ClientInfo
		for _, c := range clients {
			for _, ri := range c.Running {
				if ri.Health == string(health.StatusUp) || ri.Health == string(health.StatusLoading) {
					activeClients = append(activeClients, c)
					break
				}
			}
		}
		if len(activeClients) > 0 {
			b.WriteString("\n")
			b.WriteString(sectionTitleStyle.Render("  Remote"))
			b.WriteString("\n")
			for _, c := range activeClients {
				for _, ri := range c.Running {
					if ri.Health == string(health.StatusUp) || ri.Health == string(health.StatusLoading) {
						b.WriteString(m.renderRemoteServiceEntry(ri, contentW))
					}
				}
			}
		}
	}

	return b.String()
}

func (m Model) renderRemoteServiceEntry(ri statusserver.RunningInfo, contentW int) string {
	var b strings.Builder

	// Line 1: alias → profile key → model name + health dot.
	remoteName := ri.Alias
	if remoteName == "" {
		remoteName = ri.Profile
	}
	if remoteName == "" {
		remoteName = ri.Model
	}
	dot := loadingStyle.Render("●")
	switch ri.Health {
	case string(health.StatusUp):
		dot = runningStyle.Render("●")
	case string(health.StatusDown):
		dot = downStyle.Render("●")
	}
	b.WriteString(fmt.Sprintf("  %s %s\n", dot, modelStyle.Render(truncateText(remoteName, contentW-4))))

	narrow := contentW < 50

	// Line 2: └─ (size) [GPU|CPU] (—)  :port  (port omitted when narrow)
	modeBadge := detailMutedStyle.Render("[GPU]")
	if ri.RAMMiB > 0 {
		modeBadge = detailMutedStyle.Render("[CPU]")
	}
	detail := "  └─ "
	if ri.ModelSizeBytes > 0 {
		detail += fmt.Sprintf("(%.1fG) ", float64(ri.ModelSizeBytes)/(1024*1024*1024))
	}
	detail += modeBadge + detailMutedStyle.Render(" (—)")
	if !narrow {
		detail += profileStyle.Render(fmt.Sprintf("  :%d", ri.Port))
	}
	b.WriteString(detail + "\n")

	var cur, avg, peak string
	if ri.TokS > 0 {
		cur = fmt.Sprintf("%.0ft/s", ri.TokS)
	}
	if ri.TokAvg > 0 {
		avg = fmt.Sprintf("%.0f", ri.TokAvg)
	}
	if ri.TokPeak > 0 {
		peak = fmt.Sprintf("%.0f", ri.TokPeak)
	}

	if narrow {
		// Narrow: each stat on its own line.
		b.WriteString(detailMutedStyle.Render("     Current: "+cur) + "\n")
		b.WriteString(detailMutedStyle.Render("     Avg: "+avg) + "\n")
		b.WriteString(detailMutedStyle.Render("     Peak: "+peak+" T/S") + "\n")
	} else {
		spd := "Current: " + cur + " | Avg: " + avg + " | Peak " + peak + " T/S"
		b.WriteString(detailMutedStyle.Render("     "+spd) + "\n")
	}
	if spark := tokSparkline(ri.TokHistory); spark != "" {
		b.WriteString("   " + spark + "\n")
	}
	return b.String()
}

func (m Model) renderServiceEntry(r models.Running, contentW int) string {
	var b strings.Builder
	hkey := r.ModelKey + "/" + r.ProfileKey

	// alias: explicit Alias field → profile key → model name.
	displayName := r.ProfileName
	if displayName == "" {
		displayName = r.ModelName
	}
	if mdl, ok := m.cfg.Models[r.ModelKey]; ok {
		if p, ok := mdl.Profiles[r.ProfileKey]; ok && p.Alias != "" {
			displayName = p.Alias
		}
	}

	dot := loadingStyle.Render("●")
	switch m.health[hkey] {
	case health.StatusUp:
		dot = runningStyle.Render("●")
	case health.StatusDown:
		dot = downStyle.Render("●")
	}

	narrow := contentW < 50

	// Brief "✓ copied" flash — pad to the same height as a normal entry.
	hasSpark := len(m.tokHistory[hkey]) >= 2
	if m.overviewCopied == hkey {
		b.WriteString(fmt.Sprintf("  %s %s\n", dot, modelStyle.Render(truncateText(displayName, contentW-4))))
		b.WriteString(runningStyle.Render("  └─ ✓ copied to clipboard") + "\n")
		b.WriteString("\n")
		if narrow {
			b.WriteString("\n") // narrow has 5 lines total
			b.WriteString("\n")
		}
		if hasSpark {
			b.WriteString("\n") // match sparkline row
		}
		return b.String()
	}

	// Line 1: ● Alias
	b.WriteString(fmt.Sprintf("  %s %s\n", dot, modelStyle.Render(truncateText(displayName, contentW-4))))

	// Line 2: └─ (size) [GPU/CPU] (uptime)  :port  (port omitted when narrow)
	modeBadge := detailMutedStyle.Render("[GPU]")
	if mdl, ok := m.cfg.Models[r.ModelKey]; ok {
		if p, ok := mdl.Profiles[r.ProfileKey]; ok && p.CPUOnly {
			modeBadge = detailMutedStyle.Render("[CPU]")
		}
	}
	detail := "  └─ "
	if size := m.overviewModelSize(r.ModelKey); size != "" {
		detail += "(" + size + ") "
	}
	detail += modeBadge
	if r.StartedAt > 0 {
		detail += detailMutedStyle.Render(" (" + fmtUptime(time.Since(time.Unix(r.StartedAt, 0))) + ")")
	}
	if !narrow {
		detail += profileStyle.Render(fmt.Sprintf("  :%d", r.Port))
	}
	b.WriteString(detail + "\n")

	cur, avg, peak := m.overviewSpeeds(hkey, r.ModelKey, r.ProfileKey)
	if cur == "—" {
		cur = ""
	}

	spark := m.renderTokSparkline(hkey)

	if narrow {
		// Narrow: each stat on its own line.
		b.WriteString(detailMutedStyle.Render("     Current: "+cur) + "\n")
		b.WriteString(detailMutedStyle.Render("     Avg: "+avg) + "\n")
		b.WriteString(detailMutedStyle.Render("     Peak: "+peak+" T/S") + "\n")
	} else {
		spd := "Current: " + cur + " | Avg: " + avg + " | Peak " + peak + " T/S"
		b.WriteString(detailMutedStyle.Render("     "+spd) + "\n")
	}
	if spark != "" {
		b.WriteString("   " + spark + "\n")
	}
	return b.String()
}

// overviewSpeeds returns (current, avg, peak) display strings.
// current: live tok/s rate while generating, else "—"
// avg:     in-session rolling window average (tokHistory), falling back to
//          the persisted cross-session average (tokRateHistory) when idle
// peak:    all-time high from MaxTokPerSec / session tokPeak
func (m Model) overviewSpeeds(hkey, modelKey, profileKey string) (current, avg, peak string) {
	// All-time peak.
	var allTimePeak float64
	if mdl, ok := m.cfg.Models[modelKey]; ok {
		if p, ok := mdl.Profiles[profileKey]; ok {
			allTimePeak = p.MaxTokPerSec
		}
	}
	peakVal := allTimePeak
	if sp := m.tokPeak[hkey]; sp > peakVal {
		peakVal = sp
	}
	if peakVal > 0 {
		peak = fmt.Sprintf("%.0f", peakVal)
	}

	// Rolling average: in-session history first, then persisted cross-session.
	if hist := m.tokHistory[hkey]; len(hist) > 0 {
		var sum float64
		for _, v := range hist {
			sum += v
		}
		if inSess := sum / float64(len(hist)); inSess > 0 {
			avg = fmt.Sprintf("%.0f", inSess)
		}
	} else if histAvg, ok := m.tokRateHistory.average(hkey); ok && histAvg > 0 {
		avg = fmt.Sprintf("%.0f", histAvg)
	}

	// Current: live rate only while actively generating.
	if rate, ok := m.tokRates[hkey]; ok && rate > 0 {
		current = fmt.Sprintf("%.0ft/s", rate)
	} else {
		current = "—"
	}
	return
}

// overviewModelSize stats the model file and returns a human-readable size
// like "3.8G", or "" if the model is remote or the path isn't a plain file.
func (m Model) overviewModelSize(modelKey string) string {
	mdl, ok := m.cfg.Models[modelKey]
	if !ok || mdl.IsRemote() || mdl.Path == "" {
		return ""
	}
	info, err := os.Stat(mdl.Path)
	if err != nil || info.IsDir() {
		return ""
	}
	gb := float64(info.Size()) / (1024 * 1024 * 1024)
	return fmt.Sprintf("%.1fG", gb)
}

// ─── SYSTEM TELEMETRY ────────────────────────────────────────────────────────

func (m Model) renderSystemTelemetry(contentW, contentH int) string {
	var b strings.Builder

	// GPU 0: local GPU.
	// -1 for the margin space prepended to every line at the end of this function.
	const gpuPrefixW = 7 // len("GPU 0: ")
	b.WriteString(sectionTitleStyle.Render("GPU 0: "))
	if m.gpuAvailable && m.gpuName != "" {
		b.WriteString(profileStyle.Render(hScroll(m.gpuName, contentW-1-gpuPrefixW, m.gpuNameScroll)))
	} else if m.gpuAvailable {
		b.WriteString(detailMutedStyle.Render("(no data yet)"))
	} else {
		b.WriteString(detailMutedStyle.Render("N/A"))
	}
	b.WriteString("\n")
	if m.gpuAvailable && m.gpuUsage.TotalMiB > 0 {
		b.WriteString(m.overviewVRAMBar(m.gpuUsage.UsedMiB, m.gpuUsage.TotalMiB, contentW, true))
		b.WriteString("\n")
	}

	// RAM: total used by all CPU-only model processes on this instance.
	var totalRAMMiB int64
	for _, r := range m.running {
		if mdl, ok := m.cfg.Models[r.ModelKey]; ok {
			if p, ok := mdl.Profiles[r.ProfileKey]; ok && p.CPUOnly {
				totalRAMMiB += m.ramByPID[r.PID]
			}
		}
	}
	if totalRAMMiB > 0 {
		b.WriteString("RAM:  " + profileStyle.Render(fmt.Sprintf("%.1fG", float64(totalRAMMiB)/1024)) + "\n")
	}

	b.WriteString("\n")

	// GPU 1: remote GPU (client GPU in server mode; server GPU in client mode).
	if m.cfg.RPCEnabled {
		var remoteName string
		var remoteLabel string
		var remoteUsed, remoteTotal int64

		switch m.cfg.RPCMode {
		case "server":
			if m.statusServer != nil {
				if clients := m.statusServer.ClientStatuses(45 * time.Second); len(clients) > 0 {
					c := clients[0]
					if c.GPU != nil {
						remoteName = c.GPU.Name
						remoteUsed = c.GPU.UsedMiB
						remoteTotal = c.GPU.TotalMiB
						cname := c.ID
						if c.Name != "" {
							cname = c.Name
						}
						remoteLabel = "Client GPU (" + cname + ")"
					}
				}
			}
		case "client":
			if m.remoteStatus != nil && m.remoteStatus.GPU != nil {
				g := m.remoteStatus.GPU
				remoteName = g.Name
				remoteUsed = g.UsedMiB
				remoteTotal = g.TotalMiB
				remoteLabel = "Server GPU"
			}
		}

		if remoteName != "" {
			const gpu1PrefixW = 7 // len("GPU 1: ")
			b.WriteString(sectionTitleStyle.Render("GPU 1: ") + profileStyle.Render(hScroll(remoteName, contentW-1-gpu1PrefixW, m.gpuNameScroll)))
			b.WriteString("\n")
			const clientPrefixW = 2 // leading "  "
			b.WriteString(detailMutedStyle.Render("  " + hScroll(remoteLabel, contentW-1-clientPrefixW, m.gpuNameScroll)))
			b.WriteString("\n")
			if remoteTotal > 0 {
				b.WriteString(m.overviewVRAMBar(remoteUsed, remoteTotal, contentW, false))
				b.WriteString("\n")
			}
			b.WriteString("\n")
		}
	}

	// RPC BACKEND section.
	if m.cfg.RPCEnabled {
		b.WriteString(sectionTitleStyle.Render("RPC BACKEND"))
		b.WriteString("\n")

		switch m.cfg.RPCMode {
		case "server":
			rpcStatus := m.rpcServerHealthStatus()
			var statusLabel string
			switch rpcStatus {
			case health.StatusUp:
				statusLabel = runningStyle.Render("● ONLINE")
			case health.StatusDown:
				statusLabel = downStyle.Render("● OFFLINE")
			case health.StatusNotStarted:
				statusLabel = detailMutedStyle.Render("not started")
			default:
				statusLabel = loadingStyle.Render("loading…")
			}
			b.WriteString(detailMutedStyle.Render("Status:  ") + statusLabel + "\n")
			addr := fmt.Sprintf("%s:%d", m.cfg.RPCServerHost, m.cfg.RPCServerPort)
			b.WriteString(detailMutedStyle.Render("Address: ") + profileStyle.Render(addr) + "\n")
			if m.statusServer != nil {
				n := len(m.statusServer.ClientStatuses(45 * time.Second))
				switch n {
				case 0:
					b.WriteString(detailMutedStyle.Render("Clients: 0 connected"))
				case 1:
					b.WriteString(detailMutedStyle.Render("Clients: ") + profileStyle.Render("1 connected"))
				default:
					b.WriteString(detailMutedStyle.Render("Clients: ") + profileStyle.Render(fmt.Sprintf("%d connected", n)))
				}
				b.WriteString("\n")
			}

		case "client":
			var statusLabel string
			if m.discoveredRPCEndpoint != "" {
				statusLabel = runningStyle.Render("● CONNECTED")
			} else if m.remoteStatus != nil {
				statusLabel = runningStyle.Render("● REACHABLE")
			} else if m.cfg.RemoteStatusAddr != "" {
				statusLabel = loadingStyle.Render("● polling…")
			} else {
				statusLabel = detailMutedStyle.Render("not configured")
			}
			b.WriteString(detailMutedStyle.Render("Status:  ") + statusLabel + "\n")
			if m.discoveredRPCEndpoint != "" {
				b.WriteString(detailMutedStyle.Render("RPC:     ") + profileStyle.Render(m.discoveredRPCEndpoint) + "\n")
			} else if m.cfg.RPCEndpoint != "" {
				b.WriteString(detailMutedStyle.Render("RPC:     ") + detailMutedStyle.Render(m.cfg.RPCEndpoint) + "\n")
			}
			if m.cfg.RemoteStatusAddr != "" {
				b.WriteString(detailMutedStyle.Render("Server:  ") + detailMutedStyle.Render(m.cfg.RemoteStatusAddr) + "\n")
			}
		}
	}

	// Prepend one space to every non-blank line so content has the same left
	// margin as the Active Services box items.
	raw := b.String()
	rawLines := strings.Split(raw, "\n")
	for i, l := range rawLines {
		if l != "" {
			rawLines[i] = " " + l
		}
	}
	return strings.Join(rawLines, "\n")
}

// overviewVRAMBar renders "VRAM: [████░░░░░░] 3.3/12.0G".
// When the label doesn't fit on the same line it wraps below the bar.
// When localColor is true the fill uses traffic-light coloring.
func (m Model) overviewVRAMBar(usedMiB, totalMiB int64, contentW int, localColor bool) string {
	const barWidth = 10
	frac := float64(usedMiB) / float64(totalMiB)
	filled := int(frac * barWidth)
	if filled > barWidth {
		filled = barWidth
	}

	fillStyle := profileStyle
	if localColor {
		switch {
		case frac >= 0.9:
			fillStyle = downStyle
		case frac >= 0.7:
			fillStyle = loadingStyle
		default:
			fillStyle = runningStyle
		}
	}

	bar := detailMutedStyle.Render("[") +
		fillStyle.Render(strings.Repeat("█", filled)) +
		detailMutedStyle.Render(strings.Repeat("░", barWidth-filled)) +
		detailMutedStyle.Render("]")

	label := fmt.Sprintf("%.1f/%.1fG", float64(usedMiB)/1024, float64(totalMiB)/1024)
	// "VRAM: " (6) + "[" + 10 blocks + "]" (12) = 18 fixed chars; label adds len+1 for the space.
	const barFixedW = 18
	if barFixedW+1+len(label) <= contentW {
		return "VRAM: " + bar + profileStyle.Render(" "+label)
	}
	// Too narrow — put the label on the next line, indented under the bar.
	return "VRAM: " + bar + "\n" + strings.Repeat(" ", 6) + profileStyle.Render(label)
}

// hScroll returns a fixed-width window of name that ping-pongs left→right→left
// as tick increments. When name fits within availW it is returned unchanged.
func hScroll(name string, availW, tick int) string {
	runes := []rune(name)
	nameLen := len(runes)
	if nameLen <= availW || availW <= 0 {
		return name
	}
	overflow := nameLen - availW
	// Double the overflow to get a full round-trip cycle, then mirror the second half.
	cycle := overflow * 2
	pos := tick % cycle
	if pos > overflow {
		pos = cycle - pos
	}
	return string(runes[pos : pos+availW])
}

// fmtUptime formats a duration as "2h 14m", "45m", or "12s".
func fmtUptime(d time.Duration) string {
	s := int(d.Seconds())
	if s < 60 {
		return fmt.Sprintf("%ds", s)
	}
	if s < 3600 {
		return fmt.Sprintf("%dm", s/60)
	}
	h := s / 3600
	mins := (s % 3600) / 60
	if mins == 0 {
		return fmt.Sprintf("%dh", h)
	}
	return fmt.Sprintf("%dh %dm", h, mins)
}

// fmtAgo formats a short "X ago" string for last-seen timestamps.
func fmtAgo(d time.Duration) string {
	s := int(d.Seconds())
	if s < 5 {
		return "just now"
	}
	if s < 60 {
		return fmt.Sprintf("%ds ago", s)
	}
	if s < 3600 {
		return fmt.Sprintf("%dm ago", s/60)
	}
	return fmt.Sprintf("%dh ago", s/3600)
}
