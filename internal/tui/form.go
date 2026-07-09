package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/sockheadrps/llmctl/internal/config"
	"github.com/sockheadrps/llmctl/internal/models"
	"github.com/sockheadrps/llmctl/internal/util"
)



func (f *formState) focusedFlag() string {
	if f.focus == len(formLabels) {
		return "--flash-attn"
	}
	if f.focus == len(formLabels)+2 {
		return "--mlock"
	}
	return fieldDefaultFlag(f.focus)
}

func (f *formState) syncFlagInput() {
	def := f.focusedFlag()
	if def == "" {
		f.flagInput.SetValue("")
		return
	}
	val := def
	if override, ok := f.flagOverrides[def]; ok {
		val = override
	}
	f.flagInput.SetValue(val)
}

func (f *formState) commitFlagInput() {
	def := f.focusedFlag()
	if def == "" {
		return
	}
	val := strings.TrimSpace(f.flagInput.Value())
	if val == "" || val == def {
		delete(f.flagOverrides, def)
	} else {
		if f.flagOverrides == nil {
			f.flagOverrides = make(map[string]string)
		}
		f.flagOverrides[def] = val
	}
}

func buildFlagInput() textinput.Model {
	ti := textinput.New()
	ti.Prompt = ""
	ti.CharLimit = 64
	ti.Width = 22
	return ti
}

func buildImportInput() textinput.Model {
	ti := textinput.New()
	ti.Prompt = ""
	ti.Placeholder = "paste CLI args here..."
	ti.CharLimit = 1024
	ti.Width = 40
	return ti
}


func copyStringMap(m map[string]string) map[string]string {
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}



func (f *formState) blurAll() {
	for i := range f.fields {
		f.fields[i].input.Blur()
	}
}

func (f *formState) moveFocus(delta int, visibleRows int) {
	f.commitFlagInput()
	f.flagFocus = false
	f.flagInput.Blur()
	total := len(f.fields) + 5 // + flash toggle + cpu only toggle + mlock toggle + tensor split slider + save action
	f.blurAll()
	f.focus = ((f.focus+delta)%total + total) % total
	f.resetDescriptionScroll()
	// Don't auto-focus the text input: navigate mode controls activation via Enter.
	f.ensureVisible(visibleRows, total)
	f.syncFlagInput()
}

func (f *formState) resetDescriptionScroll() {
	f.descScroll = 0
	f.descDir = 1
	f.descPause = scrollPauseTicks
}

func (f *formState) advanceDescriptionScroll(lines, visible int) {
	f.descScroll, f.descDir, f.descPause = advanceAutoScroll(f.descScroll, f.descDir, f.descPause, lines, visible)
}

func (f formState) dirty() bool {
	if f.flash != f.initialFlash {
		return true
	}
	if f.cpuOnly != f.initialCPUOnly {
		return true
	}
	if f.mlock != f.initialMLock {
		return true
	}
	if f.rpcClientLayers != f.initialRPCClientLayers {
		return true
	}
	if len(f.initial) != len(f.fields) {
		return true
	}
	for i := range f.fields {
		if f.fields[i].input.Value() != f.initial[i] {
			return true
		}
	}
	if len(f.flagOverrides) != len(f.initialFlagOverrides) {
		return true
	}
	for k, v := range f.flagOverrides {
		if f.initialFlagOverrides[k] != v {
			return true
		}
	}
	return false
}

func (f *formState) ensureVisible(visibleRows int, totalRows int) {
	if visibleRows <= 0 {
		visibleRows = 1
	}
	if totalRows <= visibleRows {
		f.scroll = 0
		return
	}
	if f.focus < f.scroll {
		f.scroll = f.focus
	} else if f.focus >= f.scroll+visibleRows {
		f.scroll = f.focus - visibleRows + 1
	}
	maxScroll := totalRows - visibleRows
	if f.scroll < 0 {
		f.scroll = 0
	} else if f.scroll > maxScroll {
		f.scroll = maxScroll
	}
}

func (f *formState) openImportModal() {
	f.importEditing = true
	f.importErr = ""
	f.importInput = buildImportInput()
	f.importInput.Focus()
}

func (f *formState) closeImportModal() {
	f.importEditing = false
	f.importErr = ""
	f.importInput.Blur()
}


// openForm switches to the new-profile screen for modelKey, pre-filling a
// suggested free port. overrides is an optional map of field index → value
// that overwrites specific defaults (used by template presets).
func (m Model) openForm(modelKey string, overrides map[int]string) (tea.Model, tea.Cmd) {
	if _, ok := m.cfg.Models[modelKey]; !ok {
		return m, nil
	}

	// Presence/Repetition penalty start blank on purpose: llama-server's
	// own defaults for those are already no-ops (0.0 and 1.0), so leaving
	// them unset until the user opts in avoids emitting flags that do
	// nothing but add noise to the command line.
	defaults := make([]string, len(formLabels))
	defaults[fieldKey] = ""
	defaults[fieldHost] = ""
	defaults[fieldAlias] = ""
	defaults[fieldPort] = strconv.Itoa(suggestPort(m.cfg))
	defaults[fieldCtxSize] = "8192"
	defaults[fieldTemp] = "0.6"
	defaults[fieldTopP] = "0.95"
	defaults[fieldTopK] = "20"
	defaults[fieldMinP] = "0.0"
	defaults[fieldPresencePenalty] = ""
	defaults[fieldRepetitionPenalty] = ""
	defaults[fieldFrequencyPenalty] = ""
	defaults[fieldSeed] = ""
	defaults[fieldBatchSize] = ""
	defaults[fieldUBatchSize] = ""
	defaults[fieldRepeatLastN] = ""
	defaults[fieldGPULayers] = "999"
	defaults[fieldMMap] = ""
	defaults[fieldKVOffload] = ""
	defaults[fieldParallelSlots] = ""
	defaults[fieldContBatching] = ""
	defaults[fieldCachePrompt] = ""
	defaults[fieldCacheRAM] = ""
	defaults[fieldReasoning] = ""
	defaults[fieldReasoningBudget] = ""
	defaults[fieldReasoningFormat] = ""
	defaults[fieldCacheK] = ""
	defaults[fieldCacheV] = ""
	defaults[fieldExtraArgs] = ""
	defaults[fieldNotes] = ""
	defaults[fieldRPCEnabled] = ""

	for idx, val := range overrides {
		if idx >= 0 && idx < len(defaults) {
			defaults[idx] = val
		}
	}

	fi := buildFlagInput()
	m.form = formState{
		modelKey:               modelKey,
		fields:                 buildFormFields(defaults),
		initial:                append([]string(nil), defaults...),
		initialFlash:           true,
		flash:                  true,
		initialCPUOnly:         false,
		cpuOnly:                false,
		initialMLock:           false,
		mlock:                  false,
		rpcClientLayers:        0,
		initialRPCClientLayers: 0,
		focus:                  0,
		navigating:             true,
		descDir:              1,
		descPause:            scrollPauseTicks,
		flagInput:            fi,
		flagOverrides:        make(map[string]string),
		initialFlagOverrides: make(map[string]string),
		importInput:          buildImportInput(),
	}
	m.form.syncFlagInput()
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

	defaults := make([]string, len(formLabels))
	defaults[fieldKey] = profileKey
	defaults[fieldHost] = p.Host
	defaults[fieldAlias] = p.Alias
	defaults[fieldPort] = strconv.Itoa(p.Port)
	defaults[fieldCtxSize] = intOrEmpty(p.CtxSize)
	defaults[fieldTemp] = floatPtrOrEmpty(p.Temp)
	defaults[fieldTopP] = floatPtrOrEmpty(p.TopP)
	defaults[fieldTopK] = intPtrOrEmpty(p.TopK)
	defaults[fieldMinP] = floatPtrOrEmpty(p.MinP)
	defaults[fieldPresencePenalty] = floatPtrOrEmpty(p.PresencePenalty)
	defaults[fieldRepetitionPenalty] = floatPtrOrEmpty(p.RepetitionPenalty)
	defaults[fieldFrequencyPenalty] = floatPtrOrEmpty(p.FrequencyPenalty)
	defaults[fieldSeed] = intPtrOrEmpty(p.Seed)
	defaults[fieldBatchSize] = intPtrOrEmpty(p.BatchSize)
	defaults[fieldUBatchSize] = intPtrOrEmpty(p.UBatchSize)
	defaults[fieldRepeatLastN] = intPtrOrEmpty(p.RepeatLastN)
	defaults[fieldGPULayers] = intOrEmpty(p.GPULayers)
	defaults[fieldMMap] = boolPtrOrEmpty(p.MMap)
	defaults[fieldKVOffload] = boolPtrOrEmpty(p.KVOffload)
	defaults[fieldParallelSlots] = intPtrOrEmpty(p.Parallel)
	defaults[fieldContBatching] = boolPtrOrEmpty(p.ContBatching)
	defaults[fieldCachePrompt] = boolPtrOrEmpty(p.CachePrompt)
	defaults[fieldCacheRAM] = intPtrOrEmpty(p.CacheRAM)
	defaults[fieldReasoning] = p.Reasoning
	defaults[fieldReasoningBudget] = intPtrOrEmpty(p.ReasoningBudget)
	defaults[fieldReasoningFormat] = p.ReasoningFormat
	defaults[fieldCacheK] = p.CacheTypeK
	defaults[fieldCacheV] = p.CacheTypeV
	defaults[fieldExtraArgs] = strings.Join(p.ExtraArgs, " ")
	defaults[fieldNotes] = p.Notes
	defaults[fieldRPCEnabled] = boolPtrOrEmpty(p.RPCEnabled)

	initClientLayers := 0
	if p.TensorSplit != "" {
		parts := strings.SplitN(p.TensorSplit, ",", 2)
		if n, err := strconv.Atoi(strings.TrimSpace(parts[0])); err == nil && n >= 0 {
			initClientLayers = n
		}
	}

	fi := buildFlagInput()
	m.form = formState{
		modelKey:               modelKey,
		editing:                true,
		originalKey:            profileKey,
		fields:                 buildFormFields(defaults),
		initial:                append([]string(nil), defaults...),
		initialFlash:           p.FlashAttn,
		flash:                  p.FlashAttn,
		initialCPUOnly:         p.CPUOnly,
		cpuOnly:                p.CPUOnly,
		initialMLock:           p.MLock,
		mlock:                  p.MLock,
		rpcClientLayers:        initClientLayers,
		initialRPCClientLayers: initClientLayers,
		focus:                  0,
		navigating:             true,
		descDir:              1,
		descPause:            scrollPauseTicks,
		flagInput:            fi,
		flagOverrides:        copyStringMap(p.FlagOverrides),
		initialFlagOverrides: copyStringMap(p.FlagOverrides),
		importInput:          buildImportInput(),
	}
	m.form.syncFlagInput()
	m.screen = screenNewProfile
	m.clearError()
	return m, nil
}

func intOrEmpty(n int) string {
	if n == 0 {
		return ""
	}
	return strconv.Itoa(n)
}

func intPtrOrEmpty(n *int) string {
	if n == nil {
		return ""
	}
	return strconv.Itoa(*n)
}


func floatPtrOrEmpty(f *float64) string {
	if f == nil {
		return ""
	}
	return strconv.FormatFloat(*f, 'f', -1, 64)
}

func suggestPort(cfg *config.Config) int {
	maxPort := 8079
	for _, mdl := range cfg.Models {
		for _, p := range mdl.Profiles {
			if p.Port > maxPort {
				maxPort = p.Port
			}
		}
	}
	start := maxPort + 1
	if free, err := util.FindFreePort(start); err == nil {
		return free
	}
	return start
}

func (m Model) updateForm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Flag override input in the details panel has focus — route keys there.
	if m.form.importEditing {
		switch msg.String() {
		case "esc":
			m.form.closeImportModal()
			return m, nil
		case "enter":
			if err := m.form.applyImportedArgs(m.form.importInput.Value()); err != nil {
				m.form.importErr = err.Error()
				return m, nil
			}
			m.form.closeImportModal()
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
			m.form.commitFlagInput()
			m.form.flagFocus = false
			m.form.flagInput.Blur()
		case "enter":
			m.form.commitFlagInput()
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
		case "esc":
			m.form.blurAll()
			m.form.navigating = true
		case "enter":
			m.form.blurAll()
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
		if m.form.dirty() {
			m.formExit = formExitState{selected: formExitDiscard}
			m.screen = screenFormExitConfirm
			return m, nil
		}
		m.screen = screenMain
		m.clearError()
		return m, nil

	case "left", "a":
		if m.form.focus == len(m.form.fields)+3 {
			if m.form.rpcClientLayers > 0 {
				m.form.rpcClientLayers--
			}
			return m, nil
		}
		return m, nil

	case "shift+left":
		if m.form.focus == len(m.form.fields)+3 {
			m.form.rpcClientLayers -= 5
			if m.form.rpcClientLayers < 0 {
				m.form.rpcClientLayers = 0
			}
			return m, nil
		}
		return m, nil

	case "shift+right":
		if m.form.focus == len(m.form.fields)+3 {
			total := m.formSliderTotal()
			m.form.rpcClientLayers += 5
			if total > 0 && m.form.rpcClientLayers > total {
				m.form.rpcClientLayers = total
			}
			return m, nil
		}
		return m, nil

	case "right", "d":
		if m.form.focus == len(m.form.fields)+3 {
			total := m.formSliderTotal()
			if total > 0 && m.form.rpcClientLayers < total {
				m.form.rpcClientLayers++
			}
			return m, nil
		}
		if flag := m.form.focusedFlag(); flag != "" {
			m.form.flagFocus = true
			m.form.flagInput.Focus()
			if m.form.focus < len(m.form.fields) {
				m.form.fields[m.form.focus].input.Blur()
			}
		}
		return m, nil

	case "x":
		m.form.openImportModal()
		return m, nil

	case "tab", "down", "j", "s":
		m.form.moveFocus(1, m.formVisibleRows())
		return m, nil

	case "shift+tab", "up", "k", "w":
		m.form.moveFocus(-1, m.formVisibleRows())
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
			// tensor split slider: adjusted with ← →, Enter does nothing
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
	value := func(i int) string { return strings.TrimSpace(m.form.fields[i].input.Value()) }

	key := value(fieldKey)
	if key == "" {
		m.form.err = "key is required"
		return m, nil
	}

	mdl := m.cfg.Models[m.form.modelKey]
	renamed := m.form.editing && key != m.form.originalKey
	if _, exists := mdl.Profiles[key]; exists && (!m.form.editing || renamed) {
		m.form.err = fmt.Sprintf("profile %q already exists on this model", key)
		return m, nil
	}

	port, err := strconv.Atoi(value(fieldPort))
	if err != nil || port <= 0 {
		m.form.err = "port must be a positive integer"
		return m, nil
	}
	ctxSize, err := parseIntOrZero(value(fieldCtxSize))
	if err != nil {
		m.form.err = "ctx size must be an integer"
		return m, nil
	}
	temp, err := parseFloatPtr(value(fieldTemp))
	if err != nil {
		m.form.err = "temp must be a number"
		return m, nil
	}
	topP, err := parseFloatPtr(value(fieldTopP))
	if err != nil {
		m.form.err = "top p must be a number"
		return m, nil
	}
	topK, err := parseIntPtr(value(fieldTopK))
	if err != nil {
		m.form.err = "top k must be an integer"
		return m, nil
	}
	minP, err := parseFloatPtr(value(fieldMinP))
	if err != nil {
		m.form.err = "min p must be a number"
		return m, nil
	}
	presencePenalty, err := parseFloatPtr(value(fieldPresencePenalty))
	if err != nil {
		m.form.err = "presence penalty must be a number"
		return m, nil
	}
	repetitionPenalty, err := parseFloatPtr(value(fieldRepetitionPenalty))
	if err != nil {
		m.form.err = "repetition penalty must be a number"
		return m, nil
	}
	frequencyPenalty, err := parseFloatPtr(value(fieldFrequencyPenalty))
	if err != nil {
		m.form.err = "frequency penalty must be a number"
		return m, nil
	}
	seed, err := parseIntPtr(value(fieldSeed))
	if err != nil {
		m.form.err = "seed must be an integer"
		return m, nil
	}
	batchSize, err := parseIntPtr(value(fieldBatchSize))
	if err != nil {
		m.form.err = "batch size must be an integer"
		return m, nil
	}
	ubatchSize, err := parseIntPtr(value(fieldUBatchSize))
	if err != nil {
		m.form.err = "ubatch size must be an integer"
		return m, nil
	}
	repeatLastN, err := parseIntPtr(value(fieldRepeatLastN))
	if err != nil {
		m.form.err = "repeat last n must be an integer"
		return m, nil
	}
	gpuLayers, err := parseIntOrZero(value(fieldGPULayers))
	if err != nil {
		m.form.err = "gpu layers must be an integer"
		return m, nil
	}
	mmap, err := parseBoolPtr(value(fieldMMap))
	if err != nil {
		m.form.err = "mmap must be true or false"
		return m, nil
	}
	kvOffload, err := parseBoolPtr(value(fieldKVOffload))
	if err != nil {
		m.form.err = "kv offload must be true or false"
		return m, nil
	}
	parallelSlots, err := parseIntPtr(value(fieldParallelSlots))
	if err != nil {
		m.form.err = "parallel slots must be an integer"
		return m, nil
	}
	contBatching, err := parseBoolPtr(value(fieldContBatching))
	if err != nil {
		m.form.err = "continuous batching must be true or false"
		return m, nil
	}
	cachePrompt, err := parseBoolPtr(value(fieldCachePrompt))
	if err != nil {
		m.form.err = "prompt cache must be true or false"
		return m, nil
	}
	cacheRAM, err := parseIntPtr(value(fieldCacheRAM))
	if err != nil {
		m.form.err = "cache ram must be an integer"
		return m, nil
	}
	reasoning, err := parseReasoning(value(fieldReasoning))
	if err != nil {
		m.form.err = err.Error()
		return m, nil
	}
	reasoningBudget, err := parseIntPtr(value(fieldReasoningBudget))
	if err != nil {
		m.form.err = "reasoning budget must be an integer"
		return m, nil
	}

	rpcEnabled, err := parseBoolPtr(value(fieldRPCEnabled))
	if err != nil {
		m.form.err = "rpc enabled must be true or false"
		return m, nil
	}

	// Space-separated so multi-token flags like "-np 1" or "--spec-type
	// draft-mtp" split into the right argv elements — exec.Command passes
	// each ExtraArgs entry as a literal argument with no shell-splitting,
	// so "-np 1" as a single element would reach llama-server malformed.
	var extraArgs []string
	if raw := value(fieldExtraArgs); raw != "" {
		extraArgs = strings.Fields(raw)
	}

	if mdl.Profiles == nil {
		mdl.Profiles = map[string]models.Profile{}
	}
	if renamed {
		delete(mdl.Profiles, m.form.originalKey)
	}
	tensorSplit := ""
	if !m.form.cpuOnly && gpuLayers > 0 {
		client := m.form.rpcClientLayers
		if client < 0 {
			client = 0
		}
		if client > gpuLayers {
			client = gpuLayers
		}
		tensorSplit = fmt.Sprintf("%d,%d", client, gpuLayers-client)
	}

	mdl.Profiles[key] = models.Profile{
		Name:              key,
		Host:              value(fieldHost),
		Alias:             value(fieldAlias),
		Port:              port,
		CtxSize:           ctxSize,
		Temp:              temp,
		TopP:              topP,
		TopK:              topK,
		MinP:              minP,
		PresencePenalty:   presencePenalty,
		RepetitionPenalty: repetitionPenalty,
		FrequencyPenalty:  frequencyPenalty,
		Seed:              seed,
		BatchSize:         batchSize,
		UBatchSize:        ubatchSize,
		RepeatLastN:       repeatLastN,
		FlashAttn:         m.form.flash,
		CPUOnly:           m.form.cpuOnly,
		MLock:             m.form.mlock,
		GPULayers:         gpuLayers,
		MMap:              mmap,
		KVOffload:         kvOffload,
		Parallel:          parallelSlots,
		ContBatching:      contBatching,
		CachePrompt:       cachePrompt,
		CacheRAM:          cacheRAM,
		Reasoning:         reasoning,
		ReasoningBudget:   reasoningBudget,
		ReasoningFormat:   value(fieldReasoningFormat),
		CacheTypeK:        value(fieldCacheK),
		CacheTypeV:        value(fieldCacheV),
		ExtraArgs:      extraArgs,
		Notes:          value(fieldNotes),
		RPCEnabled:     rpcEnabled,
		TensorSplit:    tensorSplit,
		FlagOverrides: func() map[string]string {
			m.form.commitFlagInput()
			if len(m.form.flagOverrides) == 0 {
				return nil
			}
			return copyStringMap(m.form.flagOverrides)
		}(),
	}
	m.cfg.Models[m.form.modelKey] = mdl

	if err := m.saveConfig(); err != nil {
		m.setError(err, "")
		m.screen = screenMain
		return m, nil
	}

	modelKey := m.form.modelKey
	m.rebuildRows()
	m.rebuildRecentRows()
	m.cursor = indexOfProfileRow(m.rows, modelKey, key)
	m.screen = screenMain
	m.clearError()
	return m, nil
}

func (m Model) formVisibleRows() int {
	return max(1, m.formPaneHeight()-1)
}

func (m Model) formPaneHeight() int {
	if m.height <= 0 {
		return 20
	}
	// title + blank line + bordered pane + newline + hotkey line
	return max(8, m.height-6)
}


func indexOfProfileRow(rows []row, modelKey, profileKey string) int {
	for i, r := range rows {
		if r.kind == rowProfile && r.modelKey == modelKey && r.profileKey == profileKey {
			return i
		}
	}
	return 0
}
