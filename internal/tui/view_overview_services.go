package tui

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/sockheadrps/llmctl/internal/gpu"
	"github.com/sockheadrps/llmctl/internal/health"
	"github.com/sockheadrps/llmctl/internal/models"
	"github.com/sockheadrps/llmctl/internal/statusserver"
	tui_form "github.com/sockheadrps/llmctl/internal/tui/form"
	"github.com/sockheadrps/llmctl/internal/util"
)

// renderActiveServices renders the left column of the overview page.
func (m Model) renderActiveServices(contentW, contentH int) string {
	var b strings.Builder

	b.WriteString(detailMutedStyle.Render("ACTIVE SERVICES"))
	b.WriteString("\n")
	b.WriteString(detailMutedStyle.Render("  ALIAS"))
	b.WriteString("\n")

	b.WriteString(sectionTitleStyle.Render("  Local"))
	b.WriteString("\n")
	if len(m.running) == 0 {
		b.WriteString(detailMutedStyle.Render("    (nothing running)"))
		b.WriteString("\n")
	} else {
		var localStatus statusserver.Status
		if m.statusServer != nil {
			localStatus = m.statusServer.Status()
		}
		for _, r := range m.running {
			b.WriteString(m.renderServiceEntry(r, m.overviewRunningInfo(localStatus, r), m.overviewRemoteGPUDevices(), contentW))
		}
	}

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
						var remoteDevices []statusserver.GPUDeviceInfo
						if c.GPU != nil {
							remoteDevices = c.GPU.Devices
						}
						b.WriteString(m.renderRemoteServiceEntry(ri, remoteDevices, contentW))
					}
				}
			}
		}
	}

	return b.String()
}

// renderRemoteServiceEntry renders a single active-service entry for client mode.
func (m Model) renderRemoteServiceEntry(ri statusserver.RunningInfo, remoteDevices []statusserver.GPUDeviceInfo, contentW int) string {
	var b strings.Builder

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
	b.WriteString(fmt.Sprintf("  %s %s%s\n", dot, modelStyle.Render(tui_form.TruncateText(remoteName, contentW-4)), rpcBadge))

	narrow := contentW < 50

	modeBadge := detailMutedStyle.Render("[GPU]")
	if ri.RAMMiB > 0 {
		modeBadge = detailMutedStyle.Render("[CPU]")
	}
	detail := "  └─ "
	if size := m.overviewRunningSize(&ri, ""); size != "" {
		detail += "(" + size + ") "
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
	if gpuLines := m.renderRunningGPUBreakdown(m.combinedRemoteGPUSlices(ri), m.gpuDevices, remoteDevices, contentW); gpuLines != "" {
		b.WriteString(gpuLines)
	}
	return b.String()
}

// renderServiceEntry renders a single active-service entry for server mode.
func (m Model) renderServiceEntry(r models.Running, ri *statusserver.RunningInfo, remoteDevices []statusserver.GPUDeviceInfo, contentW int) string {
	var b strings.Builder
	hkey := r.ModelKey + "/" + r.ProfileKey

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

	hasSpark := len(m.tokHistory[hkey]) >= 2
	if m.overviewCopied == hkey {
		b.WriteString(fmt.Sprintf("  %s %s\n", dot, modelStyle.Render(tui_form.TruncateText(displayName, contentW-4))))
		b.WriteString(runningStyle.Render("  └─ ✓ copied to clipboard") + "\n")
		b.WriteString("\n")
		if narrow {
			b.WriteString("\n")
			b.WriteString("\n")
		}
		if hasSpark {
			b.WriteString("\n")
		}
		return b.String()
	}

	b.WriteString(fmt.Sprintf("  %s %s\n", dot, modelStyle.Render(tui_form.TruncateText(displayName, contentW-4))))

	modeBadge := detailMutedStyle.Render("[GPU]")
	if mdl, ok := m.cfg.Models[r.ModelKey]; ok {
		if p, ok := mdl.Profiles[r.ProfileKey]; ok && p.CPUOnly {
			modeBadge = detailMutedStyle.Render("[CPU]")
		}
	}
	detail := "  └─ "
	if size := m.overviewRunningSize(ri, r.ModelKey); size != "" {
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
	if ri != nil && len(ri.GPUs) > 0 {
		if gpuLines := m.renderRunningGPUBreakdown(m.combinedLocalGPUSlices(ri.GPUs), m.gpuDevices, remoteDevices, contentW); gpuLines != "" {
			b.WriteString(gpuLines)
		}
	}
	return b.String()
}

// overviewRunningInfo looks up the statusserver.RunningInfo for r in st.
func (m Model) overviewRunningInfo(st statusserver.Status, r models.Running) *statusserver.RunningInfo {
	for i := range st.Running {
		ri := &st.Running[i]
		if ri.Port == r.Port && ri.Model == r.ModelName && ri.Profile == r.ProfileName {
			return ri
		}
	}
	for i := range st.Running {
		ri := &st.Running[i]
		if ri.Port == r.Port {
			return ri
		}
	}
	return nil
}

// overviewRemoteGPUDevices returns GPU info for the client (server-mode) or server (client-mode).
func (m Model) overviewRemoteGPUDevices() []statusserver.GPUDeviceInfo {
	if m.remoteStatus != nil && m.remoteStatus.GPU != nil && len(m.remoteStatus.GPU.Devices) > 0 {
		return m.remoteStatus.GPU.Devices
	}
	return nil
}

// combinedRemoteGPUSlices wraps remote GPU info for the breakdown renderer.
func (m Model) combinedRemoteGPUSlices(ri statusserver.RunningInfo) []gpuLoadSlice {
	slices := make([]gpuLoadSlice, 0, len(ri.GPUs))
	for _, gpu := range ri.GPUs {
		slices = append(slices, gpuLoadSlice{label: gpu.Name, info: gpu})
	}
	return slices
}

// combinedLocalGPUSlices wraps local GPU info for the breakdown renderer.
func (m Model) combinedLocalGPUSlices(gpus []statusserver.GPUDeviceInfo) []gpuLoadSlice {
	slices := make([]gpuLoadSlice, 0, len(gpus))
	for _, gpu := range gpus {
		slices = append(slices, gpuLoadSlice{label: gpu.Name, info: gpu})
	}
	return slices
}

// renderRunningGPUBreakdown renders GPU load breakdown lines.
func (m Model) renderRunningGPUBreakdown(slices []gpuLoadSlice, localDevices []gpu.DeviceUsage, remoteDevices []statusserver.GPUDeviceInfo, contentW int) string {
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
		modelName := m.overviewGPULabelName(gpu.Name, localDevices, remoteDevices)
		label := slice.label
		if modelName != "" {
			label += " " + modelName
		}
		used := util.FormatBytes(gpu.UsedMiB * 1024 * 1024)
		line := fmt.Sprintf("     %s: %s", label, used)
		if totalModelLoad > 0 {
			totalText := util.FormatBytes(totalModelLoad * 1024 * 1024)
			pct := float64(gpu.UsedMiB) / float64(totalModelLoad) * 100
			line += fmt.Sprintf(" / %s (%.1f%%)", totalText, pct)
		}
		if lipgloss.Width(line) > contentW {
			line = fmt.Sprintf("     %s: %s", label, used)
			if totalModelLoad > 0 {
				line += fmt.Sprintf(" / %s", util.FormatBytes(totalModelLoad*1024*1024))
			}
		}
		b.WriteString(detailMutedStyle.Render(line))
		b.WriteString("\n")
	}
	return b.String()
}

// overviewGPULabelName resolves a human-readable GPU device name.
func (m Model) overviewGPULabelName(label string, localDevices []gpu.DeviceUsage, remoteDevices []statusserver.GPUDeviceInfo) string {
	label = strings.TrimSpace(label)
	if label == "" {
		return ""
	}
	switch {
	case strings.HasPrefix(strings.ToUpper(label), "RPC"):
		if name := overviewDeviceNameFromStatus(label, remoteDevices); name != "" {
			return name
		}
		return label
	case strings.HasPrefix(strings.ToUpper(label), "CUDA"):
		if name := overviewDeviceNameFromUsage(label, localDevices); name != "" {
			return name
		}
		return label
	default:
		return label
	}
}

// overviewDeviceNameFromUsage finds the device name for a CUDA index.
func overviewDeviceNameFromUsage(label string, devices []gpu.DeviceUsage) string {
	index, ok := overviewDeviceIndex(label)
	if !ok {
		return label
	}
	for _, device := range devices {
		if device.Index == index {
			if name := strings.TrimSpace(device.Name); name != "" {
				return name
			}
		}
	}
	return label
}

// overviewDeviceNameFromStatus finds the device name for an RPC index.
func overviewDeviceNameFromStatus(label string, devices []statusserver.GPUDeviceInfo) string {
	index, ok := overviewDeviceIndex(label)
	if !ok {
		return label
	}
	for _, device := range devices {
		if device.Index == index {
			if name := strings.TrimSpace(device.Name); name != "" {
				return name
			}
		}
	}
	return label
}

// overviewDeviceIndex extracts a device index from a label like "GPU 0" or "CUDA0".
func overviewDeviceIndex(label string) (int, bool) {
	label = strings.TrimSpace(strings.ToUpper(label))
	if strings.HasPrefix(label, "GPU ") {
		if n, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(label, "GPU "))); err == nil {
			return n, true
		}
	}
	var digits strings.Builder
	for i := len(label) - 1; i >= 0; i-- {
		ch := label[i]
		if ch < '0' || ch > '9' {
			break
		}
		digits.WriteByte(ch)
	}
	if digits.Len() == 0 {
		return 0, false
	}
	rev := []byte(digits.String())
	for i, j := 0, len(rev)-1; i < j; i, j = i+1, j-1 {
		rev[i], rev[j] = rev[j], rev[i]
	}
	n, err := strconv.Atoi(string(rev))
	if err != nil {
		return 0, false
	}
	return n, true
}

// overviewRunningSize returns the model size from ri or falls back to stat'ing the file.
func (m Model) overviewRunningSize(ri *statusserver.RunningInfo, modelKey string) string {
	if ri != nil {
		switch {
		case ri.VRAMMiB > 0:
			return util.FormatBytes(ri.VRAMMiB * 1024 * 1024)
		case ri.RAMMiB > 0:
			return util.FormatBytes(ri.RAMMiB * 1024 * 1024)
		case ri.ModelSizeBytes > 0:
			return util.FormatBytes(ri.ModelSizeBytes)
		}
	}
	return m.overviewModelSize(modelKey)
}

// overviewSpeeds returns (current, avg, peak) token/s rates.
func (m Model) overviewSpeeds(hkey, modelKey, profileKey string) (current, avg, peak string) {
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

	if rate, ok := m.tokRates[hkey]; ok && rate > 0 {
		current = fmt.Sprintf("%.0ft/s", rate)
	} else {
		current = "—"
	}
	return
}

// overviewModelSize stats the model file and returns a human-readable size.
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
