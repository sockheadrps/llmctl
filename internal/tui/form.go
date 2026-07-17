package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/sockheadrps/llmctl/internal/config"
	tui_form "github.com/sockheadrps/llmctl/internal/tui/form"
)

// openForm switches to the new-profile screen for modelKey, pre-filling a
// suggested free port. overrides is an optional map of field index -> value
// that overwrites specific defaults (used by template presets).
func (m Model) openForm(modelKey string, overrides map[int]string) (tea.Model, tea.Cmd) {
	if _, ok := m.cfg.Models[modelKey]; !ok {
		return m, nil
	}

	init := tui_form.NewProfileFormInit(m.cfg, overrides)
	fields := make([]formField, len(init.Fields))
	for i, field := range init.Fields {
		fields[i] = formField{label: field.Label, input: field.Input}
	}

	m.form = formState{
		modelKey:               modelKey,
		fields:                 fields,
		initial:                init.Initial,
		initialFlash:           init.InitialFlash,
		flash:                  init.Flash,
		initialCPUOnly:         init.InitialCPUOnly,
		cpuOnly:                init.CPUOnly,
		initialMLock:           init.InitialMLock,
		mlock:                  init.MLock,
		rpcClientLayers:        init.RPCClientLayers,
		initialRPCClientLayers: init.InitialRPCClientLayers,
		focus:                  0,
		navigating:             true,
		descDir:                1,
		descPause:              scrollPauseTicks,
		flagInput:              init.FlagInput,
		flagOverrides:          init.FlagOverrides,
		initialFlagOverrides:   init.InitialFlagOverrides,
		importInput:            init.ImportInput,
	}
	m.form.flagInput = tui_form.SyncFlagInput(m.form.flagInput, m.form.focus, len(formLabels), m.form.flagOverrides)
	m.screen = screenNewProfile
	m.clearError()
	return m, nil
}

// openEditForm switches to the edit-profile screen for an existing
// modelKey/profileKey, pre-filling its current settings.
func (m Model) openEditForm(modelKey, profileKey string) (tea.Model, tea.Cmd) {
	mdl, ok := m.cfg.Models[modelKey]
	if !ok {
		m.screen = screenMain
		return m, nil
	}
	p, ok := mdl.Profiles[profileKey]
	if !ok {
		m.screen = screenMain
		return m, nil
	}

	init := tui_form.EditProfileFormInit(profileKey, p)
	fields := make([]formField, len(init.Fields))
	for i, field := range init.Fields {
		fields[i] = formField{label: field.Label, input: field.Input}
	}

	m.form = formState{
		modelKey:               modelKey,
		editing:                true,
		originalKey:            profileKey,
		fields:                 fields,
		initial:                init.Initial,
		initialFlash:           init.InitialFlash,
		flash:                  init.Flash,
		initialCPUOnly:         init.InitialCPUOnly,
		cpuOnly:                init.CPUOnly,
		initialMLock:           init.InitialMLock,
		mlock:                  init.MLock,
		rpcClientLayers:        init.RPCClientLayers,
		initialRPCClientLayers: init.InitialRPCClientLayers,
		focus:                  0,
		navigating:             true,
		descDir:                1,
		descPause:              scrollPauseTicks,
		flagInput:              init.FlagInput,
		flagOverrides:          init.FlagOverrides,
		initialFlagOverrides:   init.InitialFlagOverrides,
		importInput:            init.ImportInput,
	}
	m.form.flagInput = tui_form.SyncFlagInput(m.form.flagInput, m.form.focus, len(formLabels), m.form.flagOverrides)
	m.screen = screenNewProfile
	m.clearError()
	return m, nil
}

func intOrEmpty(n int) string { return tui_form.IntOrEmpty(n) }

func intPtrOrEmpty(n *int) string { return tui_form.IntPtrOrEmpty(n) }

func floatPtrOrEmpty(f *float64) string { return tui_form.FloatPtrOrEmpty(f) }

func boolPtrOrEmpty(b *bool) string { return tui_form.BoolPtrOrEmpty(b) }

func suggestPort(cfg *config.Config) int { return tui_form.SuggestPort(cfg) }

func (m Model) updateForm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Flag override input in the details panel has focus - route keys there.
	if m.form.importEditing {
		switch msg.String() {
		case "esc":
			m.form.importEditing = false
			m.form.importErr = ""
			m.form.importInput = tui_form.CloseImportModal(m.form.importInput)
			return m, nil
		case "enter":
			parsed, err := tui_form.ApplyImportedArgs(m.form.importInput.Value(), len(formLabels), fieldDefaultFlag, len(formLabels), fieldExtraArgs)
			if err != nil {
				m.form.importErr = err.Error()
				return m, nil
			}
			for i := range m.form.fields {
				if i != fieldKey {
					m.form.fields[i].input.SetValue("")
				}
			}
			m.form.flash = false
			m.form.cpuOnly = false
			m.form.mlock = false
			m.form.rpcClientLayers = 0
			for idx, val := range parsed.Values {
				switch idx {
				case len(formLabels):
					m.form.flash = parsed.Flash
				default:
					if idx >= 0 && idx < len(m.form.fields) {
						m.form.fields[idx].input.SetValue(val)
					}
				}
			}
			m.form.flagInput = tui_form.SyncFlagInput(m.form.flagInput, m.form.focus, len(formLabels), m.form.flagOverrides)
			m.form.importEditing = false
			m.form.importErr = ""
			m.form.importInput = tui_form.CloseImportModal(m.form.importInput)
			return m, nil
		default:
			var cmd tea.Cmd
			m.form.importInput, cmd = m.form.importInput.Update(msg)
			return m, cmd
		}
	}

	if m.form.flagFocus {
		switch msg.String() {
		case "esc", "left":
			m.form.flagOverrides = tui_form.CommitFlagInput(m.form.focus, m.form.flagInput, m.form.flagOverrides)
			m.form.flagFocus = false
			m.form.flagInput.Blur()
		case "enter":
			m.form.flagOverrides = tui_form.CommitFlagInput(m.form.focus, m.form.flagInput, m.form.flagOverrides)
			m.form.flagFocus = false
			m.form.flagInput.Blur()
		default:
			var cmd tea.Cmd
			m.form.flagInput, cmd = m.form.flagInput.Update(msg)
			return m, cmd
		}
		return m, nil
	}

	// Edit mode: keystrokes go to the focused text input.
	// Enter commits and returns to navigate mode; Esc cancels without committing.
	if !m.form.navigating {
		switch msg.String() {
		case "esc", "enter":
			for i := range m.form.fields {
				m.form.fields[i].input.Blur()
			}
			m.form.navigating = true
		default:
			if m.form.focus < len(m.form.fields) {
				var cmd tea.Cmd
				m.form.fields[m.form.focus].input, cmd = m.form.fields[m.form.focus].input.Update(msg)
				return m, cmd
			}
		}
		return m, nil
	}

	// Navigate mode: arrows/WASD move between fields; Enter activates a field.
	switch msg.String() {
	case "esc":
		values := make([]string, len(m.form.fields))
		for i := range m.form.fields {
			values[i] = m.form.fields[i].input.Value()
		}
		if tui_form.Dirty(
			m.form.flash, m.form.initialFlash,
			m.form.cpuOnly, m.form.initialCPUOnly,
			m.form.mlock, m.form.initialMLock,
			m.form.rpcClientLayers, m.form.initialRPCClientLayers,
			values, m.form.initial,
			m.form.flagOverrides, m.form.initialFlagOverrides,
		) {
			m.formExit = formExitState{selected: formExitDiscard}
			m.screen = screenFormExitConfirm
			return m, nil
		}
		m.screen = screenMain
		m.clearError()
		return m, nil

	case "left", "a":
		if m.form.focus == len(m.form.fields)+3 && tui_form.FormRPCActive(m.form.fields[fieldRPCEnabled].input.Value(), m.cfg.RPCEnabled) {
			m.form.rpcClientLayers = tui_form.AdjustTensorSplit(m.form.rpcClientLayers, 0, -1)
			return m, nil
		}
		return m, nil

	case "shift+left":
		if m.form.focus == len(m.form.fields)+3 && tui_form.FormRPCActive(m.form.fields[fieldRPCEnabled].input.Value(), m.cfg.RPCEnabled) {
			m.form.rpcClientLayers = tui_form.AdjustTensorSplit(m.form.rpcClientLayers, 0, -5)
			return m, nil
		}
		return m, nil

	case "shift+right":
		if m.form.focus == len(m.form.fields)+3 && tui_form.FormRPCActive(m.form.fields[fieldRPCEnabled].input.Value(), m.cfg.RPCEnabled) {
			total := tui_form.FormSliderTotal(m.form.fields[fieldGPULayers].input.Value())
			m.form.rpcClientLayers = tui_form.AdjustTensorSplit(m.form.rpcClientLayers, total, 5)
			return m, nil
		}
		return m, nil

	case "right", "d":
		if m.form.focus == len(m.form.fields)+3 && tui_form.FormRPCActive(m.form.fields[fieldRPCEnabled].input.Value(), m.cfg.RPCEnabled) {
			total := tui_form.FormSliderTotal(m.form.fields[fieldGPULayers].input.Value())
			m.form.rpcClientLayers = tui_form.AdjustTensorSplit(m.form.rpcClientLayers, total, 1)
			return m, nil
		}
		if flag := tui_form.FocusedFlag(m.form.focus, len(m.form.fields)); flag != "" {
			m.form.flagFocus = true
			m.form.flagInput.Focus()
			if m.form.focus < len(m.form.fields) {
				m.form.fields[m.form.focus].input.Blur()
			}
		}
		return m, nil

	case "x":
		m.form.importEditing = true
		m.form.importErr = ""
		m.form.importInput = tui_form.OpenImportModal(tui_form.BuildImportInput())
		return m, nil

	case "tab", "down", "j", "s":
		m.form.flagOverrides = tui_form.CommitFlagInput(m.form.focus, m.form.flagInput, m.form.flagOverrides)
		m.form.flagFocus = false
		m.form.flagInput.Blur()
		for i := range m.form.fields {
			m.form.fields[i].input.Blur()
		}
		m.form.focus = tui_form.NextFocus(m.form.focus, 1, len(m.form.fields))
		m.form.descScroll, m.form.descDir, m.form.descPause = tui_form.ResetDescriptionScroll(scrollPauseTicks)
		m.form.scroll = tui_form.EnsureVisible(m.form.focus, m.form.scroll, tui_form.FormVisibleRows(tui_form.FormPaneHeight(m.height)), len(tui_form.FormNavOrder(len(m.form.fields))))
		m.form.flagInput = tui_form.SyncFlagInput(m.form.flagInput, m.form.focus, len(formLabels), m.form.flagOverrides)
		return m, nil

	case "shift+tab", "up", "k", "w":
		m.form.flagOverrides = tui_form.CommitFlagInput(m.form.focus, m.form.flagInput, m.form.flagOverrides)
		m.form.flagFocus = false
		m.form.flagInput.Blur()
		for i := range m.form.fields {
			m.form.fields[i].input.Blur()
		}
		m.form.focus = tui_form.NextFocus(m.form.focus, -1, len(m.form.fields))
		m.form.descScroll, m.form.descDir, m.form.descPause = tui_form.ResetDescriptionScroll(scrollPauseTicks)
		m.form.scroll = tui_form.EnsureVisible(m.form.focus, m.form.scroll, tui_form.FormVisibleRows(tui_form.FormPaneHeight(m.height)), len(tui_form.FormNavOrder(len(m.form.fields))))
		m.form.flagInput = tui_form.SyncFlagInput(m.form.flagInput, m.form.focus, len(formLabels), m.form.flagOverrides)
		return m, nil

	case "enter":
		switch m.form.focus {
		case len(m.form.fields):
			m.form.flash = !m.form.flash
		case len(m.form.fields) + 1:
			m.form.cpuOnly = !m.form.cpuOnly
		case len(m.form.fields) + 2:
			m.form.mlock = !m.form.mlock
		case len(m.form.fields) + 3:
			// tensor split slider: adjusted with <- ->, Enter does nothing
		case len(m.form.fields) + 4:
			return m.submitForm()
		default:
			m.form.navigating = false
			if m.form.focus < len(m.form.fields) {
				m.form.fields[m.form.focus].input.Focus()
			}
		}
		return m, nil
	}

	return m, nil
}

func (m Model) submitForm() (tea.Model, tea.Cmd) {
	values := make([]string, len(m.form.fields))
	for i := range m.form.fields {
		values[i] = m.form.fields[i].input.Value()
	}
	m.form.flagOverrides = tui_form.CommitFlagInput(m.form.focus, m.form.flagInput, m.form.flagOverrides)
	submission, err := tui_form.BuildProfileSubmission(values, m.form.editing, m.form.originalKey, m.cfg.Models[m.form.modelKey].Profiles, m.form.flash, m.form.cpuOnly, m.form.mlock, m.form.rpcClientLayers, m.cfg.RPCEnabled, m.form.flagOverrides)
	if err != nil {
		m.form.err = err.Error()
		return m, nil
	}
	tui_form.CommitProfileSubmission(m.cfg, m.form.modelKey, m.form.originalKey, m.form.editing, submission)

	if err := m.saveConfig(); err != nil {
		m.setError(err, "")
		m.screen = screenMain
		return m, nil
	}

	m.rebuildRows()
	m.rebuildRecentRows()
	m.cursor = indexOfProfileRow(m.rows, m.form.modelKey, submission.Key)
	m.screen = screenMain
	m.clearError()
	return m, nil
}

func indexOfProfileRow(rows []row, modelKey, profileKey string) int {
	for i, r := range rows {
		if r.kind == rowProfile && r.modelKey == modelKey && r.profileKey == profileKey {
			return i
		}
	}
	return 0
}
