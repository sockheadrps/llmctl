package tui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

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
