package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/sockheadrps/llmctl/internal/models"
)

// moveFocusLeft steps back up the focus hierarchy: Running -> left content
// -> tab bar. At the tab bar it steps to the previous tab.
func (m Model) moveFocusLeft() (tea.Model, tea.Cmd) {
	switch m.focus {
	case focusRunning:
		m.focus = focusLeft
	case focusSettingsContent:
		m.focus = focusLeft
	case focusLeft:
		// Sub-tab header: left switches Recents→Models, or Models→top tabs.
		if m.modelSubTabFocused {
			if m.leftMode == modeRecents {
				m.leftMode = modeModels
			} else {
				m.focus = focusTabs
				m.modelSubTabFocused = false
			}
			return m, nil
		}
		// Inside the Models tree, back out of a model's expanded profile
		// rows to browsing (the model's own row) before stepping up a
		// whole level, mirroring how Up already behaves there.
		if m.leftMode == modeModels {
			if r, ok := m.currentRow(); ok && (r.kind == rowProfile || r.kind == rowAddProfile) {
				m.cursor = indexOfModelRow(m.rows, m.expandedModelKey)
				m.modelProfilesMode = false
				return m, nil
			}
		}
		// From inside the list, step back up to the sub-tab header.
		if m.leftMode == modeModels || m.leftMode == modeRecents {
			m.modelSubTabFocused = true
			if m.leftMode == modeModels {
				m.expandedModelKey = ""
				m.modelProfilesMode = false
				m.rebuildRows()
			}
			return m, nil
		}
		m.focus = focusTabs
	case focusTabs:
		// Overview is conceptually the first tab (leftmost). Going left from
		// Models wraps back to it; going left from Overview does nothing.
		switch m.leftMode {
		case modeOverview:
			// already at the leftmost tab
		case modeModels:
			m.leftMode = modeOverview
		default:
			m.leftMode--
			// Recents is now a sub-tab of Models; skip it at the top-bar level.
			if m.leftMode == modeRecents {
				m.leftMode--
			}
			// Skip hidden optional tabs.
			if m.leftMode == modeRPCServer && !m.cfg.RPCEnabled {
				m.leftMode--
			}
			if m.leftMode == modeNetwork && !m.networkTabVisible() {
				m.leftMode--
			}
		}
	}
	return m, nil
}

// moveFocusRight steps forward: at the tab bar it steps to the next tab;
// from the left pane it jumps to Running (only if something is running) —
// except while browsing the Models tree with a model (not yet one of its
// profiles) under the cursor, where Right instead enters that model's
// profiles, mirroring Enter.
func (m Model) moveFocusRight() (tea.Model, tea.Cmd) {
	switch m.focus {
	case focusTabs:
		maxMode := modeRunning
		if m.networkTabVisible() {
			maxMode = modeNetwork
		}
		if m.cfg.RPCEnabled {
			maxMode = modeRPCServer
		}
		// Overview is conceptually first; Right from it jumps to Models.
		if m.leftMode == modeOverview {
			m.leftMode = modeModels
		} else if m.leftMode < maxMode {
			m.leftMode++
			// Recents is now a sub-tab of Models; skip it at the top-bar level.
			if m.leftMode == modeRecents {
				m.leftMode++
			}
			// Skip hidden optional tabs.
			if m.leftMode == modeNetwork && !m.networkTabVisible() {
				m.leftMode++
			}
		}
	case focusLeft:
		// Sub-tab header: right switches Models→Recents.
		if m.modelSubTabFocused {
			if m.leftMode == modeModels {
				m.leftMode = modeRecents
			} else if len(m.running) > 0 {
				m.focus = focusRunning
				m.modelSubTabFocused = false
			}
			return m, nil
		}
		if m.leftMode == modeModels {
			if r, ok := m.currentRow(); ok && r.kind == rowModel {
				return m.enterModel(r.modelKey)
			}
		}
		// The Running tab already shows this same list in the left pane
		// itself, so jumping to the (now absent, for this tab) glance box
		// would just strand focus with nothing visibly highlighted.
		// The RPC Server tab similarly manages everything in the left pane.
		if m.leftMode != modeRunning && m.leftMode != modeRPCServer && len(m.running) > 0 {
			m.focus = focusRunning
		}
	}
	return m, nil
}

// moveCursor moves within whichever pane/level currently has focus. At the
// tab bar, Down enters the left pane's content; inside content, moving up
// past the first row returns focus to that level.
func (m Model) moveCursor(delta int) (tea.Model, tea.Cmd) {
	switch m.focus {
	case focusTabs:
		if delta > 0 {
			// Overview is display-only; Down stays at the tab bar.
			if m.leftMode == modeOverview {
				return m, nil
			}
			m.focus = focusLeft
			if m.leftMode == modeModels || m.leftMode == modeRecents {
				// Land on the sub-tab header row first.
				m.modelSubTabFocused = true
			}
		}
		return m, nil

	case focusRunning:
		next := m.runningCursor + delta
		if next >= 0 && next < len(m.running) {
			m.runningCursor = next
		}
		return m, nil

	case focusSettingsContent:
		return m.settingsContentMoveCursor(delta)

	default: // focusLeft
		// Sub-tab header: Down enters the list; Up exits to tab bar.
		if m.modelSubTabFocused {
			if delta > 0 {
				m.modelSubTabFocused = false
				if m.leftMode == modeModels {
					m.enterModelsPane()
				}
				// modeRecents: recentCursor already at 0, nothing else needed.
			} else {
				m.focus = focusTabs
				m.modelSubTabFocused = false
			}
			return m, nil
		}

		switch m.leftMode {
		case modeRecents:
			next := m.recentCursor + delta
			switch {
			case next < 0:
				m.modelSubTabFocused = true
			case next < len(m.recentRows):
				m.recentCursor = next
			}
			return m, nil

		case modeSettings:
			next := m.settingsCursor + delta
			switch {
			case next < 0:
				m.focus = focusTabs
			case next < len(m.buildSettingsRows()):
				m.settingsCursor = next
			}
			return m, nil

		case modeRunning:
			next := m.runningCursor + delta
			switch {
			case next < 0:
				m.focus = focusTabs
			case next < len(m.running):
				m.runningCursor = next
			}
			return m, nil

		case modeRPCServer:
			if m.cfg.RPCMode == "server" && m.statusServer != nil {
				addrs := m.statusServerAddrs()
				next := m.rpcIPCursor + delta
				switch {
				case next < 0:
					m.focus = focusTabs
				case next <= len(addrs):
					m.rpcIPCursor = next
					m.rpcAddrCopied = false
				}
			} else if delta < 0 {
				m.focus = focusTabs
			}
			return m, nil

		case modeNetwork:
			next := m.netCursor + delta
			switch {
			case next < 0:
				m.focus = focusTabs
			case next < netRowCount:
				m.netCursor = next
			}
			return m, nil

		default: // modeModels
			return m.moveModelsCursor(delta)
		}
	}
}

// moveModelsCursor moves the cursor through the Models tree. While browsing
// models it only targets model rows and "+ Add Model"; Enter/Right on a
// model switches into that model's profile rows.
func (m Model) moveModelsCursor(delta int) (tea.Model, tea.Cmd) {
	if len(m.rows) == 0 {
		return m, nil
	}

	current := m.cursor
	if current < 0 || current >= len(m.rows) {
		current = 0
	}

	if delta > 0 && m.focus == focusLeft && m.expandedModelKey == "" && current == 0 {
		if len(visibleModelKeys(m.cfg)) > 0 {
			firstKey := visibleModelKeys(m.cfg)[0]
			m.expandedModelKey = firstKey
			m.rebuildRows()
			m.cursor = indexOfModelRow(m.rows, firstKey)
			return m, nil
		}
	}

	inProfiles := m.modelProfilesMode
	if current >= 0 && current < len(m.rows) {
		inProfiles = inProfiles || m.rows[current].kind == rowProfile || m.rows[current].kind == rowAddProfile
	}
	targets := modelCursorTargets(m.rows, inProfiles, m.expandedModelKey)
	if len(targets) == 0 {
		return m, nil
	}

	currentIndex := indexInSlice(targets, current)
	if currentIndex < 0 {
		currentIndex = 0
	}

	nextIndex := currentIndex + delta
	if nextIndex < 0 {
		if inProfiles {
			m.cursor = indexOfModelRow(m.rows, m.expandedModelKey)
			m.modelProfilesMode = false
			return m, nil
		}
		m.modelSubTabFocused = true
		m.expandedModelKey = ""
		m.modelProfilesMode = false
		m.rebuildRows()
		return m, nil
	}
	if nextIndex >= len(targets) {
		if inProfiles {
			m.cursor = targets[0]
		}
		return m, nil
	}

	m.cursor = targets[nextIndex]
	m.modelProfilesMode = inProfiles
	if m.cursor >= 0 && m.cursor < len(m.rows) {
		row := m.rows[m.cursor]
		if row.kind == rowModel && m.expandedModelKey != row.modelKey {
			m.expandedModelKey = row.modelKey
			m.modelProfilesMode = false
			m.rebuildRows()
			m.cursor = indexOfModelRow(m.rows, row.modelKey)
		}
	}
	return m, nil
}

func modelCursorTargets(rows []row, profilesMode bool, expandedModelKey string) []int {
	targets := make([]int, 0, len(rows))
	for i, r := range rows {
		if profilesMode {
			if r.modelKey == expandedModelKey && (r.kind == rowProfile || r.kind == rowAddProfile) {
				targets = append(targets, i)
			}
			continue
		}
		if r.kind == rowModel || r.kind == rowAddModel {
			targets = append(targets, i)
		}
	}
	return targets
}

func (m *Model) enterModelsPane() {
	m.modelProfilesMode = false
	keys := visibleModelKeys(m.cfg)
	if len(keys) == 0 {
		return
	}

	firstKey := keys[0]
	if m.expandedModelKey != firstKey {
		m.expandedModelKey = firstKey
		m.rebuildRows()
	}
	if idx := indexOfModelRow(m.rows, firstKey); idx >= 0 {
		m.cursor = idx
	}
}

func indexInSlice(values []int, target int) int {
	for i, value := range values {
		if value == target {
			return i
		}
	}
	return -1
}

// currentRow resolves the row the left-pane cursor currently points at, in
// whichever mode is active (Models tree, Recents list, or Settings menu).
// This is deliberately independent of focus — the Details panel keeps
// reflecting this row even while focus is on the Running pane or the tab
// bar, same as it always has. Callers that perform an action
// (run/edit/stop/delete) are responsible for checking focus themselves first.
func (m Model) currentRow() (row, bool) {
	switch m.leftMode {
	case modeRecents:
		if m.recentCursor < 0 || m.recentCursor >= len(m.recentRows) {
			return row{}, false
		}
		return m.recentRows[m.recentCursor], true

	case modeSettings:
		rows := m.buildSettingsRows()
		if m.settingsCursor < 0 || m.settingsCursor >= len(rows) {
			return row{}, false
		}
		return rows[m.settingsCursor], true

	case modeRunning:
		if m.runningCursor < 0 || m.runningCursor >= len(m.running) {
			return row{}, false
		}
		r := m.running[m.runningCursor]
		return row{kind: rowRunning, modelKey: r.ModelKey, profileKey: r.ProfileKey, label: r.Label()}, true

	default:
		if m.cursor < 0 || m.cursor >= len(m.rows) {
			return row{}, false
		}
		return m.rows[m.cursor], true
	}
}

// selectRow handles Enter on whichever kind of row is under the cursor.
func (m Model) selectRow() (tea.Model, tea.Cmd) {
	// Enter/Space on the sub-tab header behaves like Down: enter the list.
	if m.focus == focusLeft && m.modelSubTabFocused {
		return m.moveCursor(1)
	}

	switch m.focus {
	case focusTabs:
		m.focus = focusLeft
		return m, nil
	case focusRunning:
		if m.runningCursor >= 0 && m.runningCursor < len(m.running) {
			run := m.running[m.runningCursor]
			return m.openRunningAction(run.ModelKey, run.ProfileKey, run.Label())
		}
		return m, nil
	case focusSettingsContent:
		return m.activateSettingsContentRow()
	}

	if m.leftMode == modeRPCServer && m.focus == focusLeft {
		if m.cfg.RPCMode == "server" {
			if m.rpcIPCursor > 0 {
				return m.copyStatusServerAddr()
			}
			return m.openRPCServerAction()
		}
		return m, nil // client mode: Enter does nothing
	}

	if m.leftMode == modeNetwork && m.focus == focusLeft {
		switch m.netCursor {
		case netRowSwitchRPC, netRowSwitchInternet:
			return m.openNetworkSwitch()
		case netRowSetInternet:
			return m.openNetworkPicker(netPickerRoleInternet)
		case netRowSetRPC:
			return m.openNetworkPicker(netPickerRoleRPC)
		}
		return m, nil
	}

	r, ok := m.currentRow()
	if !ok {
		return m, nil
	}

	switch r.kind {
	case rowModel:
		return m.enterModel(r.modelKey)
	case rowProfile:
		return m.openConfirm(r)
	case rowAddProfile:
		return m.openTemplatePicker(r.modelKey)
	case rowAddModel:
		return m.openPicker()
	case rowSettingsCategory:
		return m.enterSettingsCategory(r.modelKey)
	case rowRunning:
		return m.openRunningAction(r.modelKey, r.profileKey, r.label)
	default:
		return m, nil
	}
}

// enterModel moves the cursor into modelKey's profile rows, expanding it
// first if hovering hadn't already done so (e.g. Enter pressed on the very
// first model before any arrow move). Lands on the first child row —
// a profile, or "+ New Profile" if there are none yet.
func (m Model) enterModel(modelKey string) (tea.Model, tea.Cmd) {
	if m.expandedModelKey != modelKey {
		m.expandedModelKey = modelKey
		m.rebuildRows()
	}
	m.modelProfilesMode = true
	idx := indexOfModelRow(m.rows, modelKey)
	if idx+1 < len(m.rows) && m.rows[idx+1].kind != rowModel {
		m.cursor = idx + 1
	} else {
		m.cursor = idx
	}
	return m, nil
}

// deleteSelected implements press-twice-to-confirm deletion of a profile:
// the first Delete on a profile row just marks it pending, the second
// Delete on that same row (with no other key in between) removes it.
// Deletion isn't available from the Recents tab — that view is a launch
// shortcut, not a place to expect config edits to happen from.
func (m Model) deleteSelected() (tea.Model, tea.Cmd) {
	if m.focus == focusSettingsContent {
		if m.settings.activeCategory != "model_dirs" {
			return m, nil
		}
		return m.deleteDirRow()
	}

	if m.focus != focusLeft || m.leftMode == modeRecents || m.leftMode == modeRunning {
		return m, nil
	}

	r, ok := m.currentRow()
	if !ok || r.kind != rowProfile {
		return m, nil
	}

	if m.pendingDeleteModel != r.modelKey || m.pendingDeleteProfile != r.profileKey {
		m.pendingDeleteModel = r.modelKey
		m.pendingDeleteProfile = r.profileKey
		return m, nil
	}

	m.pendingDeleteModel = ""
	m.pendingDeleteProfile = ""

	for _, run := range m.running {
		if run.ModelKey == r.modelKey && run.ProfileKey == r.profileKey {
			m.setError(fmt.Errorf("cannot delete %s: it is currently running", r.label), "")
			return m, nil
		}
	}

	mdl := m.cfg.Models[r.modelKey]
	delete(mdl.Profiles, r.profileKey)
	m.cfg.Models[r.modelKey] = mdl

	if err := m.saveConfig(); err != nil {
		m.setError(err, "")
		return m, nil
	}

	// That was the model's last profile — it's about to disappear from
	// the tree entirely, so there's nothing left to keep expanded.
	if len(mdl.Profiles) == 0 && m.expandedModelKey == r.modelKey {
		m.expandedModelKey = ""
	}

	m.rebuildRows()
	m.rebuildRecentRows()
	m.clearError()
	return m, nil
}

func (m Model) duplicateSelectedProfile() (tea.Model, tea.Cmd) {
	if m.focus != focusLeft || m.leftMode != modeModels {
		return m, nil
	}
	r, ok := m.currentRow()
	if !ok || r.kind != rowProfile {
		return m, nil
	}

	mdl := m.cfg.Models[r.modelKey]
	source, ok := mdl.Profiles[r.profileKey]
	if !ok {
		return m, nil
	}

	nextKey := uniqueProfileKey(mdl.Profiles, r.profileKey+"-copy")
	copyProfile := source
	copyProfile.Name = nextKey
	copyProfile.Port = suggestPort(m.cfg)
	mdl.Profiles[nextKey] = copyProfile
	m.cfg.Models[r.modelKey] = mdl

	if err := m.saveConfig(); err != nil {
		m.setError(err, "")
		return m, nil
	}

	m.rebuildRows()
	for i, row := range m.rows {
		if row.kind == rowProfile && row.modelKey == r.modelKey && row.profileKey == nextKey {
			m.cursor = i
			break
		}
	}
	m.modelProfilesMode = true
	m.clearError()
	return m, nil
}

func uniqueProfileKey(existing map[string]models.Profile, base string) string {
	base = strings.TrimSpace(base)
	if base == "" {
		base = "profile-copy"
	}
	if _, ok := existing[base]; !ok {
		return base
	}
	for i := 2; ; i++ {
		candidate := fmt.Sprintf("%s-%d", base, i)
		if _, ok := existing[candidate]; !ok {
			return candidate
		}
	}
}

func (m Model) updateModelSearch(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.searchEditing = false
		m.modelSearch = ""
		m.modelProfilesMode = false
		m.rebuildRows()
		return m, nil
	case "enter":
		m.searchEditing = false
		return m, nil
	case "backspace":
		if m.modelSearch != "" {
			m.modelSearch = m.modelSearch[:len(m.modelSearch)-1]
			m.rebuildRows()
		}
		return m, nil
	}

	s := msg.String()
	if len(s) == 1 {
		m.modelSearch += s
		m.modelProfilesMode = false
		m.rebuildRows()
		m.cursor = min(m.cursor, max(0, len(m.rows)-1))
	}
	return m, nil
}
