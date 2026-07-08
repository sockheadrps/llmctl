package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/sockheadrps/llmctl/internal/gpu"
	"github.com/sockheadrps/llmctl/internal/health"
	"github.com/sockheadrps/llmctl/internal/models"
	"github.com/sockheadrps/llmctl/internal/runtime"
	"github.com/sockheadrps/llmctl/internal/statusserver"
)

// client-mode only
// pollRemoteStatusCmd polls the remote llmctl's status server at remoteStatusAddr
// ("host:port"). On success the caller derives the RPC endpoint from the
// response's rpc_server fields using the same host.
func pollRemoteStatusCmd(remoteStatusAddr string) tea.Cmd {
	return func() tea.Msg {
		st, err := statusserver.PollAddr(remoteStatusAddr)
		if err != nil {
			return remoteStatusMsg{err: err}
		}
		return remoteStatusMsg{status: &st}
	}
}

// shared
// backgroundChecks batches the periodic health/tok-rate/VRAM polls fired
// after a tick or a successful start.
func (m Model) backgroundChecks() tea.Cmd {
	cmds := []tea.Cmd{checkHealthCmd(m.running), checkSlotsCmd(m.running)}
	if m.networkTabVisible() {
		cmds = append(cmds, checkNetworkStatusCmd(m.netIface, m.netInternetConn, m.netRPCConn))
	}
	if m.gpuAvailable {
		cmds = append(cmds, checkVRAMCmd())
	}
	if m.cfg.RPCEnabled {
		switch m.cfg.RPCMode {
		case "server":
			cmds = append(cmds, checkRPCServerHealthCmd(m.mgr, m.cfg.RPCServerHost, m.cfg.RPCServerPort))
		case "client":
			if m.cfg.RemoteStatusAddr != "" {
				cmds = append(cmds, pollRemoteStatusCmd(m.cfg.RemoteStatusAddr))
			}
		}
	}
	return tea.Batch(cmds...)
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
		usage, err := gpu.Total()
		if err != nil {
			return vramMsg{}
		}
		byPID, err := gpu.ByPID()
		if err != nil {
			byPID = nil
		}
		return vramMsg{usage: usage, byPID: byPID}
	}
}

// server-mode only
// checkRPCServerHealthCmd checks the ggml-rpc-server health.
// PID is the primary signal: if the process is alive, it's up — a TCP probe
// would fail while the server is busy handling an existing RPC connection
// (e.g. a model loading on the remote machine). The TCP probe is only used
// as a fallback to detect an externally-started server with no state file.
func checkRPCServerHealthCmd(mgr *runtime.Manager, host string, port int) tea.Cmd {
	return func() tea.Msg {
		state, running := mgr.RPCServerStatus()
		if running {
			_ = state
			return healthMsg{"rpc-server": health.StatusUp}
		}
		// PID dead or no state file.
		if mgr.HasRPCStateFile() {
			return healthMsg{"rpc-server": health.StatusDown}
		}
		// No state file — check if something external is on the port.
		if health.ProbeRPCPort(host, port) {
			return healthMsg{"rpc-server": health.StatusUp}
		}
		return healthMsg{"rpc-server": health.StatusNotStarted}
	}
}
