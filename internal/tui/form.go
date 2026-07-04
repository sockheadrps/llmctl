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
type formState struct {
	modelKey    string
	editing     bool
	originalKey string
	fields      []formField
	flash       bool
	focus       int
	err         string
}

// Field indices into formState.fields, matching the order of formLabels.
const (
	fieldKey = iota
	fieldPort
	fieldCtxSize
	fieldTemp
	fieldTopP
	fieldTopK
	fieldMinP
	fieldPresencePenalty
	fieldRepetitionPenalty
	fieldGPULayers
	fieldCacheK
	fieldCacheV
	fieldExtraArgs
	fieldNotes
)

var formLabels = []string{
	"Key", "Port", "Ctx Size", "Temp", "Top P", "Top K", "Min P",
	"Presence Penalty", "Repetition Penalty",
	"GPU Layers", "Cache Type K", "Cache Type V", "Extra Args (space-separated)", "Notes",
}

func buildFormFields(defaults []string) []formField {
	fields := make([]formField, len(formLabels))
	for i, label := range formLabels {
		ti := textinput.New()
		ti.Placeholder = label
		ti.SetValue(defaults[i])
		ti.CharLimit = 256
		ti.Width = 40
		fields[i] = formField{label: label, input: ti}
	}
	fields[0].input.Focus()
	return fields
}

func (f *formState) blurAll() {
	for i := range f.fields {
		f.fields[i].input.Blur()
	}
}

func (f *formState) moveFocus(delta int) {
	total := len(f.fields) + 2 // + flash toggle + save action
	f.blurAll()
	f.focus = ((f.focus+delta)%total + total) % total
	if f.focus < len(f.fields) {
		f.fields[f.focus].input.Focus()
	}
}

// openForm switches to the new-profile screen for modelKey, pre-filling a
// suggested free port.
func (m Model) openForm(modelKey string) (tea.Model, tea.Cmd) {
	if _, ok := m.cfg.Models[modelKey]; !ok {
		return m, nil
	}

	// Presence/Repetition penalty start blank on purpose: llama-server's
	// own defaults for those are already no-ops (0.0 and 1.0), so leaving
	// them unset until the user opts in avoids emitting flags that do
	// nothing but add noise to the command line.
	defaults := []string{
		"", strconv.Itoa(suggestPort(m.cfg)), "8192", "0.6", "0.95", "20", "0.0",
		"", "",
		"999", "", "", "", "",
	}

	m.form = formState{modelKey: modelKey, fields: buildFormFields(defaults), flash: true, focus: 0}
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

	defaults := []string{
		profileKey,
		strconv.Itoa(p.Port),
		intOrEmpty(p.CtxSize),
		floatPtrOrEmpty(p.Temp),
		floatPtrOrEmpty(p.TopP),
		intPtrOrEmpty(p.TopK),
		floatPtrOrEmpty(p.MinP),
		floatPtrOrEmpty(p.PresencePenalty),
		floatPtrOrEmpty(p.RepetitionPenalty),
		intOrEmpty(p.GPULayers),
		p.CacheTypeK,
		p.CacheTypeV,
		strings.Join(p.ExtraArgs, " "),
		p.Notes,
	}

	m.form = formState{
		modelKey:    modelKey,
		editing:     true,
		originalKey: profileKey,
		fields:      buildFormFields(defaults),
		flash:       p.FlashAttn,
		focus:       0,
	}
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
	switch msg.String() {
	case "esc":
		m.screen = screenMain
		m.clearError()
		return m, nil

	case "tab", "down":
		m.form.moveFocus(1)
		return m, nil

	case "shift+tab", "up":
		m.form.moveFocus(-1)
		return m, nil

	case " ":
		if m.form.focus == len(m.form.fields) {
			m.form.flash = !m.form.flash
			return m, nil
		}

	case "enter":
		switch m.form.focus {
		case len(m.form.fields):
			m.form.flash = !m.form.flash
			return m, nil
		case len(m.form.fields) + 1:
			return m.submitForm()
		default:
			m.form.moveFocus(1)
			return m, nil
		}
	}

	if m.form.focus < len(m.form.fields) {
		var cmd tea.Cmd
		m.form.fields[m.form.focus].input, cmd = m.form.fields[m.form.focus].input.Update(msg)
		return m, cmd
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
	gpuLayers, err := parseIntOrZero(value(fieldGPULayers))
	if err != nil {
		m.form.err = "gpu layers must be an integer"
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
		Port:              port,
		CtxSize:           ctxSize,
		Temp:              temp,
		TopP:              topP,
		TopK:              topK,
		MinP:              minP,
		PresencePenalty:   presencePenalty,
		RepetitionPenalty: repetitionPenalty,
		FlashAttn:         m.form.flash,
		GPULayers:         gpuLayers,
		CacheTypeK:        value(fieldCacheK),
		CacheTypeV:        value(fieldCacheV),
		ExtraArgs:         extraArgs,
		Notes:             value(fieldNotes),
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

func parseIntOrZero(s string) (int, error) {
	if s == "" {
		return 0, nil
	}
	return strconv.Atoi(s)
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
