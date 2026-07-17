package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/sockheadrps/llmctl/internal/health"
	"github.com/sockheadrps/llmctl/internal/statusserver"
)

// renderSystemTelemetry renders the right column of the overview page:
// GPU 0 (local), optional GPU 1 (remote), RAM usage, and RPC backend info.
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

// statusServerGPU extracts the GPU status from a statusserver.Status by label
// ("local" returns the server's own GPU, anything else looks at client/server
// peers via the caller's context). Kept as no-op here — the renderSystemTelemetry
// function walks statusserver structures directly for symmetry with services.
func init() {
	// Ensure statusserver import stays referenced.
	_ = &statusserver.Status{}
	_ = &lipgloss.Style{}
}

// hScroll returns the visible window of s for horizontal scrolling.
// tick cycles 0..len(s)*2 to smoothly scroll through the string when it exceeds
// the available content width.
func hScroll(s string, availW, tick int) string {
	runes := []rune(s)
	if len(runes) <= availW {
		return s
	}
	// Double the length so we scroll all the way right, then all the way left.
	cycle := len(runes) * 2
	pos := tick % cycle
	if pos >= len(runes) {
		pos = cycle - pos
	}
	end := pos + availW
	if end > len(runes) {
		end = len(runes)
	}
	return string(runes[pos:end])
}
