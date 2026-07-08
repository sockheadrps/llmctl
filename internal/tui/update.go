package tui

import (
	"fmt"
	"net"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/sockheadrps/llmctl/internal/health"
	"github.com/sockheadrps/llmctl/internal/runtime"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tickMsg:
		if m.mgr != nil {
			m.refreshRunning(true)
		}
		m.pushStatusServer()
		if m.screen == screenLogs {
			m.refreshLogs()
			return m, tickCmd()
		}
		if m.screen != screenMain {
			return m, tickCmd()
		}
		return m, tea.Batch(tickCmd(), m.backgroundChecks())

	case scrollTickMsg:
		switch m.screen {
		case screenNewProfile:
			m.form.advanceDescriptionScroll(m.formDescriptionLineCount(), m.formDescriptionVisibleLines())
		case screenMain:
			// Don't auto-scroll while the user is navigating settings content —
			// it fights cursor movement and scrolls options off-screen.
			if m.focus != focusSettingsContent {
				m.advanceDetailsScroll(m.mainDetailsLineCount(), m.mainDetailsVisibleLines())
			}
		}
		return m, scrollTickCmd()

	case healthMsg:
		for key, status := range msg {
			if m.pendingInstances[key] {
				if status == health.StatusUp {
					delete(m.pendingInstances, key)
					m.health[key] = status
					if startedAt, ok := m.loadStartedAt[key]; ok {
						dur := time.Since(startedAt)
						m.loadDuration[key] = dur
						m.loadHistory.record(key, dur.Seconds(), m.loadWithRPC[key])
						_ = saveLoadTimes(m.loadTimesPath, m.loadHistory)
						delete(m.loadStartedAt, key)
						delete(m.loadWithRPC, key)
					}
				}
				// While pending and not yet up (loading or down), leave health as StatusLoading (zero/default).
			} else {
				m.health[key] = status
			}
		}
		m.pushStatusServer()
		return m, nil

	case slotsMsg:
		m.applyTokSamples(msg)
		m.pushStatusServer()
		return m, nil

	case vramMsg:
		m.gpuUsage = msg.usage
		m.gpuByPID = msg.byPID
		m.pushStatusServer()
		return m, nil

	case remoteStatusMsg:
		if msg.err == nil {
			m.remoteStatus = msg.status
			// Derive the RPC endpoint from the status response: same host the
			// status server is on, but the ggml-rpc-server port from the payload.
			if msg.status.RPCServer != nil && msg.status.RPCServer.Up && msg.status.RPCServer.Port > 0 {
				host, _, err := net.SplitHostPort(m.cfg.RemoteStatusAddr)
				if err != nil {
					host = m.cfg.RemoteStatusAddr
				}
				m.discoveredRPCEndpoint = fmt.Sprintf("%s:%d", host, msg.status.RPCServer.Port)
			} else {
				m.discoveredRPCEndpoint = ""
			}
		} else {
			m.discoveredRPCEndpoint = ""
		}
		return m, nil

	case netStatusMsg:
		m.netStatus = msg
		return m, nil

	case netSwitchResultMsg:
		if msg.err != nil {
			m.netSwitching = false
			m.setError(msg.err, "")
			return m, nil
		}
		m.clearError()
		// Keep netSwitching=true while we verify the link came up.
		return m, verifyNetworkSwitchCmd(msg.toRPC, m.netInternetConn, m.netRPCConn)

	case netSwitchVerifyMsg:
		m.netSwitching = false
		// If nmcli couldn't be reached both flags are false — don't surface a
		// spurious error; the status panel will self-correct on the next poll.
		if msg.actualIsRPC || msg.actualIsInternet {
			if msg.toRPC && !msg.actualIsRPC {
				m.setError(fmt.Errorf("switch to RPC (%s) may have failed — connection not detected as active", m.netRPCConn), "")
			} else if !msg.toRPC && !msg.actualIsInternet {
				m.setError(fmt.Errorf("switch to internet (%s) may have failed — connection not detected as active", m.netInternetConn), "")
			}
		}
		return m, checkNetworkStatusCmd(m.netIface, m.netInternetConn, m.netRPCConn)

	case netConnectionsMsg:
		m.netPicker.loading = false
		m.netPicker.connections = msg.connections
		m.netPicker.cursor = 0
		return m, nil

	case startResultMsg:
		m.starting = false
		m.startingLabel = ""
		if msg.err != nil {
			m.setError(msg.err, msg.logPath)
			return m, nil
		}
		m.refreshRunning(false)
		m.pushStatusServer()
		m.rebuildRecentRows()
		m.clearError()
		key := msg.modelKey + "/" + msg.profileKey
		m.pendingInstances[key] = true
		m.loadStartedAt[key] = time.Now()
		m.loadWithRPC[key] = m.cfg.RPCEnabled
		return m, m.backgroundChecks()

	case stopResultMsg:
		m.stopping = false
		m.stoppingLabel = ""
		if msg.err != nil {
			m.setError(msg.err, "")
			return m, nil
		}
		m.refreshRunning(false)
		m.pushStatusServer()
		m.clearError()
		return m, nil

	case rpcServerStartMsg:
		m.starting = false
		m.startingLabel = ""
		if msg.err != nil {
			m.setError(msg.err, "")
			return m, nil
		}
		m.refreshRunning(false)
		m.pushStatusServer()
		m.clearError()
		return m, m.backgroundChecks()

	case rpcServerStopMsg:
		m.stopping = false
		m.stoppingLabel = ""
		if msg.err != nil {
			m.setError(msg.err, "")
			return m, nil
		}
		m.refreshRunning(false)
		m.pushStatusServer()
		m.clearError()
		return m, m.backgroundChecks()

	case rpcServerClearMsg:
		if msg.err != nil {
			m.setError(msg.err, "")
			return m, nil
		}
		m.rpcServerState = runtime.RPCServerState{}
		m.rpcServerAlive = false
		delete(m.health, "rpc-server")
		m.clearError()
		return m, m.backgroundChecks()

	case tea.MouseMsg:
		if m.screen == screenMain {
			return m.updateMouse(msg)
		}
		if m.screen == screenExportArgs {
			return m.updateExportArgs(msg)
		}
		return m, nil

	case tea.KeyMsg:
		switch m.screen {
		case screenPickModel:
			return m.updatePicker(msg)
		case screenNewProfile:
			return m.updateForm(msg)
		case screenFormExitConfirm:
			return m.updateFormExit(msg)
		case screenConfirmProfile:
			return m.updateConfirm(msg)
		case screenLogs:
			return m.updateLogs(msg)
		case screenRunningAction:
			return m.updateRunningAction(msg)
		case screenStopConfirm:
			return m.updateStopConfirm(msg)
		case screenProfileTemplate:
			return m.updateTemplatePicker(msg)
		case screenExportArgs:
			return m.updateExportArgs(msg)
		case screenNetworkSwitch:
			return m.updateNetworkSwitch(msg)
		case screenNetworkPicker:
			return m.updateNetworkPicker(msg)
		case screenRPCServerAction:
			return m.updateRPCServerAction(msg)
		default:
			return m.updateMain(msg)
		}
	}

	return m, nil
}

func (m Model) updateMain(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.searchEditing {
		return m.updateModelSearch(msg)
	}

	// While a settings row (e.g. a model directory) is mid add/edit, route
	// keys to its text input instead of the normal pane navigation — same
	// as the form/confirm screens, just inline in the Details pane instead
	// of a separate screen.
	if m.focus == focusSettingsContent && m.settings.dirs.editing {
		switch msg.String() {
		case "esc":
			m.settings.dirs.editing = false
			m.settings.dirs.err = ""
			return m, nil
		case "enter":
			return m.submitDirForm()
		}
		var cmd tea.Cmd
		m.settings.dirs.input, cmd = m.settings.dirs.input.Update(msg)
		return m, cmd
	}
	if m.focus == focusSettingsContent && m.settings.bin.editing {
		switch msg.String() {
		case "esc":
			m.settings.bin.editing = false
			m.settings.bin.err = ""
			return m, nil
		case "enter":
			return m.submitBinForm()
		}
		var cmd tea.Cmd
		m.settings.bin.input, cmd = m.settings.bin.input.Update(msg)
		return m, cmd
	}
	if m.focus == focusSettingsContent && m.settings.rpc.remoteAddrEditing {
		switch msg.String() {
		case "esc":
			m.settings.rpc.remoteAddrEditing = false
			m.settings.rpc.err = ""
			return m, nil
		case "enter":
			return m.submitRemoteStatusAddrForm()
		}
		var cmd tea.Cmd
		m.settings.rpc.remoteAddrInput, cmd = m.settings.rpc.remoteAddrInput.Update(msg)
		return m, cmd
	}
	if m.focus == focusSettingsContent && m.settings.rpc.editing {
		switch msg.String() {
		case "esc":
			m.settings.rpc.editing = false
			m.settings.rpc.err = ""
			return m, nil
		case "enter":
			return m.submitRPCEndpointForm()
		}
		var cmd tea.Cmd
		m.settings.rpc.input, cmd = m.settings.rpc.input.Update(msg)
		return m, cmd
	}
	if m.focus == focusSettingsContent && m.settings.rpc.rpcBinEditing {
		switch msg.String() {
		case "esc":
			m.settings.rpc.rpcBinEditing = false
			m.settings.rpc.err = ""
			return m, nil
		case "enter":
			return m.submitRPCServerBinForm()
		}
		var cmd tea.Cmd
		m.settings.rpc.rpcBinInput, cmd = m.settings.rpc.rpcBinInput.Update(msg)
		return m, cmd
	}
	if m.focus == focusSettingsContent && m.settings.rpc.portEditing {
		switch msg.String() {
		case "esc":
			m.settings.rpc.portEditing = false
			m.settings.rpc.err = ""
			return m, nil
		case "enter":
			return m.submitRPCServerPortForm()
		}
		var cmd tea.Cmd
		m.settings.rpc.portInput, cmd = m.settings.rpc.portInput.Update(msg)
		return m, cmd
	}
	if m.focus == focusSettingsContent && m.settings.statusSrv.hostEditing {
		switch msg.String() {
		case "esc":
			m.settings.statusSrv.hostEditing = false
			m.settings.statusSrv.err = ""
			return m, nil
		case "enter":
			return m.submitStatusServerHostForm()
		}
		var cmd tea.Cmd
		m.settings.statusSrv.hostInput, cmd = m.settings.statusSrv.hostInput.Update(msg)
		return m, cmd
	}
	if m.focus == focusSettingsContent && m.settings.statusSrv.portEditing {
		switch msg.String() {
		case "esc":
			m.settings.statusSrv.portEditing = false
			m.settings.statusSrv.err = ""
			return m, nil
		case "enter":
			return m.submitStatusServerPortForm()
		}
		var cmd tea.Cmd
		m.settings.statusSrv.portInput, cmd = m.settings.statusSrv.portInput.Update(msg)
		return m, cmd
	}

	// Any key other than a repeated Delete cancels a pending delete
	// confirmation, so it only fires when pressed twice in a row on the
	// same profile (or settings row).
	if !key.Matches(msg, keys.Delete) {
		m.pendingDeleteModel = ""
		m.pendingDeleteProfile = ""
		m.settings.dirs.pendingDel = ""
	}

	switch {
	case key.Matches(msg, keys.Quit):
		return m, tea.Quit

	case key.Matches(msg, keys.Left):
		m.resetDetailsScroll()
		return m.moveFocusLeft()

	case key.Matches(msg, keys.Right):
		m.resetDetailsScroll()
		return m.moveFocusRight()

	case key.Matches(msg, keys.Up):
		m.resetDetailsScroll()
		return m.moveCursor(-1)

	case key.Matches(msg, keys.Down):
		m.resetDetailsScroll()
		return m.moveCursor(1)

	case key.Matches(msg, keys.Run):
		return m.selectRow()

	case key.Matches(msg, keys.Copy):
		return m.copyOrDuplicateSelected()

	case key.Matches(msg, keys.Delete):
		return m.deleteSelected()

	case key.Matches(msg, keys.Logs):
		return m.openLogsForCurrent()
	}

	if msg.String() == "/" && m.screen == screenMain && m.leftMode == modeModels {
		m.searchEditing = true
		m.focus = focusLeft
		return m, nil
	}

	if msg.String() == "x" && m.focus == focusLeft {
		if r, ok := m.currentRow(); ok && r.kind == rowProfile {
			return m.openExportArgs(r)
		}
	}

	return m, nil
}

func (m Model) copyOrDuplicateSelected() (tea.Model, tea.Cmd) {
	if m.leftMode == modeNetwork {
		return m.copyNetworkFix()
	}
	if m.leftMode == modeRPCServer && m.cfg.RPCMode == "server" {
		return m.copyStatusServerAddr()
	}
	if m.leftMode == modeRunning || m.focus == focusRunning {
		return m.copySelectedEndpoint()
	}
	return m.duplicateSelectedProfile()
}

func (m Model) copySelectedEndpoint() (tea.Model, tea.Cmd) {
	r, ok := m.currentRow()
	if !ok || r.kind != rowRunning {
		return m, nil
	}
	run, ok := m.findRunning(r.modelKey, r.profileKey)
	if !ok {
		return m, nil
	}
	return m.copyEndpoint(run)
}

func (m Model) openNetworkPicker(role netPickerRole) (tea.Model, tea.Cmd) {
	m.netPicker = netPickerState{role: role, loading: true}
	m.screen = screenNetworkPicker
	return m, listNetworkConnectionsCmd(role)
}

func (m Model) updateNetworkPicker(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.screen = screenMain
	case "up", "w", "k":
		if m.netPicker.cursor > 0 {
			m.netPicker.cursor--
		}
	case "down", "s", "j":
		if m.netPicker.cursor < len(m.netPicker.connections)-1 {
			m.netPicker.cursor++
		}
	case "enter", " ":
		if !m.netPicker.loading && m.netPicker.cursor < len(m.netPicker.connections) {
			conn := m.netPicker.connections[m.netPicker.cursor]
			switch m.netPicker.role {
			case netPickerRoleInternet:
				m.netInternetConn = conn.name
				m.cfg.NetworkInternetConn = conn.name
			case netPickerRoleRPC:
				m.netRPCConn = conn.name
				m.cfg.NetworkRPCConn = conn.name
			}
			_ = m.saveConfig()
		}
		m.screen = screenMain
	}
	return m, nil
}

func (m Model) openNetworkSwitch() (tea.Model, tea.Cmd) {
	toRPC := m.netCursor == netRowSwitchRPC
	if toRPC && m.netStatus.isRPC {
		m.setError(fmt.Errorf("already connected via RPC (%s)", m.netRPCConn), "")
		return m, nil
	}
	if !toRPC && m.netStatus.isInternet {
		m.setError(fmt.Errorf("already connected via internet (%s)", m.netInternetConn), "")
		return m, nil
	}
	m.netSwitch = netSwitchState{toRPC: toRPC, cursor: 0}
	m.screen = screenNetworkSwitch
	return m, nil
}

func (m Model) updateNetworkSwitch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "left", "h", "a":
		m.netSwitch.cursor = 0
	case "right", "l", "d":
		m.netSwitch.cursor = 1
	case "esc":
		m.screen = screenMain
	case "enter", " ":
		m.screen = screenMain
		if m.netSwitch.cursor == 0 {
			m.netSwitching = true
			return m, switchNetworkCmd(m.netSwitch.toRPC, m.netInternetConn, m.netRPCConn)
		}
	}
	return m, nil
}

