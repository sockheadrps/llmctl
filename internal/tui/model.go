// Package tui implements llmctl's interactive terminal UI: a list of
// Models and their Profiles on the left, and currently Running instances
// on the right, per the layout in plan.md. It also supports importing new
// Models from a GGUF directory and creating new Profiles interactively.
package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/sockheadrps/llmctl/internal/config"
	"github.com/sockheadrps/llmctl/internal/gpu"
	"github.com/sockheadrps/llmctl/internal/health"
	"github.com/sockheadrps/llmctl/internal/models"
	"github.com/sockheadrps/llmctl/internal/process"
	"github.com/sockheadrps/llmctl/internal/runtime"
)

// screen selects which full-screen view is active.
type screen int

const (
	screenMain screen = iota
	screenPickModel
	screenNewProfile
	screenFormExitConfirm
	screenConfirmProfile
	screenLogs
	screenRunningAction
)

// paneFocus selects which pane arrow keys/s apply to on the main screen.
// It's a hierarchy: the Models/Recents/Settings tab bar sits above the left
// pane's content, which sits to the left of the Running pane. When the
// Settings tab is active, focusSettingsContent is where Enter on a category
// drops you — the Details pane becomes interactive (add/edit/delete rows)
// rather than just a read-only preview.
type paneFocus int

const (
	focusTabs paneFocus = iota
	focusLeft
	focusSettingsContent
	focusRunning
)

// leftMode selects what the left pane shows: the Models/Profiles tree, the
// rolling history of recently run profiles, the Settings menu, or the list
// of currently running instances. Order matters — Left/Right at the tab bar
// step through these in sequence.
type leftMode int

const (
	modeModels leftMode = iota
	modeRecents
	modeSettings
	modeRunning
)

// rowKind identifies what a flattened tree row represents.
type rowKind int

const (
	rowModel rowKind = iota
	rowProfile
	rowAddProfile
	rowAddModel
	rowSettingsCategory
	rowRunning
)

// row is a single flattened line in the left-hand models/profiles tree.
type row struct {
	kind       rowKind
	modelKey   string
	profileKey string
	label      string
}

// healthMsg carries the result of a periodic health sweep over running instances.
type healthMsg map[string]health.Status

// tokSample is a snapshot of a slot's cumulative decoded-token count at a
// point in time, used to compute tokens/sec from the delta between ticks.
type tokSample struct {
	decoded int
	at      time.Time
}

// slotsMsg carries each running instance's current cumulative decoded-token
// count, keyed by "modelKey/profileKey", for instances actively generating
// right now. Instances with nothing in flight are simply absent.
type slotsMsg map[string]int

// vramMsg carries a GPU VRAM snapshot: aggregate usage plus a per-PID
// breakdown for matching against running instances.
type vramMsg struct {
	usage gpu.Usage
	byPID map[int]int64
}

type tickMsg time.Time
type scrollTickMsg time.Time

const (
	scrollTickInterval = 500 * time.Millisecond
	scrollPauseTicks   = 2
)

// Model is the root bubbletea model for the main screen.
type Model struct {
	cfg     *config.Config
	cfgPath string
	mgr     *runtime.Manager

	screen screen

	focus    paneFocus
	leftMode leftMode

	rows              []row
	cursor            int
	expandedModelKey  string // "" = tree fully collapsed; else the one model showing its profiles
	modelProfilesMode bool   // true after entering the expanded model's profile rows
	modelSearch       string
	searchEditing     bool

	recentRuns   []models.RecentRun
	recentRows   []row
	recentCursor int

	settingsCursor int

	running       []models.Running
	runningCursor int
	health        healthMsg

	tokSamples map[string]tokSample // last decoded-count snapshot, for computing tok/s deltas
	tokRates   map[string]float64   // current tok/s while actively generating; absent when idle

	gpuAvailable bool // whether nvidia-smi was found at startup
	gpuName      string
	gpuUsage     gpu.Usage
	gpuByPID     map[int]int64

	err        error
	errLogPath string // log file behind the current error, if any; "" means none to view

	starting      bool
	startingLabel string

	stopping      bool
	stoppingLabel string

	pendingDeleteModel   string
	pendingDeleteProfile string
	detailsScroll        int
	detailsDir           int
	detailsPause         int

	picker        pickerState
	form          formState
	formExit      formExitState
	confirm       confirmState
	logs          logsState
	settings      settingsState
	runningAction runningActionState

	width  int
	height int
}

// New builds the initial TUI model from a loaded config and runtime manager.
// cfgPath is where changes made in the TUI (new models/profiles) are persisted.
// Focus starts on the tab bar (Models tab active) rather than dropping
// straight into the tree.
func New(cfg *config.Config, cfgPath string, mgr *runtime.Manager) Model {
	m := Model{
		cfg: cfg, cfgPath: cfgPath, mgr: mgr,
		health:       healthMsg{},
		focus:        focusTabs,
		tokSamples:   map[string]tokSample{},
		tokRates:     map[string]float64{},
		gpuAvailable: gpu.Available(),
	}
	if m.gpuAvailable {
		if name, err := gpu.Name(); err == nil {
			m.gpuName = name
		}
	}
	m.rebuildRows()
	m.rebuildRecentRows()
	m.refreshRunning(false)
	return m
}

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

// buildSettingsRows lists the Settings tab's menu, derived from
// settingsCategories (the same list backs the dedicated Settings screen's
// left container) so the two stay in sync automatically.
func buildSettingsRows() []row {
	rows := make([]row, len(settingsCategories))
	for i, c := range settingsCategories {
		rows[i] = row{kind: rowSettingsCategory, modelKey: c.id, label: c.label}
	}
	return rows
}

// setError records an error and, when it's tied to a specific run, the log
// file behind it — so the log viewer ('e') has something to open.
func (m *Model) setError(err error, logPath string) {
	m.err = err
	m.errLogPath = logPath
}

// clearError clears the current error and its associated log path.
func (m *Model) clearError() {
	m.err = nil
	m.errLogPath = ""
}

// refreshRunning reloads the tracked running instances. When detectCrashes
// is true (the periodic tick path only — never after an explicit user
// start/stop, which legitimately changes the set), any instance present
// before this call but missing now is treated as an unexpected exit and
// its log tail is surfaced as an error.
func (m *Model) refreshRunning(detectCrashes bool) {
	prev := m.running

	running, err := m.mgr.List()
	if err != nil {
		m.setError(err, "")
		return
	}

	// Only overwrite m.err when there's something new to report — a
	// detected crash — never blanket-clear it. This runs on every 2s
	// tick, so unconditionally resetting m.err here would wipe out
	// errors from other actions (a failed start, a blocked delete, …)
	// moments after they're shown.
	if detectCrashes {
		for _, old := range prev {
			if !runningContains(running, old) {
				m.setError(fmt.Errorf("%s exited unexpectedly:\n%s", old.Label(), tailOrReason(old.LogFile)), old.LogFile)
				break
			}
		}
	}

	m.running = running

	if m.runningCursor >= len(m.running) {
		m.runningCursor = len(m.running) - 1
	}
	if len(m.running) == 0 {
		m.runningCursor = 0
		// Only reclaim focus if it was actually on the now-empty Running
		// pane — don't yank it away from the tab bar or left content.
		if m.focus == focusRunning {
			m.focus = focusLeft
		}
	}

	// Drop tok/s tracking for anything no longer running, so a stopped
	// instance's last rate doesn't linger, and a new instance that happens
	// to reuse the same model+profile key doesn't inherit a stale sample.
	live := make(map[string]bool, len(m.running))
	for _, r := range m.running {
		live[r.ModelKey+"/"+r.ProfileKey] = true
	}
	for key := range m.tokSamples {
		if !live[key] {
			delete(m.tokSamples, key)
			delete(m.tokRates, key)
		}
	}
}

// applyTokSamples updates tok/s for every key present in a slotsMsg by
// diffing against the previous sample, and clears the rate for any
// instance that's no longer actively generating (absent from msg).
func (m *Model) applyTokSamples(msg slotsMsg) {
	now := time.Now()

	for key, decoded := range msg {
		if prev, ok := m.tokSamples[key]; ok && decoded >= prev.decoded {
			if dt := now.Sub(prev.at).Seconds(); dt > 0 {
				m.tokRates[key] = float64(decoded-prev.decoded) / dt
			}
		}
		m.tokSamples[key] = tokSample{decoded: decoded, at: now}
	}

	// Anything not in msg isn't generating right now (checkSlotsCmd only
	// reports processing slots) — drop its rate and sample so a fresh
	// baseline is taken next time it starts generating, rather than
	// diffing across an idle gap.
	for key := range m.tokSamples {
		if _, ok := msg[key]; !ok {
			delete(m.tokRates, key)
			delete(m.tokSamples, key)
		}
	}
}

// backgroundChecks batches the periodic health/tok-rate/VRAM polls fired
// after a tick or a successful start.
func (m Model) backgroundChecks() tea.Cmd {
	cmds := []tea.Cmd{checkHealthCmd(m.running), checkSlotsCmd(m.running)}
	if m.gpuAvailable {
		cmds = append(cmds, checkVRAMCmd())
	}
	return tea.Batch(cmds...)
}

func runningContains(list []models.Running, target models.Running) bool {
	for _, r := range list {
		if r.ModelKey == target.ModelKey && r.ProfileKey == target.ProfileKey {
			return true
		}
	}
	return false
}

// findRunning looks up a running instance by model/profile key, e.g. to
// resolve a rowRunning row (which only carries the keys) back to its full
// Running record (port, PID, log file, ...).
func (m Model) findRunning(modelKey, profileKey string) (models.Running, bool) {
	for _, r := range m.running {
		if r.ModelKey == modelKey && r.ProfileKey == profileKey {
			return r, true
		}
	}
	return models.Running{}, false
}

func tailOrReason(logPath string) string {
	tail, err := process.TailLog(logPath, 8)
	if err != nil || tail == "" {
		return "(no log output — check " + logPath + ")"
	}
	return tail
}

// Init starts the periodic refresh tick.
func (m Model) Init() tea.Cmd {
	return tea.Batch(tickCmd(), scrollTickCmd())
}

func tickCmd() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func scrollTickCmd() tea.Cmd {
	return tea.Tick(scrollTickInterval, func(t time.Time) tea.Msg {
		return scrollTickMsg(t)
	})
}

func checkHealthCmd(running []models.Running) tea.Cmd {
	return func() tea.Msg {
		result := make(healthMsg, len(running))
		for _, r := range running {
			result[r.ModelKey+"/"+r.ProfileKey] = health.Check(r.Port)
		}
		return result
	}
}

// checkSlotsCmd polls /slots for each running instance and reports the
// cumulative decoded-token count for any slot currently generating. The
// rate itself is computed in Update, which has the previous sample to
// diff against.
func checkSlotsCmd(running []models.Running) tea.Cmd {
	return func() tea.Msg {
		result := make(slotsMsg, len(running))
		for _, r := range running {
			slots, err := health.Slots(r.Port)
			if err != nil {
				continue
			}
			decoded := 0
			processing := false
			for _, s := range slots {
				if s.IsProcessing {
					processing = true
					decoded += s.Decoded()
				}
			}
			if processing {
				result[r.ModelKey+"/"+r.ProfileKey] = decoded
			}
		}
		return result
	}
}

// checkVRAMCmd polls nvidia-smi for aggregate and per-PID VRAM usage. Only
// call this when gpuAvailable — it shells out, so there's no point retrying
// every tick on a machine without nvidia-smi.
func checkVRAMCmd() tea.Cmd {
	return func() tea.Msg {
		usage, err := gpu.Total()
		if err != nil {
			return vramMsg{}
		}
		byPID, err := gpu.ByPID()
		if err != nil {
			byPID = nil
		}
		return vramMsg{usage: usage, byPID: byPID}
	}
}
