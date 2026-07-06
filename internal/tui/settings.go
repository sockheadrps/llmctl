package tui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/sockheadrps/llmctl/internal/util"
)

// settingsCategoryDef is one entry in the Settings tab's category list. id
// is the stable key selectRow/enterSettingsCategory use to route to it;
// label is what's displayed.
type settingsCategoryDef struct {
	id    string
	label string
}

// settingsCategories is the single source of truth for the Settings tab's
// left-pane category list (via buildSettingsRows). Add a new settings page
// by appending here and giving renderSettingsDetail a case for its id.
var settingsCategories = []settingsCategoryDef{
	{id: "model_dirs", label: "Model Directories"},
	{id: "llama_bin", label: "llama-server Binary"},
	{id: "rpc", label: "RPC Server"},
}

// settingsState backs the Settings tab's content — currently just Model
// Directories, but a struct (rather than dirsContentState directly) leaves
// room to add another category's state alongside it later.
type rpcContentState struct {
	cursor     int // 0 = toggle RPC, 1 = endpoint, 2 = binary, 3 = network tab (Linux only)
	editing    bool
	input      textinput.Model
	binEditing bool
	binInput   textinput.Model
	err        string
}

type settingsState struct {
	activeCategory string
	dirs           dirsContentState
	bin            binContentState
	rpc            rpcContentState
}

// dirsContentState is the Model Directories category's content: the
// configured directory list (row 0 is always "+ Add Directory", rows
// 1..N are list[0..N-1]) plus an inline add/edit form.
type dirsContentState struct {
	list       []string
	cursor     int // 0 = "+ Add Directory"; i = list[i-1] for i>0
	pendingDel string

	editing    bool
	editingIdx int // index into list being edited; -1 while adding new
	input      textinput.Model
	err        string
}

type binContentState struct {
	editing bool
	input   textinput.Model
	err     string
}

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
		maxRPCCursor := 2
		if m.netSupported && m.cfg.RPCEnabled {
			maxRPCCursor = 3
		}
		next := m.settings.rpc.cursor + delta
		switch {
		case next < 0:
			m.focus = focusLeft
		case next <= maxRPCCursor:
			m.settings.rpc.cursor = next
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

func (m Model) activateRPCRow() (tea.Model, tea.Cmd) {
	switch m.settings.rpc.cursor {
	case 0:
		wasEnabled := m.cfg.RPCEnabled
		m.cfg.RPCEnabled = !m.cfg.RPCEnabled
		if m.cfg.RPCEnabled && !wasEnabled {
			m.cfg.NetworkTabEnabled = true
		}
		if err := m.saveConfig(); err != nil {
			m.settings.rpc.err = err.Error()
		}
		return m, nil
	case 1:
		return m.openRPCEndpointForm()
	case 2:
		return m.openRPCBinForm()
	case 3:
		if m.netSupported && m.cfg.RPCEnabled {
			m.cfg.NetworkTabEnabled = !m.cfg.NetworkTabEnabled
			if err := m.saveConfig(); err != nil {
				m.settings.rpc.err = err.Error()
			}
		}
		return m, nil
	}
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

func (m Model) openRPCBinForm() (tea.Model, tea.Cmd) {
	ti := textinput.New()
	ti.Placeholder = "llama-server"
	ti.CharLimit = 512
	ti.Width = 50
	ti.SetValue(m.cfg.RPCServerBin)
	ti.Focus()
	ti.CursorEnd()
	m.settings.rpc.binInput = ti
	m.settings.rpc.binEditing = true
	m.settings.rpc.err = ""
	return m, nil
}

func (m Model) submitRPCBinForm() (tea.Model, tea.Cmd) {
	val := strings.TrimSpace(m.settings.rpc.binInput.Value())
	m.cfg.RPCServerBin = val
	if err := m.saveConfig(); err != nil {
		m.settings.rpc.err = err.Error()
		return m, nil
	}
	m.settings.rpc.binEditing = false
	m.settings.rpc.err = ""
	return m, nil
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
