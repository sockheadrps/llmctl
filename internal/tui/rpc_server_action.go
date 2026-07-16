package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/sockheadrps/llmctl/internal/health"
	"github.com/sockheadrps/llmctl/internal/runtime"
)

// rpcServerStartMsg carries the outcome of an async RPC server start.
type rpcServerStartMsg struct {
	err error
}

// rpcServerStopMsg carries the outcome of an async RPC server stop.
type rpcServerStopMsg struct {
	err error
}

// rpcServerClearMsg carries the outcome of clearing stale RPC server state.
type rpcServerClearMsg struct {
	err error
}

func (m Model) startRPCServerCmd() tea.Cmd {
	mgr, cfg := m.mgr, m.cfg
	return func() tea.Msg {
		err := mgr.StartRPCServer(cfg)
		return rpcServerStartMsg{err: err}
	}
}

func (m Model) stopRPCServerCmd() tea.Cmd {
	mgr := m.mgr
	return func() tea.Msg {
		err := mgr.StopRPCServer()
		return rpcServerStopMsg{err: err}
	}
}

func (m Model) clearRPCServerCmd() tea.Cmd {
	mgr := m.mgr
	return func() tea.Msg {
		err := mgr.ClearRPCServer()
		return rpcServerClearMsg{err: err}
	}
}

// rpcServerActionChoice selects which button is highlighted.
// The button set shown depends on the current health status:
//
//	not started → [ Start ]  [ Cancel ]
//	up          → [ View Output ]  [ Stop ]  [ Cancel ]
//	down        → [ View Output ]  [ Clear ]  [ Cancel ]
type rpcServerActionChoice int

const (
	rpcServerActionPrimary   rpcServerActionChoice = iota // Start / View Output
	rpcServerActionSecondary                              // Cancel / Stop / Clear
	rpcServerActionCancel
)

type rpcServerActionState struct {
	selected rpcServerActionChoice
}

func (m Model) openRPCServerAction() (tea.Model, tea.Cmd) {
	m.rpcServerActionState = rpcServerActionState{selected: rpcServerActionPrimary}
	m.screen = screenRPCServerAction
	m.err = nil
	return m, nil
}

// rpcServerHealthStatus returns the current RPC server health, defaulting to
// StatusNotStarted when no check has run yet.
func (m Model) rpcServerHealthStatus() health.Status {
	if s := m.health["rpc-server"]; s != "" {
		return s
	}
	return health.StatusNotStarted
}

// maxRPCActionChoice returns the maximum choice index for the current state.
// Not-started has only 2 choices (Primary=Start, Secondary=Cancel);
// up/down have 3 (Primary=ViewOutput, Secondary=Stop/Clear, Tertiary=Cancel).
func (m Model) maxRPCActionChoice() rpcServerActionChoice {
	if m.rpcServerHealthStatus() == health.StatusNotStarted {
		return rpcServerActionSecondary // 2 buttons: Start, Cancel
	}
	return rpcServerActionCancel // 3 buttons: View Output, Stop/Clear, Cancel
}

func (m Model) updateRPCServerAction(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.screen = screenMain
		return m, nil

	case "left", "h", "a":
		if m.rpcServerActionState.selected > rpcServerActionPrimary {
			m.rpcServerActionState.selected--
		}
		return m, nil

	case "right", "l", "d":
		if m.rpcServerActionState.selected < m.maxRPCActionChoice() {
			m.rpcServerActionState.selected++
		}
		return m, nil

	case "enter", " ":
		m.screen = screenMain
		status := m.rpcServerHealthStatus()

		switch status {
		case health.StatusNotStarted:
			// 2-button layout: Primary=Start, Secondary=Cancel
			switch m.rpcServerActionState.selected {
			case rpcServerActionPrimary:
				m.starting = true
				m.startingLabel = "RPC server"
				m.clearError()
				return m, m.startRPCServerCmd()
			default:
				return m, nil
			}

		case health.StatusUp:
			// 3-button layout: Primary=ViewOutput, Secondary=Stop, Tertiary=Cancel
			switch m.rpcServerActionState.selected {
			case rpcServerActionPrimary:
				return m.openLogs(m.rpcServerState.LogFile, "RPC Server")
			case rpcServerActionSecondary:
				m.stopping = true
				m.stoppingLabel = "RPC server"
				m.clearError()
				return m, m.stopRPCServerCmd()
			default:
				return m, nil
			}

		default: // StatusDown
			// 3-button layout: Primary=ViewOutput, Secondary=Clear, Tertiary=Cancel
			switch m.rpcServerActionState.selected {
			case rpcServerActionPrimary:
				logPath := m.rpcServerState.LogFile
				if logPath == "" {
					var err error
					logPath, err = runtime.RPCServerLogPath()
					if err != nil {
						m.setError(err, "")
						return m, nil
					}
				}
				return m.openLogs(logPath, "RPC Server")
			case rpcServerActionSecondary:
				m.clearError()
				return m, m.clearRPCServerCmd()
			default:
				return m, nil
			}
		}
	}
	return m, nil
}
