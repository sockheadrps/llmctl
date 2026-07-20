package tui

import (
	"fmt"
	"os/exec"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/sockheadrps/llmctl/internal/health"
)

func (m Model) activateRPCRow() (tea.Model, tea.Cmd) {
	switch m.settings.rpc.cursor {
	case 0:
		wasEnabled := m.cfg.RPCEnabled
		wasServer := m.cfg.RPCMode == "server"
		m.cfg.RPCEnabled = !m.cfg.RPCEnabled
		if !m.cfg.RPCEnabled {
			// clear mode when disabling so next enable starts fresh
			m.cfg.RPCMode = ""
			m.settings.rpc.cursor = 0
		}
		if m.cfg.RPCEnabled && !wasEnabled {
			m.cfg.NetworkTabEnabled = true
		}
		var stopCmd tea.Cmd
		if wasEnabled && !m.cfg.RPCEnabled && wasServer && m.rpcServerAlive {
			stopCmd = m.stopRPCServerCmd()
		}
		if err := m.saveConfig(); err != nil {
			m.settings.rpc.err = err.Error()
		} else if err := m.reconcileStatusServer(); err != nil {
			m.settings.rpc.err = "status server: " + err.Error()
		} else {
			m.reconcileStatusPublisher()
		}
		return m, stopCmd
	case 1:
		// Select Client mode
		m.cfg.RPCMode = "client"
		if err := m.saveConfig(); err != nil {
			m.settings.rpc.err = err.Error()
		} else if err := m.reconcileStatusServer(); err != nil {
			m.settings.rpc.err = "status server: " + err.Error()
		} else {
			m.reconcileStatusPublisher()
		}
		return m, nil
	case 2:
		// Select Server mode
		m.cfg.RPCMode = "server"
		if err := m.saveConfig(); err != nil {
			m.settings.rpc.err = err.Error()
		} else if err := m.reconcileStatusServer(); err != nil {
			m.settings.rpc.err = "status server: " + err.Error()
		} else {
			m.reconcileStatusPublisher()
			if m.rpcServerHealthStatus() == health.StatusNotStarted {
				m.starting = true
				m.startingLabel = "RPC server"
				m.clearError()
				return m, m.startRPCServerCmd()
			}
		}
		return m, nil
	case 3:
		switch m.cfg.RPCMode {
		case "client":
			return m.openRemoteStatusAddrForm()
		case "server":
			return m.openStatusServerHostForm()
		}
		return m, nil
	case 4:
		switch m.cfg.RPCMode {
		case "client":
			return m.openRPCEndpointForm()
		case "server":
			return m.openStatusServerPortForm()
		}
		return m, nil
	case 5:
		if m.cfg.RPCMode == "server" {
			if runtime.GOOS == "windows" {
				return m.copyFirewallRule()
			}
			if m.netSupported && m.cfg.RPCEnabled {
				if !m.cfg.NetworkTabEnabled {
					if _, err := exec.LookPath("nmcli"); err != nil {
						m.settings.rpc.err = "nmcli not found - install NetworkManager to use the Network tab"
						return m, nil
					}
				}
				m.cfg.NetworkTabEnabled = !m.cfg.NetworkTabEnabled
				m.settings.rpc.err = ""
				if err := m.saveConfig(); err != nil {
					m.settings.rpc.err = err.Error()
				}
			}
		}
		return m, nil
	case 6:
		if m.cfg.RPCMode == "server" && runtime.GOOS == "windows" {
			return m.openRPCServerBinForm()
		}
		return m, nil
	case 7:
		if m.cfg.RPCMode == "server" && runtime.GOOS == "windows" {
			return m.openRPCServerPortForm()
		}
		return m, nil
	default:
		if m.cfg.RPCMode == "server" && m.settings.rpc.cursor == m.rpcServerEnvCursor() {
			return m.openRPCServerEnvForm()
		}
	}
	return m, nil
}

func (m Model) renderRPCEnvLabel() string {
	if len(m.cfg.RPCServerEnv) == 0 {
		return "RPC Server Env"
	}
	keys := make([]string, 0, len(m.cfg.RPCServerEnv))
	for k := range m.cfg.RPCServerEnv {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+"="+m.cfg.RPCServerEnv[k])
	}
	return "RPC Server Env (" + strings.Join(parts, " ") + ")"
}

// rpcServerEnvCursor returns the cursor index of the env row in server mode.
func (m Model) rpcServerEnvCursor() int {
	if runtime.GOOS == "windows" {
		return 8
	}
	if m.netSupported {
		return 6
	}
	return 5
}

func (m Model) openRemoteStatusAddrForm() (tea.Model, tea.Cmd) {
	ti := textinput.New()
	ti.Placeholder = "192.168.1.100:11435"
	ti.CharLimit = 128
	ti.Width = 40
	ti.SetValue(m.cfg.RemoteStatusAddr)
	ti.Focus()
	ti.CursorEnd()
	m.settings.rpc.remoteAddrInput = ti
	m.settings.rpc.remoteAddrEditing = true
	m.settings.rpc.err = ""
	return m, nil
}

func (m Model) submitRemoteStatusAddrForm() (tea.Model, tea.Cmd) {
	val := strings.TrimSpace(m.settings.rpc.remoteAddrInput.Value())
	m.cfg.RemoteStatusAddr = val
	if err := m.saveConfig(); err != nil {
		m.settings.rpc.err = err.Error()
		return m, nil
	}
	m.reconcileStatusPublisher()
	m.settings.rpc.remoteAddrEditing = false
	m.settings.rpc.err = ""
	return m, nil
}

func (m Model) openRPCEndpointForm() (tea.Model, tea.Cmd) {
	ti := textinput.New()
	ti.Placeholder = "192.168.1.100:50052"
	ti.CharLimit = 128
	ti.Width = 40
	ti.SetValue(m.cfg.RPCEndpoint)
	ti.Focus()
	ti.CursorEnd()
	m.settings.rpc.input = ti
	m.settings.rpc.editing = true
	m.settings.rpc.err = ""
	return m, nil
}

func (m Model) submitRPCEndpointForm() (tea.Model, tea.Cmd) {
	val := strings.TrimSpace(m.settings.rpc.input.Value())
	m.cfg.RPCEndpoint = val
	if err := m.saveConfig(); err != nil {
		m.settings.rpc.err = err.Error()
		return m, nil
	}
	m.settings.rpc.editing = false
	m.settings.rpc.err = ""
	return m, nil
}

func (m Model) openRPCServerBinForm() (tea.Model, tea.Cmd) {
	ti := textinput.New()
	ti.Placeholder = "ggml-rpc-server"
	ti.CharLimit = 512
	ti.Width = 50
	ti.SetValue(m.cfg.RPCServerBin)
	ti.Focus()
	ti.CursorEnd()
	m.settings.rpc.rpcBinInput = ti
	m.settings.rpc.rpcBinEditing = true
	m.settings.rpc.err = ""
	return m, nil
}

func (m Model) submitRPCServerBinForm() (tea.Model, tea.Cmd) {
	val := strings.TrimSpace(m.settings.rpc.rpcBinInput.Value())
	m.cfg.RPCServerBin = val
	if err := m.saveConfig(); err != nil {
		m.settings.rpc.err = err.Error()
		return m, nil
	}
	m.settings.rpc.rpcBinEditing = false
	m.settings.rpc.err = ""
	return m, nil
}

func (m Model) openRPCServerPortForm() (tea.Model, tea.Cmd) {
	ti := textinput.New()
	ti.Placeholder = "50052"
	ti.CharLimit = 5
	ti.Width = 40
	ti.SetValue(strconv.Itoa(m.cfg.RPCServerPort))
	ti.Focus()
	ti.CursorEnd()
	m.settings.rpc.portInput = ti
	m.settings.rpc.portEditing = true
	m.settings.rpc.err = ""
	return m, nil
}

func (m Model) submitRPCServerPortForm() (tea.Model, tea.Cmd) {
	raw := strings.TrimSpace(m.settings.rpc.portInput.Value())
	port, err := strconv.Atoi(raw)
	if err != nil || port <= 0 || port > 65535 {
		m.settings.rpc.err = "port must be a number between 1 and 65535"
		return m, nil
	}
	if m.isPortInUse(raw) {
		m.settings.rpc.err = fmt.Sprintf("port %d is already in use", port)
		return m, nil
	}
	m.cfg.RPCServerPort = port
	if err := m.saveConfig(); err != nil {
		m.settings.rpc.err = err.Error()
		return m, nil
	}
	m.settings.rpc.portEditing = false
	m.settings.rpc.err = ""
	return m, nil
}

func (m Model) openRPCServerEnvForm() (tea.Model, tea.Cmd) {
	ti := textinput.New()
	ti.Placeholder = "KEY=VALUE KEY2=VALUE2"
	ti.CharLimit = 512
	ti.Width = 50

	keys := make([]string, 0, len(m.cfg.RPCServerEnv))
	for k := range m.cfg.RPCServerEnv {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+"="+m.cfg.RPCServerEnv[k])
	}
	ti.SetValue(strings.Join(parts, " "))
	ti.Focus()
	ti.CursorEnd()
	m.settings.rpc.envInput = ti
	m.settings.rpc.envEditing = true
	m.settings.rpc.err = ""
	return m, nil
}

func (m Model) submitRPCServerEnvForm() (tea.Model, tea.Cmd) {
	raw := strings.TrimSpace(m.settings.rpc.envInput.Value())
	env := map[string]string{}
	if raw != "" {
		for _, pair := range strings.Fields(raw) {
			k, v, _ := strings.Cut(pair, "=")
			if k != "" {
				env[k] = v
			}
		}
	}
	if len(env) == 0 {
		env = nil
	}
	m.cfg.RPCServerEnv = env
	if err := m.saveConfig(); err != nil {
		m.settings.rpc.err = err.Error()
		return m, nil
	}
	m.settings.rpc.envEditing = false
	m.settings.rpc.err = ""
	return m, nil
}

func (m Model) copyFirewallRule() (tea.Model, tea.Cmd) {
	port := m.cfg.StatusServerPort
	if port == 0 {
		port = 11435
	}
	rule := fmt.Sprintf(
		`netsh advfirewall firewall add rule name="llmctl status server" dir=in action=allow protocol=TCP localport=%d profile=any`,
		port,
	)
	if err := writeClipboard(rule); err != nil {
		m.settings.statusSrv.err = "copy failed: " + err.Error()
		return m, nil
	}
	m.settings.statusSrv.copied = true
	m.settings.statusSrv.err = ""
	return m, nil
}
