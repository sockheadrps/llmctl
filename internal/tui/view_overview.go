package tui

import (
	"fmt"
	"os"
	"strings"
	"time"
	"unicode"

	"github.com/charmbracelet/lipgloss"

	"github.com/sockheadrps/llmctl/internal/build"
	"github.com/sockheadrps/llmctl/internal/health"
	"github.com/sockheadrps/llmctl/internal/models"
	"github.com/sockheadrps/llmctl/internal/statusserver"
	"github.com/sockheadrps/llmctl/internal/util"
)

type gpuLoadSlice struct {
	label string
	info  statusserver.GPUDeviceInfo
}

// viewOverviewPage renders the complete Overview screen as a single outer box
// (╭ at col 0, ╮ at col totalW-1) with a vertical │ separator between two
// columns: ACTIVE SERVICES on the left, SYSTEM TELEMETRY on the right.
// Using one box eliminates the "disconnected inner boxes" problem entirely.
func (m Model) viewOverviewPage() string {
	totalW := m.width
	if totalW <= 0 {
		totalW = fallbackWidth
	}
	totalH := m.height
	if totalH <= 0 {
		totalH = fallbackHeight
	}

	bs := lipgloss.NewStyle().Foreground(lipgloss.Color("38"))

	// top(1) + blank(1) + content(contentH) + bottom(1) = totalH
	contentH := totalH - 3
	if contentH < 2 {
		contentH = 2
	}

	// Row format: │ leftContent │ rightContent │
	// 1 + 1 + leftCW + 1 + 1 + rightCW + 1 + 1 = totalW → leftCW+rightCW = totalW-6
	leftCW, rightCW := m.overviewColumnWidths(totalW)

	// sepCol is the visual column of the │ separator.
	// Row: │(0) space(1) leftContent(leftCW, cols 2..leftCW+1) space(leftCW+2) │(leftCW+3) …
	sepCol := leftCW + 3

	leftLines := strings.Split(strings.TrimRight(m.renderActiveServices(leftCW, contentH), "\n"), "\n")
	rightLines := strings.Split(strings.TrimRight(m.renderSystemTelemetry(rightCW, contentH), "\n"), "\n")

	navText, versionText := m.overviewNavVersion(leftCW, rightCW)

	var sb strings.Builder
	sb.WriteString(m.buildOverviewTopBorder(totalW, sepCol))
	sb.WriteString("\n")
	// Blank row between tab bar and content — │ at separator column.
	sb.WriteString(bs.Render("│") + strings.Repeat(" ", sepCol-1) + bs.Render("│") + strings.Repeat(" ", totalW-2-sepCol) + bs.Render("│") + "\n")

	sep := bs.Render("│")
	for i := 0; i < contentH; i++ {
		l := ""
		if i < len(leftLines) {
			l = leftLines[i]
		}
		r := ""
		if i < len(rightLines) {
			r = rightLines[i]
		}
		lpad := leftCW - lipgloss.Width(l)
		if lpad < 0 {
			lpad = 0
		}
		rpad := rightCW - lipgloss.Width(r)
		if rpad < 0 {
			rpad = 0
		}
		sb.WriteString(bs.Render("│") + " " + l + strings.Repeat(" ", lpad) + " " + sep + r + strings.Repeat(" ", rpad) + " " + bs.Render("│") + "\n")
	}
	sb.WriteString(m.buildOverviewBottomBorder(totalW, navText, versionText, sepCol))
	return sb.String()
}

// buildOverviewBottomBorder builds ╰─ navText ─┴─ versionText ─╯ with a ┴ at
// sepCol where the column separator │ meets the bottom border.
// Left zone (cols 0..sepCol-1): ╰─ nav ─×n
// Right zone (cols sepCol+1..totalW-1): ─×m version ─╯
func (m Model) buildOverviewBottomBorder(totalW int, navText, versionText string, sepCol int) string {
	bs := lipgloss.NewStyle().Foreground(lipgloss.Color("38"))
	navW := lipgloss.Width(navText)
	verW := lipgloss.Width(versionText)

	// Left zone: ╰─ nav ─×leftDashes  (visual width = sepCol)
	// ╰(1) ─(1) sp(1) nav(navW) sp(1) ─×n = navW+4+n = sepCol → n = sepCol-navW-4
	var leftPart string
	if navText != "" {
		leftDashes := sepCol - navW - 4
		if leftDashes < 0 {
			leftDashes = 0
		}
		leftPart = bs.Render("╰") + bs.Render("─") + " " + navText + " " + bs.Render(strings.Repeat("─", leftDashes))
	} else {
		leftPart = bs.Render("╰") + bs.Render(strings.Repeat("─", sepCol-1))
	}

	// Right zone: ─×rightDashes version ─╯  (visual width = totalW-1-sepCol)
	// rightLen = totalW-1-sepCol chars for cols sepCol+1..totalW-1
	rightLen := totalW - 1 - sepCol
	var rightPart string
	if versionText != "" {
		// ─×r sp ver sp ─ ╯ → r + verW + 4 = rightLen → r = rightLen-verW-4
		rightDashes := rightLen - verW - 4
		if rightDashes < 0 {
			rightDashes = 0
		}
		rightPart = bs.Render(strings.Repeat("─", rightDashes)) + " " + versionText + " " + bs.Render("─") + bs.Render("╯")
	} else {
		rightPart = bs.Render(strings.Repeat("─", rightLen-1)) + bs.Render("╯")
	}

	return leftPart + bs.Render("┴") + rightPart
}

// overviewNavVersion returns the nav and version strings for the bottom border,
// sized to fit the column widths.
func (m Model) overviewNavVersion(leftCW, rightCW int) (navText, versionText string) {
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

	navFull := helpStyle.Render("click to copy addr  ·  ←→/ad tabs  ·  q quit") + statusStr
	navMid := helpStyle.Render("←→/ad tabs  ·  q quit")
	navMin := helpStyle.Render("q quit")
	switch {
	case leftCW >= lipgloss.Width(navFull)+2:
		navText = navFull
	case leftCW >= lipgloss.Width(navMid)+2:
		navText = navMid
	case leftCW >= lipgloss.Width(navMin)+2:
		navText = navMin
	}

	verFull := detailMutedStyle.Render("llmctl " + build.Version)
	verShort := detailMutedStyle.Render(build.Version)
	switch {
	case rightCW >= lipgloss.Width(verFull)+2:
		versionText = verFull
	case rightCW >= lipgloss.Width(verShort)+2:
		versionText = verShort
	}
	return
}

// buildOverviewTopBorder builds ╭─ tabs ─┬─╮ with a ┬ at sepCol where the
// column separator │ meets the top border.
func (m Model) buildOverviewTopBorder(totalW, sepCol int) string {
	tabs := m.renderTabBarLabels()
	tabsW := lipgloss.Width(tabs)

	focused := m.focus == focusTabs
	dashColor := lipgloss.Color("38")
	if focused {
		dashColor = lipgloss.Color("39")
	}
	dashStyle := lipgloss.NewStyle().Foreground(dashColor)
	bs := lipgloss.NewStyle().Foreground(lipgloss.Color("38"))

	// Fixed prefix: ╭─ tabs  (visual width = tabsW+4: ╭─ space tabs space)
	preW := tabsW + 4
	// Dashes between prefix and ┬, then dashes from ┬ to ╮.
	leftDashes := sepCol - preW
	if leftDashes < 0 {
		leftDashes = 0
	}
	rightDashes := totalW - 1 - sepCol - 1 // cols sepCol+1..totalW-2
	if rightDashes < 0 {
		rightDashes = 0
	}

	return bs.Render("╭") +
		dashStyle.Render("─") + " " + tabs + " " +
		dashStyle.Render(strings.Repeat("─", leftDashes)) +
		bs.Render("┬") +
		dashStyle.Render(strings.Repeat("─", rightDashes)) +
		bs.Render("╮")
}

// overviewColumnWidths returns (leftCW, rightCW) respecting the user-dragged
// separator position stored in overviewSepX.
func (m Model) overviewColumnWidths(totalW int) (leftCW, rightCW int) {
	const minLeft, minRight = 18, 14
	avail := totalW - 6
	if avail < minLeft+minRight {
		avail = minLeft + minRight
	}
	if m.overviewSepX > 0 {
		leftCW = m.overviewSepX - 3
	} else {
		leftCW = avail * 3 / 5
	}
	if leftCW < minLeft {
		leftCW = minLeft
	}
	if leftCW > avail-minRight {
		leftCW = avail - minRight
	}
	rightCW = avail - leftCW
	return
}

// ─── ACTIVE SERVICES ─────────────────────────────────────────────────────────

func (m Model) renderActiveServices(contentW, contentH int) string {
	var b strings.Builder

	b.WriteString(detailMutedStyle.Render("ACTIVE SERVICES"))
	b.WriteString("\n")
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
	rpcBadge := ""
	if len(ri.GPUs) > 0 || ri.VRAMMiB > 0 {
		rpcBadge = " " + detailMutedStyle.Render("[RPC]")
	}
	b.WriteString(fmt.Sprintf("  %s %s%s\n", dot, modelStyle.Render(truncateText(remoteName, contentW-4)), rpcBadge))

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
	if gpuLines := m.renderRunningGPUBreakdown(m.combinedRemoteGPUSlices(ri), contentW); gpuLines != "" {
		b.WriteString(gpuLines)
	}
	return b.String()
}

func (m Model) combinedRemoteGPUSlices(ri statusserver.RunningInfo) []gpuLoadSlice {
	slices := make([]gpuLoadSlice, 0, len(ri.GPUs))
	for _, gpu := range ri.GPUs {
		slices = append(slices, gpuLoadSlice{label: "Remote", info: gpu})
	}

	if m.cfg != nil && m.cfg.RPCMode == "server" && m.statusServer != nil {
		current := m.statusServer.Status()
		matchedLocal := false
		for _, local := range current.Running {
			if !sameRunningIdentity(local, ri) {
				continue
			}
			matchedLocal = true
			for _, gpu := range local.GPUs {
				slices = append(slices, gpuLoadSlice{label: "Local", info: gpu})
			}
		}
		if !matchedLocal && len(current.Running) == 1 && len(current.Running[0].GPUs) > 0 {
			for _, gpu := range current.Running[0].GPUs {
				slices = append(slices, gpuLoadSlice{label: "Local", info: gpu})
			}
		}
		for _, gpu := range current.RPCServer.GPUs {
			slices = append(slices, gpuLoadSlice{label: "Local", info: gpu})
		}
	}
	return slices
}

func sameRunningIdentity(a, b statusserver.RunningInfo) bool {
	modelA := normalizeIdentity(a.Model)
	modelB := normalizeIdentity(b.Model)
	profileA := normalizeIdentity(a.Profile)
	profileB := normalizeIdentity(b.Profile)
	aliasA := normalizeIdentity(a.Alias)
	aliasB := normalizeIdentity(b.Alias)

	switch {
	case modelA != "" && modelB != "" && modelA == modelB && profileA == profileB:
		return true
	case modelA != "" && modelB != "" && modelA == modelB:
		return true
	case aliasA != "" && aliasA == modelB:
		return true
	case aliasB != "" && aliasB == modelA:
		return true
	default:
		return false
	}
}

func normalizeIdentity(v string) string {
	v = strings.TrimSpace(strings.ToLower(v))
	if v == "" {
		return ""
	}
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.IsSpace(r) {
			return r
		}
		return -1
	}, v)
}

func (m Model) renderRunningGPUBreakdown(slices []gpuLoadSlice, contentW int) string {
	if len(slices) == 0 {
		return ""
	}
	totalModelLoad := int64(0)
	for _, slice := range slices {
		totalModelLoad += slice.info.UsedMiB
	}
	var b strings.Builder
	b.WriteString(detailMutedStyle.Render("     RPC GPU load") + "\n")
	for _, slice := range slices {
		gpu := slice.info
		label := gpu.Name
		if label == "" {
			label = gpu.UUID
		}
		if label == "" {
			if gpu.Index >= 0 {
				label = fmt.Sprintf("GPU %d", gpu.Index)
			} else {
				label = "GPU"
			}
		}
		label = slice.label + " " + label
		used := fmt.Sprintf("%.1fG", float64(gpu.UsedMiB)/1024)
		if gpu.UsedMiB < 1024 {
			used = fmt.Sprintf("%d MiB", gpu.UsedMiB)
		}
		line := fmt.Sprintf("     %s: %s loaded", label, used)
		if totalModelLoad > 0 {
			pct := float64(gpu.UsedMiB) / float64(totalModelLoad) * 100
			line += fmt.Sprintf(" of %s model", util.FormatBytes(totalModelLoad*1024*1024))
			line += fmt.Sprintf("  (%.1f%%)", pct)
		}
		if lipgloss.Width(line) > contentW {
			line = fmt.Sprintf("     %s: %s loaded", label, used)
		}
		b.WriteString(detailMutedStyle.Render(line))
		b.WriteString("\n")
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
//
//	the persisted cross-session average (tokRateHistory) when idle
//
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

	b.WriteString(detailMutedStyle.Render("SYSTEM TELEMETRY"))
	b.WriteString("\n")

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
