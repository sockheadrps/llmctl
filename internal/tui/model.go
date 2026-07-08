// Package tui implements llmctl's interactive terminal UI: a list of
// Models and their Profiles on the left, and currently Running instances
// on the right, per the layout in plan.md. It also supports importing new
// Models from a GGUF directory and creating new Profiles interactively.
package tui

import (
	"fmt"
	"os"
	runtimeos "runtime"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/sockheadrps/llmctl/internal/build"
	"github.com/sockheadrps/llmctl/internal/config"
	"github.com/sockheadrps/llmctl/internal/gpu"
	"github.com/sockheadrps/llmctl/internal/health"
	"github.com/sockheadrps/llmctl/internal/models"
	"github.com/sockheadrps/llmctl/internal/process"
	"github.com/sockheadrps/llmctl/internal/runtime"
	"github.com/sockheadrps/llmctl/internal/statusserver"
	"github.com/sockheadrps/llmctl/internal/util"
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

	running          []models.Running
	runningCursor    int
	rpcServerState   runtime.RPCServerState
	rpcServerAlive   bool
	health           healthMsg
	pendingInstances map[string]bool // keys still loading after start; cleared on first StatusUp

	loadStartedAt map[string]time.Time     // when loading began, keyed by "modelKey/profileKey"
	loadWithRPC   map[string]bool          // whether RPC was enabled when load started
	loadDuration  map[string]time.Duration // set when load completes, persists while model is up
	loadHistory   loadTimeStore
	loadTimesPath string

	tokSamples map[string]tokSample // last decoded-count snapshot, for computing tok/s deltas
	tokRates   map[string]float64   // current tok/s while actively generating; absent when idle
	tokPeak    map[string]float64   // session-high tok/s per instance, for scaling the rate meter

	gpuAvailable bool // whether nvidia-smi was found at startup
	gpuName      string
	gpuUsage     gpu.Usage
	gpuByPID     map[int]int64

	statusServer          *statusserver.Server
	statusServerHost      string
	statusServerPort      int
	statusPublisher       *statusserver.Publisher
	statusPublisherAddr   string
	remoteStatus          *statusserver.Status
	discoveredRPCEndpoint string // derived from remote status poll: host:rpc_port
	rpcAddrCopied         bool   // true briefly after copying the status server address
	rpcIPCursor           int    // which LAN IP is selected on the RPC Server tab

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

	picker               pickerState
	form                 formState
	formExit             formExitState
	confirm              confirmState
	logs                 logsState
	settings             settingsState
	runningAction        runningActionState
	rpcServerActionState rpcServerActionState
	stopConfirm          stopConfirmState
	exportArgs           exportArgsState
	templatePicker       templatePickerState

	tokHistory map[string][]float64

	dividerDragging      bool
	leftWidthOverride    int // 0 = auto (avail*2/5); positive = user-dragged override
	rightDividerDragging bool
	rightSplitOverride   int // 0 = auto; positive = user-dragged running-box content height

	netSupported bool // false on non-Linux; tab is hidden entirely

	netStatus    netStatusMsg
	netSwitching bool
	netCursor    int
	netSwitch    netSwitchState
	netPicker    netPickerState

	netInternetConn string
	netRPCConn      string
	netIface        string

	width  int
	height int
}

// New builds the initial TUI model from a loaded config and runtime manager.
// cfgPath is where changes made in the TUI (new models/profiles) are persisted.
// Focus starts on the tab bar (Models tab active) rather than dropping
// straight into the tree.
func New(cfg *config.Config, cfgPath string, mgr *runtime.Manager, netInternetConn, netRPCConn, netIface string) Model {
	m := Model{
		cfg: cfg, cfgPath: cfgPath, mgr: mgr,
		health:           healthMsg{},
		pendingInstances: map[string]bool{},
		focus:            focusTabs,
		tokSamples:       map[string]tokSample{},
		tokRates:         map[string]float64{},
		tokPeak:          map[string]float64{},
		tokHistory:       map[string][]float64{},
		gpuAvailable:     gpu.Available(),
		netSupported:     runtimeos.GOOS == "linux",
		netInternetConn:  firstNonEmpty(cfg.NetworkInternetConn, netInternetConn),
		netRPCConn:       firstNonEmpty(cfg.NetworkRPCConn, netRPCConn),
		netIface:         firstNonEmpty(cfg.NetworkIface, netIface),
	}
	if m.gpuAvailable {
		if name, err := gpu.Name(); err == nil {
			m.gpuName = name
		}
	}
	m.statusPublisher = statusserver.NewPublisher(clientID(), clientName())
	if err := m.reconcileStatusServer(); err != nil {
		m.setError(fmt.Errorf("status server: %w", err), "")
	}
	m.reconcileStatusPublisher()
	m.loadStartedAt = map[string]time.Time{}
	m.loadWithRPC = map[string]bool{}
	m.loadDuration = map[string]time.Duration{}
	ltPath, _ := util.LoadTimesFile()
	m.loadTimesPath = ltPath
	m.loadHistory = loadLoadTimes(ltPath)
	m.rebuildRows()
	m.rebuildRecentRows()
	m.refreshRunning(false)
	m.pushStatusServer()
	return m
}

func (m Model) shouldRunStatusServer() bool {
	if m.cfg == nil {
		return false
	}
	return m.cfg.RPCEnabled && m.cfg.RPCMode == "server"
}

func (m Model) statusServerBindAddr() (string, int) {
	if m.cfg == nil {
		return "0.0.0.0", 11435
	}
	host := m.cfg.StatusServerHost
	if host == "" {
		host = "0.0.0.0"
	}
	port := m.cfg.StatusServerPort
	if port == 0 {
		port = 11435
	}
	return host, port
}

func (m *Model) reconcileStatusServer() error {
	if !m.shouldRunStatusServer() {
		if m.statusServer != nil {
			m.statusServer.Stop()
			m.statusServer = nil
			m.statusServerHost = ""
			m.statusServerPort = 0
		}
		return nil
	}

	host, port := m.statusServerBindAddr()
	if m.statusServer != nil && m.statusServerHost == host && m.statusServerPort == port {
		return nil
	}
	if m.statusServer != nil {
		m.statusServer.Stop()
		m.statusServer = nil
		m.statusServerHost = ""
		m.statusServerPort = 0
	}

	srv := statusserver.NewServer()
	if err := srv.Start(host, port); err != nil {
		return err
	}
	m.statusServer = srv
	m.statusServerHost = host
	m.statusServerPort = port
	m.pushStatusServer()
	return nil
}

func (m *Model) reconcileStatusPublisher() {
	if m.statusPublisher == nil {
		return
	}
	if m.cfg == nil || !m.cfg.RPCEnabled || m.cfg.RPCMode != "client" || strings.TrimSpace(m.cfg.RemoteStatusAddr) == "" {
		m.statusPublisher.Stop()
		m.statusPublisherAddr = ""
		return
	}
	addr := strings.TrimSpace(m.cfg.RemoteStatusAddr)
	if m.statusPublisherAddr == addr {
		return
	}
	m.statusPublisher.Start(addr)
	m.statusPublisherAddr = addr
	m.statusPublisher.Update(m.buildStatusSnapshot())
}

func clientID() string {
	if host, err := os.Hostname(); err == nil && strings.TrimSpace(host) != "" {
		return strings.TrimSpace(host)
	}
	return "llmctl-client"
}

func clientName() string {
	return clientID()
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

	if m.cfg.RPCEnabled && m.cfg.RPCMode == "server" {
		state, alive := m.mgr.RPCServerStatus()
		m.rpcServerState = state
		m.rpcServerAlive = alive
	}

	// Mark any instance that just appeared as pending so health checks keep
	// it in the "loading" state until it passes its first health check.
	for _, r := range m.running {
		key := r.ModelKey + "/" + r.ProfileKey
		if !runningContains(prev, r) {
			m.pendingInstances[key] = true
		}
	}
	// Clean up pending/health state for instances that are no longer running.
	for _, r := range prev {
		if !runningContains(m.running, r) {
			key := r.ModelKey + "/" + r.ProfileKey
			delete(m.pendingInstances, key)
			delete(m.health, key)
			delete(m.loadDuration, key)
			delete(m.loadStartedAt, key)
			delete(m.loadWithRPC, key)
		}
	}

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
			delete(m.tokPeak, key)
			delete(m.tokHistory, key)
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
				rate := float64(decoded-prev.decoded) / dt
				m.tokRates[key] = rate
				if rate > m.tokPeak[key] {
					m.tokPeak[key] = rate
					m.persistPeakIfRecord(key, rate)
				}
				hist := append(m.tokHistory[key], rate)
				const maxHistLen = 30
				if len(hist) > maxHistLen {
					hist = hist[len(hist)-maxHistLen:]
				}
				m.tokHistory[key] = hist
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

// persistPeakIfRecord saves rate to the profile's MaxTokPerSec when it beats
// the previously stored all-time peak, so the rate-meter scale survives restarts.
func (m *Model) persistPeakIfRecord(key string, rate float64) {
	parts := strings.SplitN(key, "/", 2)
	if len(parts) != 2 {
		return
	}
	modelKey, profileKey := parts[0], parts[1]
	mdl, ok := m.cfg.Models[modelKey]
	if !ok {
		return
	}
	p, ok := mdl.Profiles[profileKey]
	if !ok {
		return
	}
	if rate <= p.MaxTokPerSec {
		return
	}
	p.MaxTokPerSec = rate
	mdl.Profiles[profileKey] = p
	m.cfg.Models[modelKey] = mdl
	_ = m.saveConfig()
}

// pushStatusServer updates the local status server snapshot with current state.
// In RPC client mode it also publishes the snapshot to the remote status server.
func (m *Model) pushStatusServer() {
	st := m.buildStatusSnapshot()
	if m.statusServer != nil {
		m.statusServer.SetStatus(st)
	}
	if m.statusPublisher != nil {
		m.statusPublisher.Update(st)
	}
}

// buildStatusSnapshot assembles a statusserver.Status from current model state.
func (m Model) buildStatusSnapshot() statusserver.Status {
	running := make([]statusserver.RunningInfo, 0, len(m.running))
	for _, r := range m.running {
		key := r.ModelKey + "/" + r.ProfileKey
		h := m.health[key]
		if h == "" || m.pendingInstances[key] {
			h = health.StatusLoading
		}
		info := statusserver.RunningInfo{
			Model:   r.ModelName,
			Profile: r.ProfileName,
			Port:    r.Port,
			Health:  string(h),
			TokS:    m.tokRates[key],
		}
		if mdl, ok := m.cfg.Models[r.ModelKey]; ok && mdl.Path != "" {
			if stat, err := os.Stat(mdl.Path); err == nil {
				info.ModelSizeBytes = stat.Size()
			}
		}
		if mb, ok := m.gpuByPID[r.PID]; ok {
			info.VRAMMiB = mb
		}
		running = append(running, info)
	}

	st := statusserver.Status{
		Version: build.Version,
		Running: running,
	}

	if m.cfg.RPCEnabled && m.cfg.RPCMode == "server" && m.rpcServerAlive {
		rpcInfo := &statusserver.RPCInfo{
			Up:   true,
			Host: m.cfg.RPCServerHost,
			Port: m.cfg.RPCServerPort,
		}
		if mb, ok := m.gpuByPID[m.rpcServerState.PID]; ok {
			rpcInfo.VRAMMiB = mb
		}
		st.RPCServer = rpcInfo
	}

	if m.gpuAvailable && m.gpuUsage.TotalMiB > 0 {
		st.GPU = &statusserver.GPUInfo{
			Name:     m.gpuName,
			TotalMiB: m.gpuUsage.TotalMiB,
			UsedMiB:  m.gpuUsage.UsedMiB,
		}
	}

	return st
}

// pollRemoteStatusCmd polls the remote llmctl's status server at remoteStatusAddr
// ("host:port"). On success the caller derives the RPC endpoint from the
// response's rpc_server fields using the same host.
func pollRemoteStatusCmd(remoteStatusAddr string) tea.Cmd {
	return func() tea.Msg {
		st, err := statusserver.PollAddr(remoteStatusAddr)
		if err != nil {
			return remoteStatusMsg{err: err}
		}
		return remoteStatusMsg{status: &st}
	}
}

// return nil.
// backgroundChecks batches the periodic health/tok-rate/VRAM polls fired
// after a tick or a successful start.
func (m Model) backgroundChecks() tea.Cmd {
	cmds := []tea.Cmd{checkHealthCmd(m.running), checkSlotsCmd(m.running)}
	if m.networkTabVisible() {
		cmds = append(cmds, checkNetworkStatusCmd(m.netIface, m.netInternetConn, m.netRPCConn))
	}
	if m.gpuAvailable {
		cmds = append(cmds, checkVRAMCmd())
	}
	if m.cfg.RPCEnabled {
		switch m.cfg.RPCMode {
		case "server":
			cmds = append(cmds, checkRPCServerHealthCmd(m.mgr, m.cfg.RPCServerHost, m.cfg.RPCServerPort))
		case "client":
			if m.cfg.RemoteStatusAddr != "" {
				cmds = append(cmds, pollRemoteStatusCmd(m.cfg.RemoteStatusAddr))
			}
		}
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
			s := health.Check(r.Host, r.Port)
			if s == health.StatusUp && r.LogFile != "" && !health.LogReady(r.LogFile) {
				s = health.StatusLoading
			}
			result[r.ModelKey+"/"+r.ProfileKey] = s
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
			slots, err := health.Slots(r.Host, r.Port)
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

// checkRPCServerHealthCmd checks the ggml-rpc-server health.
// PID is the primary signal: if the process is alive, it's up — a TCP probe
// would fail while the server is busy handling an existing RPC connection
// (e.g. a model loading on the remote machine). The TCP probe is only used
// as a fallback to detect an externally-started server with no state file.
func checkRPCServerHealthCmd(mgr *runtime.Manager, host string, port int) tea.Cmd {
	return func() tea.Msg {
		state, running := mgr.RPCServerStatus()
		if running {
			_ = state
			return healthMsg{"rpc-server": health.StatusUp}
		}
		// PID dead or no state file.
		if mgr.HasRPCStateFile() {
			return healthMsg{"rpc-server": health.StatusDown}
		}
		// No state file — check if something external is on the port.
		if health.ProbeRPCPort(host, port) {
			return healthMsg{"rpc-server": health.StatusUp}
		}
		return healthMsg{"rpc-server": health.StatusNotStarted}
	}
}
