package tui

import (
	"time"

	"github.com/sockheadrps/llmctl/internal/gpu"
	"github.com/sockheadrps/llmctl/internal/health"
	"github.com/sockheadrps/llmctl/internal/statusserver"
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
	screenStopConfirm
	screenProfileTemplate
	screenExportArgs
	screenNetworkSwitch
	screenNetworkPicker
	screenRPCServerAction
	screenRPCLayerSplit
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
	modeNetwork
	modeRPCServer
	modeOverview // displayed first in the tab bar; numeric value is last to avoid shifting existing constants
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

// ramMsg carries RSS MiB per PID for CPU-only model processes.
type ramMsg struct {
	byPID map[int]int64
}

type tickMsg time.Time
type scrollTickMsg time.Time

// remoteStatusMsg carries the result of polling a remote llmctl's status server.
type remoteStatusMsg struct {
	status *statusserver.Status
	err    error
}

const (
	scrollTickInterval = 500 * time.Millisecond
	scrollPauseTicks   = 2
)
