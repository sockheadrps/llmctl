package tui

import (
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/sockheadrps/llmctl/internal/runtime"
)

// logsState backs the full-screen log viewer.
type logsState struct {
	label  string
	path   string
	lines  []string
	offset int
	err    error
}

// logsVisibleHeight is how many log lines fit in the viewer box, leaving
// room for the title, path line, and help footer.
func logsVisibleHeight(termHeight int) int {
	h := termHeight - 8
	if h < 10 {
		h = 10
	}
	return h
}

// openLogsForCurrent opens the log viewer for whatever's most relevant
// right now: the log behind the current error if there is one, otherwise
// whichever profile/instance is currently selected.
func (m Model) openLogsForCurrent() (tea.Model, tea.Cmd) {
	if m.errLogPath != "" {
		return m.openLogs(m.errLogPath, "error")
	}

	if m.leftMode == modeRPCServer {
		logPath := m.rpcServerState.LogFile
		if logPath == "" {
			var err error
			logPath, err = runtime.RPCServerLogPath()
			if err != nil {
				m.setError(err, "")
				return m, nil
			}
		}
		return m.openLogs(logPath, "RPC Server")
	}

	if m.focus == focusRunning {
		if m.runningCursor < 0 || m.runningCursor >= len(m.running) {
			return m, nil
		}
		r := m.running[m.runningCursor]
		return m.openLogs(r.LogFile, r.Label())
	}

	r, ok := m.currentRow()
	if !ok || (r.kind != rowProfile && r.kind != rowRunning) {
		return m, nil
	}
	path, err := runtime.LogPath(r.modelKey, r.profileKey)
	if err != nil {
		m.setError(err, "")
		return m, nil
	}
	return m.openLogs(path, r.label)
}

// openLogs reads path and switches to the log-viewer screen, scrolled to
// the end — the most recent output, and where a crash's error would be.
func (m Model) openLogs(path, label string) (tea.Model, tea.Cmd) {
	state := logsState{label: label, path: path}

	data, err := os.ReadFile(path)
	if err != nil {
		state.err = err
	} else {
		state.lines = logDisplayLines(string(data), m.width)
		state.offset = max(0, len(state.lines)-logsVisibleHeight(m.height))
	}

	m.logs = state
	m.screen = screenLogs
	return m, nil
}

// logDisplayLines converts raw log file content into display lines — one
// element per terminal row — by applying the same word-wrap used in the
// Running tab preview. This ensures the pane Height constraint (in terminal
// rows) matches the slice length exactly, avoiding content clipping.
func logDisplayLines(raw string, termWidth int) []string {
	boxWidth := termWidth - 4
	if boxWidth < 40 {
		boxWidth = 40
	}
	lines := wrappedLogPreviewLines(strings.TrimRight(raw, "\n"), boxWidth)
	if len(lines) == 0 {
		// fall back to raw split so empty/short logs still display
		return strings.Split(strings.TrimRight(raw, "\n"), "\n")
	}
	return lines
}

// refreshLogs re-reads the log file in place. If the viewer was scrolled to
// the bottom it stays there; otherwise the offset is preserved.
func (m *Model) refreshLogs() {
	data, err := os.ReadFile(m.logs.path)
	if err != nil {
		m.logs.err = err
		return
	}
	lines := logDisplayLines(string(data), m.width)
	visible := logsVisibleHeight(m.height)
	atBottom := len(m.logs.lines) == 0 || m.logs.offset >= max(0, len(m.logs.lines)-visible)
	m.logs.lines = lines
	if atBottom {
		m.logs.offset = max(0, len(lines)-visible)
	}
}

func (m Model) updateLogs(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	visible := logsVisibleHeight(m.height)
	maxOffset := max(0, len(m.logs.lines)-visible)

	switch msg.String() {
	case "esc", "q", "e":
		m.screen = screenMain
		return m, nil

	case "up", "k":
		m.logs.offset = max(0, m.logs.offset-1)

	case "down", "j":
		m.logs.offset = min(maxOffset, m.logs.offset+1)

	case "pgup":
		m.logs.offset = max(0, m.logs.offset-visible)

	case "pgdown":
		m.logs.offset = min(maxOffset, m.logs.offset+visible)

	case "g":
		m.logs.offset = 0

	case "G":
		m.logs.offset = maxOffset
	}
	return m, nil
}
