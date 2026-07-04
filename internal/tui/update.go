package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tickMsg:
		if m.screen != screenMain {
			return m, tickCmd()
		}
		m.refreshRunning(true)
		return m, tea.Batch(tickCmd(), m.backgroundChecks())

	case healthMsg:
		m.health = msg
		return m, nil

	case slotsMsg:
		m.applyTokSamples(msg)
		return m, nil

	case vramMsg:
		m.gpuUsage = msg.usage
		m.gpuByPID = msg.byPID
		return m, nil

	case startResultMsg:
		m.starting = false
		m.startingLabel = ""
		if msg.err != nil {
			m.setError(msg.err, msg.logPath)
			return m, nil
		}
		m.refreshRunning(false)
		m.rebuildRecentRows()
		m.clearError()
		return m, m.backgroundChecks()

	case stopResultMsg:
		m.stopping = false
		m.stoppingLabel = ""
		if msg.err != nil {
			m.setError(msg.err, "")
			return m, nil
		}
		m.refreshRunning(false)
		m.clearError()
		return m, nil

	case tea.KeyMsg:
		switch m.screen {
		case screenPickModel:
			return m.updatePicker(msg)
		case screenNewProfile:
			return m.updateForm(msg)
		case screenConfirmProfile:
			return m.updateConfirm(msg)
		case screenLogs:
			return m.updateLogs(msg)
		case screenRunningAction:
			return m.updateRunningAction(msg)
		default:
			return m.updateMain(msg)
		}
	}

	return m, nil
}

func (m Model) updateMain(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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
		return m.moveFocusLeft()

	case key.Matches(msg, keys.Right):
		return m.moveFocusRight()

	case key.Matches(msg, keys.Up):
		return m.moveCursor(-1)

	case key.Matches(msg, keys.Down):
		return m.moveCursor(1)

	case key.Matches(msg, keys.Run):
		return m.selectRow()

	case key.Matches(msg, keys.Stop):
		return m.stopSelected()

	case key.Matches(msg, keys.Delete):
		return m.deleteSelected()

	case key.Matches(msg, keys.Logs):
		return m.openLogsForCurrent()
	}

	return m, nil
}

// moveFocusLeft steps back up the focus hierarchy: Running -> left content
// -> tab bar. At the tab bar it steps to the previous tab.
func (m Model) moveFocusLeft() (tea.Model, tea.Cmd) {
	switch m.focus {
	case focusRunning:
		m.focus = focusLeft
	case focusSettingsContent:
		m.focus = focusLeft
	case focusLeft:
		// Inside the Models tree, back out of a model's expanded profile
		// rows to browsing (the model's own row) before stepping up a
		// whole level, mirroring how Up already behaves there.
		if m.leftMode == modeModels {
			if r, ok := m.currentRow(); ok && (r.kind == rowProfile || r.kind == rowAddProfile) {
				m.cursor = indexOfModelRow(m.rows, m.expandedModelKey)
				return m, nil
			}
		}
		m.focus = focusTabs
	case focusTabs:
		if m.leftMode > modeModels {
			m.leftMode--
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
		if m.leftMode < modeRunning {
			m.leftMode++
		}
	case focusLeft:
		if m.leftMode == modeModels {
			if r, ok := m.currentRow(); ok && r.kind == rowModel {
				return m.enterModel(r.modelKey)
			}
		}
		// The Running tab already shows this same list in the left pane
		// itself, so jumping to the (now absent, for this tab) glance box
		// would just strand focus with nothing visibly highlighted.
		if m.leftMode != modeRunning && len(m.running) > 0 {
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
			m.focus = focusLeft
		}
		return m, nil

	case focusRunning:
		next := m.runningCursor + delta
		if next >= 0 && next < len(m.running) {
			m.runningCursor = next
		}
		return m, nil

	case focusSettingsContent:
		next := m.settings.dirs.cursor + delta
		switch {
		case next < 0:
			m.focus = focusLeft
		case next <= len(m.settings.dirs.list):
			m.settings.dirs.cursor = next
		}
		return m, nil

	default: // focusLeft
		switch m.leftMode {
		case modeRecents:
			next := m.recentCursor + delta
			switch {
			case next < 0:
				m.focus = focusTabs
			case next < len(m.recentRows):
				m.recentCursor = next
			}
			return m, nil

		case modeSettings:
			next := m.settingsCursor + delta
			switch {
			case next < 0:
				m.focus = focusTabs
			case next < len(buildSettingsRows()):
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

		default: // modeModels
			return m.moveModelsCursor(delta)
		}
	}
}

// moveModelsCursor moves the cursor within the Models tree. While
// "browsing" — the cursor sits on a model's own row — Up/Down step between
// models only, skipping over any profile rows, and hovering a model
// auto-expands its profiles into the tree below it as a preview
// (accordion-style, collapsing whichever model was previously expanded).
// Once Enter has moved the cursor onto one of those child rows ("entered"),
// Up/Down instead step through that model's children, only backing out to
// browsing when stepping past the top or bottom child.
func (m Model) moveModelsCursor(delta int) (tea.Model, tea.Cmd) {
	if r, ok := m.currentRow(); ok && r.kind != rowModel {
		return m.moveWithinExpandedModel(delta)
	}
	return m.browseModels(delta)
}

// browseModels moves delta steps through visibleModelKeys, expanding the
// newly-focused model as a preview.
func (m Model) browseModels(delta int) (tea.Model, tea.Cmd) {
	keys := visibleModelKeys(m.cfg)
	if len(keys) == 0 {
		if delta < 0 {
			m.focus = focusTabs
		}
		return m, nil
	}

	pos := 0
	curRow, curIsModel := m.currentRow()
	if curIsModel {
		for i, k := range keys {
			if k == curRow.modelKey {
				pos = i
				break
			}
		}
	}

	next := pos + delta
	switch {
	case next < 0:
		m.focus = focusTabs
		return m, nil
	case next >= len(keys):
		return m, nil
	case next == pos && curIsModel:
		// Already on this model — just expand it in place.
		targetKey := keys[next]
		if m.expandedModelKey != targetKey {
			m.expandedModelKey = targetKey
			m.rebuildRows()
		}
		m.cursor = indexOfModelRow(m.rows, targetKey)
		return m, nil
	}

	targetKey := keys[next]
	if m.expandedModelKey != targetKey {
		m.expandedModelKey = targetKey
		m.rebuildRows()
	}
	m.cursor = indexOfModelRow(m.rows, targetKey)
	return m, nil
}

// moveWithinExpandedModel steps the cursor through the currently-expanded
// model's profile/add-profile rows, clamping at either end rather than
// spilling into a neighboring model's rows — stepping past the top backs
// out to browsing (cursor returns to the model's own row); stepping past
// the bottom is a no-op.
func (m Model) moveWithinExpandedModel(delta int) (tea.Model, tea.Cmd) {
	modelIdx := indexOfModelRow(m.rows, m.expandedModelKey)

	lastChildIdx := modelIdx
	for i := modelIdx + 1; i < len(m.rows) && m.rows[i].kind != rowModel; i++ {
		lastChildIdx = i
	}

	next := m.cursor + delta
	switch {
	case next <= modelIdx:
		m.cursor = modelIdx
	case next <= lastChildIdx:
		m.cursor = next
	}
	return m, nil
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
		rows := buildSettingsRows()
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

// selectRow handles Enter on whichever kind of row is under the cursor:
// open the Run/Edit confirm for a profile, the add-model picker, or the
// new-profile form. On the tab bar, Enter just drops into that tab's
// content, same as pressing Down. It's a no-op from the Running pane —
// that pane's rows aren't run/edit targets, just use 's' to stop one.
func (m Model) selectRow() (tea.Model, tea.Cmd) {
	switch m.focus {
	case focusTabs:
		if m.leftMode == modeModels {
			m.focus = focusLeft
		} else {
			m.focus = focusLeft
		}
		return m, nil
	case focusRunning:
		return m, nil
	case focusSettingsContent:
		return m.activateDirsRow()
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
		return m.openForm(r.modelKey)
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
	idx := indexOfModelRow(m.rows, modelKey)
	if idx+1 < len(m.rows) && m.rows[idx+1].kind != rowModel {
		m.cursor = idx + 1
	} else {
		m.cursor = idx
	}
	return m, nil
}

func (m Model) stopSelected() (tea.Model, tea.Cmd) {
	if m.focus == focusRunning {
		if m.runningCursor < 0 || m.runningCursor >= len(m.running) {
			return m, nil
		}
		run := m.running[m.runningCursor]
		return m.stopRunning(run.ModelKey, run.ProfileKey, run.Label())
	}

	if m.focus != focusLeft {
		return m, nil
	}
	r, ok := m.currentRow()
	if !ok || (r.kind != rowProfile && r.kind != rowRunning) {
		return m, nil
	}
	return m.stopRunning(r.modelKey, r.profileKey, r.label)
}

// deleteSelected implements press-twice-to-confirm deletion of a profile:
// the first Delete on a profile row just marks it pending, the second
// Delete on that same row (with no other key in between) removes it.
// Deletion isn't available from the Recents tab — that view is a launch
// shortcut, not a place to expect config edits to happen from.
func (m Model) deleteSelected() (tea.Model, tea.Cmd) {
	if m.focus == focusSettingsContent {
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
