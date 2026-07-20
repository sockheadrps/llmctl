package tui

import (
	"fmt"
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/sockheadrps/llmctl/internal/util"
)

// enterSettingsCategory moves focus into the selected category's content in
// the Details pane — the caller (selectRow, on Enter) already picked the
// category, so there's nothing more to navigate before showing it. State is
// loaded fresh from config so edits made elsewhere (or a previous visit)
// aren't stale.
func (m Model) enterSettingsCategory(categoryID string) (tea.Model, tea.Cmd) {
	switch categoryID {
	case "model_dirs":
		m.settings.activeCategory = categoryID
		m.settings.dirs = dirsContentState{list: append([]string(nil), m.cfg.ModelsDirs...)}
	case "llama_bin":
		m.settings.activeCategory = categoryID
		m.settings.bin = binContentState{}
	case "rpc":
		m.settings.activeCategory = categoryID
		m.settings.rpc = rpcContentState{}
	case "status_server":
		m.settings.activeCategory = categoryID
		m.settings.statusSrv = statusServerContentState{}
	}
	m.focus = focusSettingsContent
	m.clearError()
	return m, nil
}

func (m Model) activateSettingsContentRow() (tea.Model, tea.Cmd) {
	switch m.settings.activeCategory {
	case "llama_bin":
		return m.openBinForm()
	case "rpc":
		return m.activateRPCRow()
	case "status_server":
		return m.activateStatusServerRow()
	default:
		return m.activateDirsRow()
	}
}

func (m Model) settingsContentMoveCursor(delta int) (tea.Model, tea.Cmd) {
	switch m.settings.activeCategory {
	case "llama_bin":
		if delta < 0 {
			m.focus = focusLeft
		}
		return m, nil
	case "rpc":
		maxRPCCursor := 0
		if m.cfg.RPCEnabled {
			maxRPCCursor = 2 // mode selector rows always visible when enabled
			switch m.cfg.RPCMode {
			case "client":
				maxRPCCursor = 4
			case "server":
				if runtime.GOOS == "windows" {
					maxRPCCursor = 8
				} else if m.netSupported {
					maxRPCCursor = 6
				} else {
					maxRPCCursor = 5
				}
			}
		}
		next := m.settings.rpc.cursor + delta
		switch {
		case next < 0:
			m.focus = focusLeft
		case next <= maxRPCCursor:
			m.settings.rpc.cursor = next
		}
		return m, nil
	case "status_server":
		maxStatusSrvCursor := 4
		if runtime.GOOS == "windows" && m.cfg.StatusServerEnabled {
			maxStatusSrvCursor = 5
		}
		next := m.settings.statusSrv.cursor + delta
		switch {
		case next < 0:
			m.focus = focusLeft
		case next <= maxStatusSrvCursor:
			m.settings.statusSrv.cursor = next
		}
		return m, nil
	default:
		next := m.settings.dirs.cursor + delta
		switch {
		case next < 0:
			m.focus = focusLeft
		case next <= len(m.settings.dirs.list):
			m.settings.dirs.cursor = next
		}
		return m, nil
	}
}

func (m Model) activateStatusServerRow() (tea.Model, tea.Cmd) {
	switch m.settings.statusSrv.cursor {
	case 0:
		m.cfg.StatusServerEnabled = !m.cfg.StatusServerEnabled
		if err := m.saveConfig(); err != nil {
			m.settings.statusSrv.err = err.Error()
		} else if err := m.reconcileStatusServer(); err != nil {
			m.settings.statusSrv.err = err.Error()
		}
	case 1:
		return m.openStatusServerHostForm()
	case 2:
		return m.openStatusServerPortForm()
	case 3:
		m.cfg.SetStatusHistoryPersistEnabled(!m.cfg.StatusHistoryPersistEnabled())
		if err := m.saveConfig(); err != nil {
			m.settings.statusSrv.err = err.Error()
		} else if err := m.reconcileStatusServer(); err != nil {
			m.settings.statusSrv.err = err.Error()
		}
	case 4:
		m.cfg.SetStatusDashboardEnabled(!m.cfg.StatusDashboardEnabled())
		if err := m.saveConfig(); err != nil {
			m.settings.statusSrv.err = err.Error()
		} else if err := m.reconcileStatusServer(); err != nil {
			m.settings.statusSrv.err = err.Error()
		}
	case 5:
		if runtime.GOOS == "windows" {
			return m.copyFirewallRule()
		}
	}
	return m, nil
}

func (m Model) openStatusServerHostForm() (tea.Model, tea.Cmd) {
	ti := textinput.New()
	ti.Placeholder = "0.0.0.0"
	ti.CharLimit = 64
	ti.Width = 40
	host := m.cfg.StatusServerHost
	if host == "" {
		host = "0.0.0.0"
	}
	ti.SetValue(host)
	ti.Focus()
	ti.CursorEnd()
	m.settings.statusSrv.hostInput = ti
	m.settings.statusSrv.hostEditing = true
	m.settings.statusSrv.err = ""
	return m, nil
}

func (m Model) submitStatusServerHostForm() (tea.Model, tea.Cmd) {
	val := strings.TrimSpace(m.settings.statusSrv.hostInput.Value())
	if val == "" {
		val = "0.0.0.0"
	}
	m.cfg.StatusServerHost = val
	if err := m.saveConfig(); err != nil {
		m.settings.statusSrv.err = err.Error()
		return m, nil
	}
	if err := m.reconcileStatusServer(); err != nil {
		m.settings.statusSrv.err = err.Error()
		return m, nil
	}
	m.pushStatusServer()
	m.settings.statusSrv.hostEditing = false
	m.settings.statusSrv.err = ""
	return m, nil
}

func (m Model) openStatusServerPortForm() (tea.Model, tea.Cmd) {
	ti := textinput.New()
	ti.Placeholder = "11435"
	ti.CharLimit = 5
	ti.Width = 40
	ti.SetValue(strconv.Itoa(m.cfg.StatusServerPort))
	ti.Focus()
	ti.CursorEnd()
	m.settings.statusSrv.portInput = ti
	m.settings.statusSrv.portEditing = true
	m.settings.statusSrv.err = ""
	return m, nil
}

func (m Model) submitStatusServerPortForm() (tea.Model, tea.Cmd) {
	raw := strings.TrimSpace(m.settings.statusSrv.portInput.Value())
	port, err := strconv.Atoi(raw)
	if err != nil || port <= 0 || port > 65535 {
		m.settings.statusSrv.err = "port must be a number between 1 and 65535"
		return m, nil
	}
	m.cfg.StatusServerPort = port
	if err := m.saveConfig(); err != nil {
		m.settings.statusSrv.err = err.Error()
		return m, nil
	}
	if err := m.reconcileStatusServer(); err != nil {
		m.settings.statusSrv.err = err.Error()
		return m, nil
	}
	m.pushStatusServer()
	m.settings.statusSrv.portEditing = false
	m.settings.statusSrv.err = ""
	return m, nil
}

func (m Model) isPortInUse(portStr string) bool {
	addr, err := net.ResolveTCPAddr("tcp", net.JoinHostPort("127.0.0.1", portStr))
	if err != nil {
		return false
	}
	ln, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return true
	}
	ln.Close()
	return false
}

func (m Model) openBinForm() (tea.Model, tea.Cmd) {
	ti := textinput.New()
	ti.Placeholder = "llama-server"
	ti.CharLimit = 512
	ti.Width = 60
	ti.SetValue(m.cfg.LlamaServerBin)
	ti.Focus()
	ti.CursorEnd()

	m.settings.bin.input = ti
	m.settings.bin.editing = true
	m.settings.bin.err = ""
	return m, nil
}

func (m Model) submitBinForm() (tea.Model, tea.Cmd) {
	raw := strings.TrimSpace(m.settings.bin.input.Value())
	if raw == "" {
		m.settings.bin.err = "binary path is required"
		return m, nil
	}

	m.cfg.LlamaServerBin = raw
	if err := m.saveConfig(); err != nil {
		m.settings.bin.err = err.Error()
		return m, nil
	}

	m.settings.bin.editing = false
	m.settings.bin.err = ""
	return m, nil
}

// activateDirsRow handles Enter while focus is on the content container:
// row 0 opens the add form, any other row opens that directory for editing.
func (m Model) activateDirsRow() (tea.Model, tea.Cmd) {
	if m.settings.dirs.cursor == 0 {
		return m.openDirForm(-1)
	}
	return m.openDirForm(m.settings.dirs.cursor - 1)
}

// openDirForm opens the inline text input, pre-filled for editing when idx
// is a valid list index, blank for adding a new one when idx is -1.
func (m Model) openDirForm(idx int) (tea.Model, tea.Cmd) {
	ti := textinput.New()
	ti.Placeholder = "~/path/to/models"
	ti.CharLimit = 256
	ti.Width = 50
	if idx >= 0 && idx < len(m.settings.dirs.list) {
		ti.SetValue(m.settings.dirs.list[idx])
	}
	ti.Focus()
	ti.CursorEnd()

	m.settings.dirs.input = ti
	m.settings.dirs.editingIdx = idx
	m.settings.dirs.editing = true
	m.settings.dirs.err = ""
	return m, nil
}

func (m Model) submitDirForm() (tea.Model, tea.Cmd) {
	raw := strings.TrimSpace(m.settings.dirs.input.Value())
	if raw == "" {
		m.settings.dirs.err = "path is required"
		return m, nil
	}

	expanded, err := util.ExpandHome(raw)
	if err != nil {
		m.settings.dirs.err = err.Error()
		return m, nil
	}
	if info, err := os.Stat(expanded); err != nil || !info.IsDir() {
		m.settings.dirs.err = fmt.Sprintf("%s is not a directory", expanded)
		return m, nil
	}

	idx := m.settings.dirs.editingIdx
	if idx >= 0 {
		for i, d := range m.cfg.ModelsDirs {
			if i != idx && d == raw {
				m.settings.dirs.err = "already in the list"
				return m, nil
			}
		}
		m.cfg.ModelsDirs[idx] = raw
	} else if !m.cfg.AddModelsDir(raw) {
		m.settings.dirs.err = "already in the list"
		return m, nil
	}

	if err := m.saveConfig(); err != nil {
		m.settings.dirs.err = err.Error()
		return m, nil
	}

	m.settings.dirs.list = append([]string(nil), m.cfg.ModelsDirs...)
	m.settings.dirs.editing = false
	m.settings.dirs.err = ""
	if idx >= 0 {
		m.settings.dirs.cursor = idx + 1
	} else {
		m.settings.dirs.cursor = len(m.settings.dirs.list)
	}
	return m, nil
}

// deleteDirRow implements press-twice-to-confirm removal, same pattern as
// deleting a profile. Row 0 ("+ Add Directory") isn't deletable.
func (m Model) deleteDirRow() (tea.Model, tea.Cmd) {
	if m.settings.dirs.cursor == 0 {
		return m, nil
	}
	idx := m.settings.dirs.cursor - 1
	if idx < 0 || idx >= len(m.settings.dirs.list) {
		return m, nil
	}
	dir := m.settings.dirs.list[idx]

	if m.settings.dirs.pendingDel != dir {
		m.settings.dirs.pendingDel = dir
		return m, nil
	}
	m.settings.dirs.pendingDel = ""

	m.cfg.RemoveModelsDir(dir)
	if err := m.saveConfig(); err != nil {
		m.settings.dirs.err = err.Error()
		return m, nil
	}

	m.settings.dirs.list = append([]string(nil), m.cfg.ModelsDirs...)
	if m.settings.dirs.cursor > len(m.settings.dirs.list) {
		m.settings.dirs.cursor = len(m.settings.dirs.list)
	}
	return m, nil
}
