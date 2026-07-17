package logs

import (
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

// State backs the full-screen log viewer.
type State struct {
	Label  string
	Path   string
	Lines  []string
	Offset int
	Err    error
}

// VisibleHeight is how many log lines fit in the viewer box, leaving room for
// the title, path line, and help footer.
func VisibleHeight(termHeight int) int {
	h := termHeight - 8
	if h < 10 {
		h = 10
	}
	return h
}

// Open reads path and initializes the log viewer state, scrolled to the end.
func Open(path, label string, termWidth, termHeight int) State {
	state := State{Label: label, Path: path}

	data, err := os.ReadFile(path)
	if err != nil {
		state.Err = err
		return state
	}

	state.Lines = PreviewLines(string(data), termWidth)
	state.Offset = max(0, len(state.Lines)-VisibleHeight(termHeight))
	return state
}

// Refresh re-reads the log file in place. If the viewer was scrolled to the
// bottom it stays there; otherwise the offset is preserved.
func (s *State) Refresh(termWidth, termHeight int) {
	data, err := os.ReadFile(s.Path)
	if err != nil {
		s.Err = err
		return
	}
	lines := PreviewLines(string(data), termWidth)
	visible := VisibleHeight(termHeight)
	atBottom := len(s.Lines) == 0 || s.Offset >= max(0, len(s.Lines)-visible)
	s.Lines = lines
	s.Err = nil
	if atBottom {
		s.Offset = max(0, len(lines)-visible)
	}
}

// Update handles log-viewer keys. It returns true when the caller should exit
// back to the main screen.
func (s *State) Update(msg tea.KeyMsg, termHeight int) bool {
	visible := VisibleHeight(termHeight)
	maxOffset := max(0, len(s.Lines)-visible)

	switch msg.String() {
	case "esc", "q", "e":
		return true
	case "up", "k":
		s.Offset = max(0, s.Offset-1)
	case "down", "j":
		s.Offset = min(maxOffset, s.Offset+1)
	case "pgup":
		s.Offset = max(0, s.Offset-visible)
	case "pgdown":
		s.Offset = min(maxOffset, s.Offset+visible)
	case "g":
		s.Offset = 0
	case "G":
		s.Offset = maxOffset
	}
	return false
}
