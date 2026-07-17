package tui

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/sockheadrps/llmctl/internal/config"
	"github.com/sockheadrps/llmctl/internal/models"
)

// pickerState backs the "Add Model" screen: a list of .gguf files found
// under the config's models_dirs that aren't already registered. unreadable
// notes which configured directories couldn't be scanned (missing, no
// permission, …) — a per-directory problem, not fatal to the whole screen.
type pickerState struct {
	files      []string
	cursor     int
	err        error
	unreadable []string
}

// openPicker scans every configured models directory for importable GGUF
// files and switches to the model-picker screen. A directory that can't be
// scanned (not mounted yet, typo, …) is skipped rather than failing the
// whole picker — its problem is surfaced via pickerState.unreadable instead.
func (m Model) openPicker() (tea.Model, tea.Cmd) {
	dirs, err := m.cfg.ResolvedModelsDirs()
	if err != nil {
		m.setError(err, "")
		return m, nil
	}

	// Only paths belonging to models that are actually visible (i.e. have
	// at least one profile) count as "already imported". A model hidden
	// for having zero profiles should be offered again so re-selecting it
	// can bring it back rather than dead-ending at "no new files found".
	used := make(map[string]bool, len(m.cfg.Models))
	for _, mdl := range m.cfg.Models {
		if len(mdl.Profiles) > 0 {
			used[mdl.Path] = true
		}
	}

	var files, unreadable []string
	for _, dir := range dirs {
		found, err := models.ScanGGUF(dir)
		if err != nil {
			unreadable = append(unreadable, dir)
			continue
		}
		for _, f := range found {
			if !used[f] {
				files = append(files, f)
			}
		}
	}
	sort.Strings(files)

	if len(dirs) == 0 {
		m.picker = pickerState{err: fmt.Errorf("no model directories configured — add one from the Model Directories screen")}
	} else {
		m.picker = pickerState{files: files, unreadable: unreadable}
	}
	m.screen = screenPickModel
	m.clearError()
	return m, nil
}

func (m Model) updatePicker(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q":
		m.screen = screenMain
		return m, nil

	case "up", "k":
		if m.picker.cursor > 0 {
			m.picker.cursor--
		}
		return m, nil

	case "down", "j":
		if m.picker.cursor < len(m.picker.files)-1 {
			m.picker.cursor++
		}
		return m, nil

	case "enter":
		return m.importSelectedModel()
	}
	return m, nil
}

// importSelectedModel adds the picked GGUF file as a model, or — if it
// matches a model that already exists but is currently hidden for having
// no profiles — reuses that entry instead of creating a duplicate. Either
// way the result gets a "default" profile so it has something to run and
// isn't immediately hidden again by buildRows.
func (m Model) importSelectedModel() (tea.Model, tea.Cmd) {
	if len(m.picker.files) == 0 {
		return m, nil
	}
	path := m.picker.files[m.picker.cursor]

	key := modelKeyByPath(m.cfg.Models, path)
	if key == "" {
		key = uniqueModelKey(m.cfg.Models, modelKeyFromPath(path))
		m.cfg.Models[key] = models.Model{
			Key:      key,
			Name:     modelNameFromPath(path),
			Path:     path,
			Profiles: map[string]models.Profile{},
		}
	}

	mdl := m.cfg.Models[key]
	if len(mdl.Profiles) == 0 {
		if mdl.Profiles == nil {
			mdl.Profiles = map[string]models.Profile{}
		}
		mdl.Profiles["default"] = defaultProfile(m.cfg)
		m.cfg.Models[key] = mdl
	}

	if err := m.saveConfig(); err != nil {
		m.setError(err, "")
		return m, nil
	}

	// Expand it in the tree so the profile you just landed with (a fresh
	// default one, or an existing model's) is immediately visible instead
	// of hiding behind another "enter to expand" step.
	m.expandedModelKey = key
	m.rebuildRows()
	m.cursor = indexOfModelRow(m.rows, key)
	m.screen = screenMain
	m.clearError()
	return m, nil
}

// defaultProfile is the starting-point profile given to a newly (re-)added
// model, mirroring the New Profile form's own defaults.
func defaultProfile(cfg *config.Config) models.Profile {
	temp, topP, minP, topK := 0.6, 0.95, 0.0, 20
	return models.Profile{
		Name:      "default",
		Port:      suggestPort(cfg),
		CtxSize:   8192,
		Temp:      &temp,
		TopP:      &topP,
		TopK:      &topK,
		MinP:      &minP,
		FlashAttn: true,
		GPULayers: 999,
	}
}

// modelKeyByPath returns the key of the model whose Path matches path, or
// "" if none does.
func modelKeyByPath(existing map[string]models.Model, path string) string {
	for k, mdl := range existing {
		if mdl.Path == path {
			return k
		}
	}
	return ""
}

func modelNameFromPath(path string) string {
	base := filepath.Base(path)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

func modelKeyFromPath(path string) string {
	name := strings.ToLower(modelNameFromPath(path))
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '.', r == '-':
			b.WriteRune(r)
		case r == '_', r == ' ':
			b.WriteRune('-')
		}
	}
	key := b.String()
	if key == "" {
		key = "model"
	}
	return key
}

func uniqueModelKey(existing map[string]models.Model, base string) string {
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

func indexOfModelRow(rows []row, modelKey string) int {
	for i, r := range rows {
		if r.kind == rowModel && r.modelKey == modelKey {
			return i
		}
	}
	return 0
}
