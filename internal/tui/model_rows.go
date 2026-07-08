package tui

import (
	"sort"
	"strings"

	"github.com/sockheadrps/llmctl/internal/config"
	"github.com/sockheadrps/llmctl/internal/models"
)

func (m *Model) rebuildRows() {
	m.rows = buildRowsFiltered(m.cfg, m.expandedModelKey, m.modelSearch)
	if m.cursor >= len(m.rows) {
		m.cursor = len(m.rows) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

// rebuildRecentRows reloads the recent-runs history and resolves each
// entry against the current config, silently dropping any whose model or
// profile no longer exists (e.g. deleted since it was last run).
func (m *Model) rebuildRecentRows() {
	recent, err := m.mgr.RecentRuns()
	if err != nil {
		m.setError(err, "")
		return
	}
	m.recentRuns = recent
	m.recentRows = buildRecentRows(m.cfg, recent)
	if m.recentCursor >= len(m.recentRows) {
		m.recentCursor = len(m.recentRows) - 1
	}
	if m.recentCursor < 0 {
		m.recentCursor = 0
	}
}

func buildRecentRows(cfg *config.Config, recent []models.RecentRun) []row {
	var rows []row
	for _, rr := range recent {
		mdl, ok := cfg.Models[rr.ModelKey]
		if !ok {
			continue
		}
		p, ok := mdl.Profiles[rr.ProfileKey]
		if !ok {
			continue
		}
		rows = append(rows, row{
			kind:       rowProfile,
			modelKey:   rr.ModelKey,
			profileKey: rr.ProfileKey,
			label:      mdl.Name + " / " + p.Name,
		})
	}
	return rows
}

// buildRows flattens the Models tree. Only expandedModelKey's profiles (and
// its "+ New Profile" row) are shown — every other model collapses to just
// its name, an accordion: entering a model expands it and collapses
// whichever was open before. A "+ Add Model" row always trails the list.
func buildRows(cfg *config.Config, expandedModelKey string) []row {
	return buildRowsFiltered(cfg, expandedModelKey, "")
}

func buildRowsFiltered(cfg *config.Config, expandedModelKey, filter string) []row {
	filter = strings.ToLower(strings.TrimSpace(filter))
	modelKeys := make([]string, 0, len(cfg.Models))
	for k := range cfg.Models {
		modelKeys = append(modelKeys, k)
	}
	sort.Strings(modelKeys)

	var rows []row
	for _, mk := range modelKeys {
		mdl := cfg.Models[mk]

		// A model with no saved profiles has nothing to run, so it just
		// clutters the tree — hide it. Re-adding it via "+ Add Model"
		// creates a default profile, which is what brings it back.
		if len(mdl.Profiles) == 0 {
			continue
		}

		profileKeys := make([]string, 0, len(mdl.Profiles))
		for pk := range mdl.Profiles {
			profileKeys = append(profileKeys, pk)
		}
		sort.Strings(profileKeys)

		matchingProfiles := make(map[string]bool, len(profileKeys))
		modelMatches := filter == "" || modelMatchesFilter(mk, mdl, filter)
		for _, pk := range profileKeys {
			p := mdl.Profiles[pk]
			if filter == "" || profileMatchesFilter(pk, p, filter) {
				matchingProfiles[pk] = true
			}
		}
		if filter != "" && !modelMatches && len(matchingProfiles) == 0 {
			continue
		}

		rows = append(rows, row{kind: rowModel, modelKey: mk, label: mdl.Name})

		if mk != expandedModelKey && filter == "" {
			continue
		}

		for _, pk := range profileKeys {
			if filter != "" && !modelMatches && !matchingProfiles[pk] {
				continue
			}
			rows = append(rows, row{
				kind:       rowProfile,
				modelKey:   mk,
				profileKey: pk,
				label:      mdl.Profiles[pk].Name,
			})
		}

		rows = append(rows, row{kind: rowAddProfile, modelKey: mk, label: "+ New Profile"})
	}

	rows = append(rows, row{kind: rowAddModel, label: "+ Add Model"})

	return rows
}

func modelMatchesFilter(key string, mdl models.Model, filter string) bool {
	return strings.Contains(strings.ToLower(key), filter) ||
		strings.Contains(strings.ToLower(mdl.Name), filter) ||
		strings.Contains(strings.ToLower(mdl.Path), filter) ||
		strings.Contains(strings.ToLower(mdl.HFRepo), filter) ||
		strings.Contains(strings.ToLower(mdl.Notes), filter)
}

func profileMatchesFilter(key string, p models.Profile, filter string) bool {
	return strings.Contains(strings.ToLower(key), filter) ||
		strings.Contains(strings.ToLower(p.Name), filter) ||
		strings.Contains(strings.ToLower(p.Notes), filter)
}

// visibleModelKeys returns the sorted keys of every model that appears in
// the tree (i.e. has at least one profile) — the same filter buildRows
// applies, exposed separately so cursor movement can browse models without
// needing the full row list expanded.
func visibleModelKeys(cfg *config.Config) []string {
	keys := make([]string, 0, len(cfg.Models))
	for k, mdl := range cfg.Models {
		if len(mdl.Profiles) == 0 {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// buildSettingsRows lists the Settings tab's menu. Status Server is only
// exposed for RPC server mode; clients publish to the remote server instead.
func (m Model) buildSettingsRows() []row {
	var rows []row
	for _, c := range settingsCategories {
		if c.id == "status_server" && (m.cfg == nil || !m.cfg.RPCEnabled || m.cfg.RPCMode != "server") {
			continue
		}
		rows = append(rows, row{kind: rowSettingsCategory, modelKey: c.id, label: c.label})
	}
	return rows
}

func runningContains(list []models.Running, target models.Running) bool {
	for _, r := range list {
		if r.ModelKey == target.ModelKey && r.ProfileKey == target.ProfileKey {
			return true
		}
	}
	return false
}
