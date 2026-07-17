package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/sockheadrps/llmctl/internal/health"
	"github.com/sockheadrps/llmctl/internal/models"
	"github.com/sockheadrps/llmctl/internal/util"
)

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
	cpuOnly := false
	if mdl, ok := m.cfg.Models[r.ModelKey]; ok {
		if p, ok := mdl.Profiles[r.ProfileKey]; ok {
			cpuOnly = p.CPUOnly
		}
	}
	if !cpuOnly {
		if m.health[hkey] == health.StatusUp {
			if slices, err := m.modelLoadSlices(r.LogFile); err == nil && len(slices) > 0 {
				var total int64
				for _, slice := range slices {
					total += slice.UsedMiB
				}
				text += fmt.Sprintf("  %.1fG", float64(total)/1024)
			}
		}
	} else {
		if mb, ok := m.ramByPID[r.PID]; ok {
			text += fmt.Sprintf("  %.1fG RAM", float64(mb)/1024)
		}
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
	hasSpark := hasMeter && len(m.tokHistory[key]) >= 2

	// header + (meter) + (sparkline) + blank + tail + blank + help
	overhead := 4
	if hasMeter {
		overhead++
	}
	if hasSpark {
		overhead++
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
	if hasSpark {
		b.WriteString(m.renderTokSparkline(key))
		b.WriteString("\n")
	}
	b.WriteString("\n")
	if tail := tailFittingHeight(m.ctrl, run.LogFile, rightW, budget); tail != "" {
		b.WriteString(profileStyle.Render(tail))
	} else {
		b.WriteString(profileStyle.Render("(no output yet)"))
	}
	b.WriteString("\n\n")
	b.WriteString(help)
	return b.String()
}

// renderTokSparkline renders a compact Unicode bar chart of recent tok/s samples.
func (m Model) renderTokSparkline(key string) string {
	return tokSparkline(m.tokHistory[key])
}

// tokSparkline builds the sparkline string from a raw sample slice so it can
// be called with history received over the status protocol (remote entries).
func tokSparkline(hist []float64) string {
	if len(hist) < 2 {
		return ""
	}
	const maxWidth = 20
	const sparks = "▁▂▃▄▅▆▇█"
	n := len(hist)
	if n > maxWidth {
		n = maxWidth
	}
	samples := hist[len(hist)-n:]
	var peak float64
	for _, v := range samples {
		if v > peak {
			peak = v
		}
	}
	if peak <= 0 {
		return ""
	}
	runes := []rune(sparks)
	var sb strings.Builder
	for _, v := range samples {
		idx := int(v/peak*float64(len(runes)-1) + 0.5)
		if idx < 0 {
			idx = 0
		}
		if idx >= len(runes) {
			idx = len(runes) - 1
		}
		sb.WriteRune(runes[idx])
	}
	return infoStyle.Render(sb.String())
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
