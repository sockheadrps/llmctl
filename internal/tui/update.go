package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/sockheadrps/llmctl/internal/models"
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

	case scrollTickMsg:
		switch m.screen {
		case screenNewProfile:
			m.form.advanceDescriptionScroll(m.formDescriptionLineCount(), m.formDescriptionVisibleLines())
		case screenMain:
			m.advanceDetailsScroll(m.mainDetailsLineCount(), m.mainDetailsVisibleLines())
		}
		return m, scrollTickCmd()

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

	case netStatusMsg:
		m.netStatus = msg
		return m, nil

	case netSwitchResultMsg:
		m.netSwitching = false
		if msg.err != nil {
			m.setError(msg.err, "")
			return m, nil
		}
		m.clearError()
		return m, checkNetworkStatusCmd(m.netIface, m.netInternetConn, m.netRPCConn)

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
		default:
			return m.updateMain(msg)
		}
	}

	return m, nil
}

func (m Model) updateMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	leftW, _, _ := m.paneDimensions()
	dividerLeft := leftW + 1  // right border of left pane
	dividerRight := leftW + 2 // left border of right pane

	// Use the actual rendered runningH (accounts for content overflow) to
	// locate the horizontal divider. Y=0 header, Y=1 body top border, so
	// divider row = 1 (top border) + runningH + 1 (bottom border of running) = runningH+2.
	_, actualRunningH, actualDetailsH := m.mainDetailsGeometry()
	hDividerY := actualRunningH + 2
	inRightColumn := msg.X > dividerRight

	switch msg.Action {
	case tea.MouseActionPress:
		if msg.Button != tea.MouseButtonLeft {
			break
		}
		if msg.X == dividerLeft || msg.X == dividerRight {
			m.dividerDragging = true
		} else if inRightColumn && m.leftMode != modeRunning {
			if msg.Y >= hDividerY-1 && msg.Y <= hDividerY+2 {
				m.rightDividerDragging = true
			}
		}

	case tea.MouseActionMotion:
		if m.dividerDragging {
			newLeft := msg.X - 1
			avail := m.width - 4
			if newLeft < minLeftWidth {
				newLeft = minLeftWidth
			}
			if newLeft > avail-minRightWidth {
				newLeft = avail - minRightWidth
			}
			m.leftWidthOverride = newLeft
		}
		if m.rightDividerDragging {
			// newRunningH = drag Y minus header row minus body top border.
			// The minimum is the raw content line count of the running list
			// (so the box is never set smaller than its content — that would
			// trigger the overflow correction which can push leftH past the
			// terminal height). actualRunningH is the rendered box height
			// (padded/filled), not the content height, so we measure content
			// directly.
			_, rightW, _ := m.paneDimensions()
			rightMeasure := lipgloss.NewStyle().Width(rightW).Padding(0, 1)
			contentH := lipgloss.Height(rightMeasure.Render(m.renderRunning()))
			minRunning := max(3, contentH)

			newRunningH := msg.Y - 2
			totalBudget := actualRunningH + actualDetailsH
			if newRunningH < minRunning {
				newRunningH = minRunning
			}
			if newRunningH > totalBudget-3 {
				newRunningH = totalBudget - 3
			}
			m.rightSplitOverride = newRunningH
		}

	case tea.MouseActionRelease:
		m.dividerDragging = false
		m.rightDividerDragging = false
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
	if m.focus == focusSettingsContent && m.settings.rpc.binEditing {
		switch msg.String() {
		case "esc":
			m.settings.rpc.binEditing = false
			m.settings.rpc.err = ""
			return m, nil
		case "enter":
			return m.submitRPCBinForm()
		}
		var cmd tea.Cmd
		m.settings.rpc.binInput, cmd = m.settings.rpc.binInput.Update(msg)
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

func (m Model) openNetworkSwitch() (tea.Model, tea.Cmd) {
	m.netSwitch = netSwitchState{
		toRPC:  m.netCursor == netRowSwitchRPC,
		cursor: 0,
	}
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
				m.modelProfilesMode = false
				return m, nil
			}
		}
		m.focus = focusTabs
		if m.leftMode == modeModels {
			m.expandedModelKey = ""
			m.modelProfilesMode = false
			m.rebuildRows()
		}
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
		if m.leftMode < modeNetwork {
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
			if m.leftMode == modeModels {
				m.enterModelsPane()
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

		case modeNetwork:
			next := m.netCursor + delta
			switch {
			case next < 0:
				m.focus = focusTabs
			case next < 2:
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
		m.focus = focusTabs
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

// selectRow handles Enter on whichever kind of row is under the cursor.
func (m Model) selectRow() (tea.Model, tea.Cmd) {
	switch m.focus {
	case focusTabs:
		m.focus = focusLeft
		return m, nil
	case focusRunning:
		if m.runningCursor >= 0 && m.runningCursor < len(m.running) {
			run := m.running[m.runningCursor]
			return m.openStopConfirm(run.ModelKey, run.ProfileKey, run.Label())
		}
		return m, nil
	case focusSettingsContent:
		return m.activateSettingsContentRow()
	}

	if m.leftMode == modeNetwork && m.focus == focusLeft {
		return m.openNetworkSwitch()
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
		return m.openStopConfirm(r.modelKey, r.profileKey, r.label)
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
