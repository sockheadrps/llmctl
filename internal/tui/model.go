// Package tui implements llmctl's interactive terminal UI: a list of
// Models and their Profiles on the left, and currently Running instances
// on the right, per the layout in plan.md. It also supports importing new
// Models from a GGUF directory and creating new Profiles interactively.
package tui

import (
	"fmt"
	"os"
	runtimeos "runtime"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/sockheadrps/llmctl/internal/config"
	"github.com/sockheadrps/llmctl/internal/controller"
	"github.com/sockheadrps/llmctl/internal/gpu"
	"github.com/sockheadrps/llmctl/internal/models"
	"github.com/sockheadrps/llmctl/internal/statusserver"
	"github.com/sockheadrps/llmctl/internal/util"
)

// Model is the root bubbletea model for the main screen.
type Model struct {
	cfg     *config.Config
	cfgPath string
	ctrl    *controller.Controller

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
	rpcServerState   controller.RPCServerState
	rpcServerAlive   bool
	health           healthMsg
	pendingInstances map[string]bool // keys still loading after start; cleared on first StatusUp

	loadStartedAt map[string]time.Time     // when loading began, keyed by "modelKey/profileKey"
	loadWithRPC   map[string]bool          // whether RPC was enabled when load started
	loadDuration  map[string]time.Duration // set when load completes, persists while model is up
	loadHistory   loadTimeStore
	loadTimesPath string

	tokRateHistory tokRateStore // persisted per-session tok/s averages by "modelKey/profileKey"
	tokRatesPath   string

	tokSamples map[string]tokSample // last decoded-count snapshot, for computing tok/s deltas
	tokRates   map[string]float64   // current tok/s while actively generating; absent when idle
	tokPeak    map[string]float64   // session-high tok/s per instance, for scaling the rate meter

	gpuAvailable bool // whether nvidia-smi was found at startup
	gpuName      string
	gpuUsage     gpu.Usage
	gpuDevices   []gpu.DeviceUsage
	ramByPID     map[int]int64 // RSS MiB for CPU-only model processes

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
	detailsHovered       bool // mouse is over the details pane; suppresses auto-scroll
	detailsManualScroll  bool // user scrolled with wheel; stays suppressed until row changes

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

	modelLoadCache map[string]cachedModelLoad

	dividerDragging      bool
	leftWidthOverride    int // 0 = auto (avail*2/5); positive = user-dragged override
	rightDividerDragging bool
	rightSplitOverride   int // 0 = auto; positive = user-dragged running-box content height

	overviewSepDragging bool
	overviewSepX        int // 0 = auto; positive = user-dragged overview column separator X
	backgroundPollUntil time.Time

	modelSubTabFocused bool // true when cursor is on the Models/Recents sub-tab header row

	overviewCopied string // modelKey/profileKey briefly set after copying from the Overview tab
	gpuNameScroll  int    // ever-incrementing tick counter for horizontal GPU name scroll

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
func New(cfg *config.Config, cfgPath string, ctrl *controller.Controller, netInternetConn, netRPCConn, netIface string) Model {
	m := Model{
		cfg: cfg, cfgPath: cfgPath, ctrl: ctrl,
		health:           healthMsg{},
		pendingInstances: map[string]bool{},
		focus:            focusTabs,
		leftMode:         modeOverview,
		tokSamples:       map[string]tokSample{},
		tokRates:         map[string]float64{},
		tokPeak:          map[string]float64{},
		tokHistory:       map[string][]float64{},
		modelLoadCache:   map[string]cachedModelLoad{},
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
	m.statusPublisher = m.ctrl.NewPublisher(clientID(), clientName())
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
	trPath, _ := util.TokRatesFile()
	m.tokRatesPath = trPath
	m.tokRateHistory = loadTokRates(trPath)
	m.rebuildRows()
	m.rebuildRecentRows()
	m.refreshRunning(false)
	m.pushStatusServer()
	return m
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

	running, err := m.ctrl.ListRunning()
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
				m.setError(fmt.Errorf("%s exited unexpectedly:\n%s", old.Label(), tailOrReason(old.LogFile, m.ctrl)), old.LogFile)
				break
			}
		}
	}

	m.running = running

	if m.cfg.RPCEnabled && m.cfg.RPCMode == "server" {
		state, alive := m.ctrl.RPCServerStatus()
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
			// Persist this session's rolling-window average before discarding it.
			if hist := m.tokHistory[key]; len(hist) >= 3 {
				var sum float64
				for _, v := range hist {
					sum += v
				}
				m.tokRateHistory.record(key, sum/float64(len(hist)))
				_ = saveTokRates(m.tokRatesPath, m.tokRateHistory)
			}
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
			if dt := now.Sub(prev.at).Seconds(); dt >= 0.25 {
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

func tailOrReason(logPath string, ctrl *controller.Controller) string {
	tail, err := ctrl.TailLog(logPath, 8)
	if err != nil || tail == "" {
		return "(no log output — check " + logPath + ")"
	}
	return tail
}

// Init starts the periodic refresh tick.
func (m Model) Init() tea.Cmd {
	return tea.Batch(tickCmd(), scrollTickCmd())
}

func (m Model) shouldContinueScrollTick() bool {
	switch m.screen {
	case screenNewProfile, screenMain:
		return true
	default:
		return false
	}
}

func (m *Model) modelLoadSlices(logPath string) ([]controller.GPUDeviceInfo, error) {
	if strings.TrimSpace(logPath) == "" {
		return nil, nil
	}
	info, err := os.Stat(logPath)
	if err != nil {
		return nil, err
	}
	if cached, ok := m.modelLoadCache[logPath]; ok {
		// Once we have a non-empty slice summary, the load split is stable for
		// the lifetime of this log file. Appended runtime tokens/events should not
		// trigger a full rescan on every render/tick.
		if cached.complete && info.Size() >= cached.size {
			out := make([]statusserver.GPUDeviceInfo, len(cached.slices))
			copy(out, cached.slices)
			return out, nil
		}
		if !cached.complete && cached.modTime.Equal(info.ModTime()) && cached.size == info.Size() {
			if len(cached.slices) == 0 {
				return nil, nil
			}
			out := make([]statusserver.GPUDeviceInfo, len(cached.slices))
			copy(out, cached.slices)
			return out, nil
		}
	}

	slices, _, err := m.ctrl.ParseModelLoadSlices(logPath)
	if err != nil {
		return nil, err
	}
	if m.modelLoadCache == nil {
		m.modelLoadCache = make(map[string]cachedModelLoad)
	}
	if len(slices) == 0 {
		m.modelLoadCache[logPath] = cachedModelLoad{
			modTime:  info.ModTime(),
			size:     info.Size(),
			slices:   nil,
			complete: false,
		}
		return nil, nil
	}
	cached := cachedModelLoad{
		modTime:  info.ModTime(),
		size:     info.Size(),
		slices:   append([]statusserver.GPUDeviceInfo(nil), slices...),
		complete: true,
	}
	m.modelLoadCache[logPath] = cached
	out := make([]statusserver.GPUDeviceInfo, len(cached.slices))
	copy(out, cached.slices)
	return out, nil
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
