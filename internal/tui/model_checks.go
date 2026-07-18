package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/sockheadrps/llmctl/internal/controller"
	"github.com/sockheadrps/llmctl/internal/gpu"
	"github.com/sockheadrps/llmctl/internal/health"
	"github.com/sockheadrps/llmctl/internal/models"
)

// client-mode only
// pollRemoteStatusCmd polls the remote llmctl's status server at remoteStatusAddr
// ("host:port"). On success the caller derives the RPC endpoint from the
// response's rpc_server fields using the same host.
func (m *Model) pollRemoteStatusCmd(remoteStatusAddr string) tea.Cmd {
	ctrl := m.ctrl
	return func() tea.Msg {
		status, err := ctrl.PollRemoteStatus(remoteStatusAddr)
		if err != nil || status == nil {
			return remoteStatusMsg{err: err}
		}
		return remoteStatusMsg{status: status}
	}
}

// shared
// checkRAMCmd reads RSS MiB for the given PIDs (CPU-only model processes).
func (m *Model) checkRAMCmd(pids []int) tea.Cmd {
	return func() tea.Msg {
		byPID := make(map[int]int64, len(pids))
		for _, pid := range pids {
			if mb := m.ctrl.GetRSSMiB(pid); mb > 0 {
				byPID[pid] = mb
			}
		}
		return ramMsg{byPID: byPID}
	}
}

// shared
// backgroundChecks batches the periodic health/tok-rate/VRAM polls fired
// after a tick or a successful start.
func (m Model) backgroundChecks() tea.Cmd {
	cmds := []tea.Cmd{
		timedCmd("checkHealth", checkHealthCmd(m.running)),
	}
	if slotsTargets := m.slotsPollTargets(); len(slotsTargets) > 0 {
		cmds = append(cmds, timedCmd("checkSlots", checkSlotsCmd(slotsTargets)))
	}
	if m.networkTabVisible() {
		cmds = append(cmds, timedCmd("checkNetworkStatus", checkNetworkStatusCmd(m.netIface, m.netInternetConn, m.netRPCConn)))
	}
	if m.gpuAvailable {
		cmds = append(cmds, timedCmd("checkVRAM", checkVRAMCmd()))
	}
	// Poll RSS for any CPU-only model processes.
	var cpuPIDs []int
	for _, r := range m.running {
		if mdl, ok := m.cfg.Models[r.ModelKey]; ok {
			if p, ok := mdl.Profiles[r.ProfileKey]; ok && p.CPUOnly {
				cpuPIDs = append(cpuPIDs, r.PID)
			}
		}
	}
	if len(cpuPIDs) > 0 {
		cmds = append(cmds, timedCmd("checkRAM", m.checkRAMCmd(cpuPIDs)))
	}
	if m.cfg.RPCEnabled {
		switch m.cfg.RPCMode {
		case "server":
			cmds = append(cmds, timedCmd("checkRPCServerHealth", checkRPCServerHealthCmd(m.ctrl, m.cfg.RPCServerHost, m.cfg.RPCServerPort)))
		case "client":
			if m.cfg.RemoteStatusAddr != "" {
				cmds = append(cmds, timedCmd("pollRemoteStatus", m.pollRemoteStatusCmd(m.cfg.RemoteStatusAddr)))
			}
		}
	}
	return tea.Batch(cmds...)
}

// backgroundPollInterval returns how often the periodic telemetry sweep
// should run. RPC client mode is intentionally a little slower because it
// has to juggle local GPU telemetry plus the remote status publisher.
func (m Model) backgroundPollInterval() time.Duration {
	if m.hasPendingInstances() {
		if m.cfg != nil && m.cfg.RPCEnabled && m.cfg.RPCMode == "client" && len(m.running) > 0 {
			return 8 * time.Second
		}
		return 4 * time.Second
	}
	if m.cfg != nil && m.cfg.RPCEnabled && m.cfg.RPCMode == "client" && len(m.running) > 0 {
		return 4 * time.Second
	}
	return 2 * time.Second
}

func (m Model) hasPendingInstances() bool {
	return len(m.pendingInstances) > 0
}

func (m Model) slotsPollTargets() []models.Running {
	if len(m.running) == 0 {
		return nil
	}
	targets := make([]models.Running, 0, len(m.running))
	for _, r := range m.running {
		key := r.ModelKey + "/" + r.ProfileKey
		if m.pendingInstances[key] {
			continue
		}
		if m.health[key] != health.StatusUp {
			continue
		}
		targets = append(targets, r)
	}
	return targets
}

// shared
func checkHealthCmd(running []models.Running) tea.Cmd {
	return func() tea.Msg {
		result := make(healthMsg, len(running))
		for _, r := range running {
			s := health.Check(r.Host, r.Port)
			if s == health.StatusUp && r.LogFile != "" && !health.LogReady(r.LogFile) {
				s = health.StatusLoading
			}
			result[r.ModelKey+"/"+r.ProfileKey] = s
		}
		return result
	}
}

// shared
// checkSlotsCmd polls /slots for each running instance and reports the
// cumulative decoded-token count for any slot currently generating. The
// rate itself is computed in Update, which has the previous sample to
// diff against.
func checkSlotsCmd(running []models.Running) tea.Cmd {
	return func() tea.Msg {
		result := make(slotsMsg, len(running))
		for _, r := range running {
			slots, err := health.Slots(r.Host, r.Port)
			if err != nil {
				continue
			}
			decoded := 0
			processing := false
			for _, s := range slots {
				if s.IsProcessing {
					processing = true
					decoded += s.Decoded()
				}
			}
			if processing {
				result[r.ModelKey+"/"+r.ProfileKey] = decoded
			}
		}
		return result
	}
}

// shared
// checkVRAMCmd polls nvidia-smi for aggregate and per-PID VRAM usage. Only
// call this when gpuAvailable — it shells out, so there's no point retrying
// every tick on a machine without nvidia-smi.
func checkVRAMCmd() tea.Cmd {
	return func() tea.Msg {
		devices, err := gpu.Devices()
		if err != nil {
			return vramMsg{}
		}
		var usage gpu.Usage
		for _, device := range devices {
			usage.UsedMiB += device.UsedMiB
			usage.TotalMiB += device.TotalMiB
		}
		return vramMsg{usage: usage, devices: devices}
	}
}

// server-mode only
// checkRPCServerHealthCmd checks the ggml-rpc-server health.
// PID is the primary signal: if the process is alive, it's up — a TCP probe
// would fail while the server is busy handling an existing RPC connection
// (e.g. a model loading on the remote machine). The TCP probe is only used
// as a fallback to detect an externally-started server with no state file.
func checkRPCServerHealthCmd(ctrl *controller.Controller, host string, port int) tea.Cmd {
	return func() tea.Msg {
		state, running := ctrl.RPCServerStatus()
		if running {
			_ = state
			return healthMsg{"rpc-server": health.StatusUp}
		}
		// PID dead or no state file.
		if ctrl.HasRPCStateFile() {
			return healthMsg{"rpc-server": health.StatusDown}
		}
		// No state file — check if something external is on the port.
		if health.ProbeRPCPort(host, port) {
			return healthMsg{"rpc-server": health.StatusUp}
		}
		return healthMsg{"rpc-server": health.StatusNotStarted}
	}
}
