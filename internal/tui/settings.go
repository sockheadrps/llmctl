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
}

// settingsState backs the Settings tab's content — currently just Model
// Directories, but a struct (rather than dirsContentState directly) leaves
// room to add another category's state alongside it later.
type settingsState struct {
	dirs dirsContentState
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

// enterSettingsCategory moves focus into the selected category's content in
// the Details pane — the caller (selectRow, on Enter) already picked the
// category, so there's nothing more to navigate before showing it. State is
// loaded fresh from config so edits made elsewhere (or a previous visit)
// aren't stale.
func (m Model) enterSettingsCategory(categoryID string) (tea.Model, tea.Cmd) {
	switch categoryID {
	case "model_dirs":
		m.settings.dirs = dirsContentState{list: append([]string(nil), m.cfg.ModelsDirs...)}
	}
	m.focus = focusSettingsContent
	m.clearError()
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
