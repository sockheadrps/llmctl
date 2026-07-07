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

// formField is one text input row in the new-profile form.
type formField struct {
	label string
	input textinput.Model
}

// formState backs the "New Profile"/"Edit Profile" screen. Focus indices
// 0..len(fields)-1 are the text fields, len(fields) is the Flash Attention
// toggle, and len(fields)+1 is the Save action.
//
// navigating=true is navigate mode (arrow/WASD moves between fields; Enter
// activates a field for editing). navigating=false is edit mode (keystrokes
// go to the focused textinput; Enter commits and returns to navigate mode).
type formState struct {
	modelKey             string
	editing              bool
	originalKey          string
	fields               []formField
	initial              []string
	initialFlash         bool
	flash                bool
	focus                int
	scroll               int
	descScroll           int
	descDir              int
	descPause            int
	err                  string
	navigating           bool
	flagFocus            bool
	flagInput            textinput.Model
	flagOverrides        map[string]string
	initialFlagOverrides map[string]string
	importEditing        bool
	importInput          textinput.Model
	importErr            string
}

// fieldDefaultFlag returns the default llama-server CLI flag for a form field
// index, or "" for fields that don't map to a single CLI flag.
func fieldDefaultFlag(idx int) string {
	switch idx {
	case fieldHost:
		return "--host"
	case fieldAlias:
		return "--alias"
	case fieldPort:
		return "--port"
	case fieldCtxSize:
		return "--ctx-size"
	case fieldTemp:
		return "--temp"
	case fieldTopP:
		return "--top-p"
	case fieldTopK:
		return "--top-k"
	case fieldMinP:
		return "--min-p"
	case fieldPresencePenalty:
		return "--presence-penalty"
	case fieldRepetitionPenalty:
		return "--repeat-penalty"
	case fieldFrequencyPenalty:
		return "--frequency-penalty"
	case fieldSeed:
		return "--seed"
	case fieldBatchSize:
		return "--batch-size"
	case fieldUBatchSize:
		return "--ubatch-size"
	case fieldRepeatLastN:
		return "--repeat-last-n"
	case fieldGPULayers:
		return "--n-gpu-layers"
	case fieldMMap:
		return "--mmap"
	case fieldKVOffload:
		return "--kv-offload"
	case fieldParallelSlots:
		return "--parallel"
	case fieldContBatching:
		return "--cont-batching"
	case fieldCachePrompt:
		return "--cache-prompt"
	case fieldCacheRAM:
		return "--cache-ram"
	case fieldReasoning:
		return "--reasoning"
	case fieldReasoningBudget:
		return "--reasoning-budget"
	case fieldReasoningFormat:
		return "--reasoning-format"
	case fieldCacheK:
		return "--cache-type-k"
	case fieldCacheV:
		return "--cache-type-v"
	}
	return ""
}

func (f *formState) focusedFlag() string {
	if f.focus == len(formLabels) {
		return "--flash-attn"
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

func splitCLIArgs(argsStr string) []string {
	var tokens []string
	var cur strings.Builder
	inSingle := false
	inDouble := false
	escaped := false
	started := false

	flush := func() {
		if started {
			tokens = append(tokens, cur.String())
			cur.Reset()
			started = false
		}
	}

	for _, r := range argsStr {
		switch {
		case escaped:
			cur.WriteRune(r)
			escaped = false
			started = true
		case r == '\\' && !inSingle:
			escaped = true
			started = true
		case r == '\'' && !inDouble:
			inSingle = !inSingle
			started = true
		case r == '"' && !inSingle:
			inDouble = !inDouble
			started = true
		case !inSingle && !inDouble && (r == ' ' || r == '\t' || r == '\n' || r == '\r'):
			flush()
		default:
			cur.WriteRune(r)
			started = true
		}
	}
	flush()
	return tokens
}

func isNumericToken(s string) bool {
	if s == "" {
		return false
	}
	if s[0] == '-' {
		s = s[1:]
	}
	if s == "" {
		return false
	}
	dotSeen := false
	digitSeen := false
	for i := 0; i < len(s); i++ {
		switch {
		case s[i] >= '0' && s[i] <= '9':
			digitSeen = true
		case s[i] == '.':
			if dotSeen {
				return false
			}
			dotSeen = true
		default:
			return false
		}
	}
	return digitSeen
}

// parseProfileArgs parses a space-separated CLI arg string (e.g. copied from
// export) and returns a map of field-index → value. Index len(formLabels) is
// used for the flash-attention toggle.
func parseProfileArgs(argsStr string) (map[int]string, []string) {
	tokens := splitCLIArgs(argsStr)
	result := make(map[int]string)
	extra := make([]string, 0)
	sawRelevantFlag := false

	flagToField := make(map[string]int)
	for i := 0; i < len(formLabels); i++ {
		if f := fieldDefaultFlag(i); f != "" {
			flagToField[f] = i
		}
	}

	isKnownFlag := func(tok string) bool {
		if tok == "--flash-attn" {
			return true
		}
		if strings.HasPrefix(tok, "--no-") {
			return true
		}
		if eqIdx := strings.Index(tok, "="); eqIdx > 0 && strings.HasPrefix(tok, "-") {
			tok = tok[:eqIdx]
		}
		_, ok := flagToField[tok]
		return ok
	}

	consumeValue := func(i int) (string, bool, int) {
		if i+1 >= len(tokens) {
			return "", false, i
		}
		next := tokens[i+1]
		if isKnownFlag(next) {
			return "", false, i
		}
		if strings.HasPrefix(next, "-") && !isNumericToken(next) {
			return "", false, i
		}
		return next, true, i + 1
	}

	skipModelSource := func(i int) int {
		if i+1 < len(tokens) && !strings.HasPrefix(tokens[i+1], "-") {
			return i + 2
		}
		return i + 1
	}

	i := 0
	for i < len(tokens) {
		tok := tokens[i]

		// Drop the leading binary token from a pasted full command line.
		if i == 0 && !strings.HasPrefix(tok, "-") {
			i++
			continue
		}

		if tok == "--model" || tok == "-m" || tok == "-hf" {
			i = skipModelSource(i)
			continue
		}

		if tok == "llama-server" || tok == "llama-server.exe" {
			i++
			continue
		}

		// --flash-attn[=value]
		if tok == "--flash-attn" || strings.HasPrefix(tok, "--flash-attn=") {
			sawRelevantFlag = true
			val := "on"
			if eqIdx := strings.Index(tok, "="); eqIdx > 0 {
				raw := strings.ToLower(strings.TrimSpace(tok[eqIdx+1:]))
				if raw == "off" || raw == "false" || raw == "0" {
					val = "off"
				}
				result[len(formLabels)] = val
				i++
				continue
			}
			if next, ok, nextIdx := consumeValue(i); ok {
				raw := strings.ToLower(strings.TrimSpace(next))
				if raw == "off" || raw == "false" || raw == "0" {
					val = "off"
				}
				result[len(formLabels)] = val
				i = nextIdx + 1
				continue
			}
			result[len(formLabels)] = val
			i++
			continue
		}

		// --flag=value
		if eqIdx := strings.Index(tok, "="); eqIdx > 0 && strings.HasPrefix(tok, "-") {
			flagPart := tok[:eqIdx]
			if fieldIdx, ok := flagToField[flagPart]; ok {
				sawRelevantFlag = true
				result[fieldIdx] = tok[eqIdx+1:]
			} else {
				extra = append(extra, tok)
			}
			i++
			continue
		}

		// --no-flag → bool false
		if strings.HasPrefix(tok, "--no-") {
			posFlag := "--" + tok[len("--no-"):]
			if fieldIdx, ok := flagToField[posFlag]; ok {
				sawRelevantFlag = true
				result[fieldIdx] = "false"
			} else {
				extra = append(extra, tok)
			}
			i++
			continue
		}

		// --flag value  or  bare --flag (= true)
		if strings.HasPrefix(tok, "-") {
			if fieldIdx, ok := flagToField[tok]; ok {
				sawRelevantFlag = true
				if next, ok, nextIdx := consumeValue(i); ok {
					result[fieldIdx] = next
					i = nextIdx + 1
				} else {
					result[fieldIdx] = "true"
					i++
				}
				continue
			}
		}

		if !sawRelevantFlag && !strings.HasPrefix(tok, "-") {
			i++
			continue
		}

		extra = append(extra, tok)
		i++
	}
	if len(extra) > 0 {
		result[fieldExtraArgs] = strings.Join(extra, " ")
	}
	return result, extra
}

func copyStringMap(m map[string]string) map[string]string {
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// Field indices into formState.fields, matching the order of formLabels.
const (
	fieldKey = iota
	fieldHost
	fieldAlias
	fieldPort
	fieldCtxSize
	fieldTemp
	fieldTopP
	fieldTopK
	fieldMinP
	fieldPresencePenalty
	fieldRepetitionPenalty
	fieldFrequencyPenalty
	fieldSeed
	fieldBatchSize
	fieldUBatchSize
	fieldRepeatLastN
	fieldGPULayers
	fieldMMap
	fieldKVOffload
	fieldParallelSlots
	fieldContBatching
	fieldCachePrompt
	fieldCacheRAM
	fieldReasoning
	fieldReasoningBudget
	fieldReasoningFormat
	fieldCacheK
	fieldCacheV
	fieldExtraArgs
	fieldNotes
	fieldRPCEnabled
)

var formLabels = []string{
	"Key", "Host", "Alias", "Port", "Ctx Size", "Temp", "Top P", "Top K", "Min P",
	"Presence Penalty", "Repetition Penalty", "Frequency Penalty", "Seed",
	"Batch Size", "UBatch Size", "Repeat Last N", "GPU Layers", "MMap", "KV Offload",
	"Parallel Slots", "Continuous Batching", "Prompt Cache", "Cache RAM",
	"Reasoning", "Reasoning Budget", "Reasoning Format", "Cache Type K", "Cache Type V",
	"Extra Args (space-separated)", "Notes",
	"RPC Enabled",
}

func buildFormFields(defaults []string) []formField {
	fields := make([]formField, len(formLabels))
	for i, label := range formLabels {
		ti := textinput.New()
		ti.Prompt = ""
		ti.Placeholder = label
		ti.SetValue(defaults[i])
		ti.CharLimit = 256
		ti.Width = 40
		fields[i] = formField{label: label, input: ti}
	}
	return fields
}

func formFieldDescription(idx int) string {
	switch idx {
	case fieldHost:
		return "Sets the network interface that llama-server listens on, such as 127.0.0.1 for local-only access or 0.0.0.0 for all interfaces."
	case fieldAlias:
		return "A friendly identifier for the profile that can help distinguish multiple profiles using the same model."
	case fieldPort:
		return "The TCP port used by the server. Each running profile should use a unique port."
	case fieldCtxSize:
		return "The model context window size. Larger values increase memory usage and allow longer conversations."
	case fieldTemp:
		return "Controls how random the generated output is. Lower values are more deterministic."
	case fieldTopP:
		return "Limits sampling to the smallest set of tokens whose cumulative probability exceeds this value."
	case fieldTopK:
		return "Restricts sampling to the most likely K tokens."
	case fieldMinP:
		return "Filters out low-probability tokens. A value of 0.0 disables this filter."
	case fieldPresencePenalty:
		return "Encourages the model to introduce new topics rather than repeating earlier ones."
	case fieldRepetitionPenalty:
		return "Discourages repeated text and can reduce loops or repetition."
	case fieldFrequencyPenalty:
		return "Penalizes tokens that have already appeared frequently to improve diversity."
	case fieldSeed:
		return "Sets the random seed for reproducible results. Use -1 for randomness."
	case fieldBatchSize:
		return "Maximum logical prompt processing batch size. Larger values can improve prompt throughput but need more memory."
	case fieldUBatchSize:
		return "Maximum physical micro-batch size. Advanced tuning option for throughput and memory tradeoffs."
	case fieldRepeatLastN:
		return "Number of recent tokens considered when applying repetition penalties."
	case fieldGPULayers:
		return "How many transformer layers to load on the GPU. Larger values usually increase performance."
	case fieldMMap:
		return "Enables memory-mapped model loading for faster startup and lower RAM use."
	case fieldKVOffload:
		return "Lets KV cache operations use the GPU. This can improve performance on supported hardware."
	case fieldParallelSlots:
		return "How many simultaneous inference slots the server should support."
	case fieldContBatching:
		return "Enables dynamic batching across multiple clients for better throughput."
	case fieldCachePrompt:
		return "Caches prompt processing to speed up repeated requests."
	case fieldCacheRAM:
		return "Maximum RAM allocated for prompt caching."
	case fieldReasoning:
		return "Turns reasoning mode on, off, or auto for compatible models."
	case fieldReasoningBudget:
		return "Sets a budget for reasoning tokens to limit thinking time and latency."
	case fieldReasoningFormat:
		return "Changes how reasoning content is returned, such as auto, none, or DeepSeek-style output."
	case fieldCacheK:
		return "The data type used for the key portion of the KV cache."
	case fieldCacheV:
		return "The data type used for the value portion of the KV cache."
	case fieldExtraArgs:
		return "Additional raw llama.cpp arguments, split by spaces, for advanced or experimental features."
	case fieldNotes:
		return "Optional notes for this profile that help you remember how it was intended to be used."
	case fieldRPCEnabled:
		return "Override the global RPC setting for this profile. Leave blank to follow the global setting; set true to force RPC on, false to force it off."
	case len(formLabels):
		return "The Flash Attention toggle enables hardware-optimized attention when supported by your build."
	case len(formLabels) + 1:
		return "Save this profile to your model configuration and return to the main view."
	default:
		return "Adjust this option to change how llama-server starts for this profile."
	}
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
	total := len(f.fields) + 2 // + flash toggle + save action
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

func (f *formState) applyImportedArgs(argsStr string) error {
	values, extra := parseProfileArgs(argsStr)
	if len(values) == 0 && len(extra) == 0 {
		return fmt.Errorf("no recognizable CLI args found")
	}

	for idx, val := range values {
		switch idx {
		case len(formLabels):
			f.flash = strings.EqualFold(val, "on") || strings.EqualFold(val, "true") || val == "1"
		default:
			if idx >= 0 && idx < len(f.fields) {
				f.fields[idx].input.SetValue(val)
			}
		}
	}

	f.syncFlagInput()
	f.importErr = ""
	return nil
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
		modelKey:             modelKey,
		fields:               buildFormFields(defaults),
		initial:              append([]string(nil), defaults...),
		initialFlash:         true,
		flash:                true,
		focus:                0,
		navigating:           true,
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

	fi := buildFlagInput()
	m.form = formState{
		modelKey:             modelKey,
		editing:              true,
		originalKey:          profileKey,
		fields:               buildFormFields(defaults),
		initial:              append([]string(nil), defaults...),
		initialFlash:         p.FlashAttn,
		flash:                p.FlashAttn,
		focus:                0,
		navigating:           true,
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

func boolPtrOrEmpty(b *bool) string {
	if b == nil {
		return ""
	}
	if *b {
		return "true"
	}
	return "false"
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

	case "right", "d":
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

func parseIntOrZero(s string) (int, error) {
	if s == "" {
		return 0, nil
	}
	return strconv.Atoi(s)
}

func parseBoolPtr(s string) (*bool, error) {
	if s == "" {
		return nil, nil
	}
	v, err := strconv.ParseBool(s)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func parseReasoning(s string) (string, error) {
	if s == "" {
		return "", nil
	}
	s = strings.ToLower(strings.TrimSpace(s))
	switch s {
	case "on", "off", "auto":
		return s, nil
	default:
		return "", fmt.Errorf("reasoning must be on, off, or auto")
	}
}

// parseIntPtr and parseFloatPtr return nil for a blank field — used for
// sampling params where an explicit 0 (e.g. min_p=0.0 to disable min-p
// filtering) must be distinguishable from "flag omitted, use the
// llama-server default", per Profile's doc comment.
func parseIntPtr(s string) (*int, error) {
	if s == "" {
		return nil, nil
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func parseFloatPtr(s string) (*float64, error) {
	if s == "" {
		return nil, nil
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func indexOfProfileRow(rows []row, modelKey, profileKey string) int {
	for i, r := range rows {
		if r.kind == rowProfile && r.modelKey == modelKey && r.profileKey == profileKey {
			return i
		}
	}
	return 0
}
