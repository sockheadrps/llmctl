package tui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/sockheadrps/llmctl/internal/health"
	"github.com/sockheadrps/llmctl/internal/process"
	"github.com/sockheadrps/llmctl/internal/runtime"
	"github.com/sockheadrps/llmctl/internal/statusserver"
	"github.com/sockheadrps/llmctl/internal/util"
)

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

