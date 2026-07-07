package tui

import "github.com/charmbracelet/bubbles/key"

// keyMap defines the keybindings for the main TUI screen.
type keyMap struct {
	Up     key.Binding
	Down   key.Binding
	Left   key.Binding
	Right  key.Binding
	Run    key.Binding
	Copy   key.Binding
	Delete key.Binding
	Logs   key.Binding
	Quit   key.Binding
}

var keys = keyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k", "w"),
		key.WithHelp("↑/k/w", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j", "s"),
		key.WithHelp("↓/j/s", "down"),
	),
	Left: key.NewBinding(
		key.WithKeys("left", "h", "a"),
		key.WithHelp("←/h/a", "models pane"),
	),
	Right: key.NewBinding(
		key.WithKeys("right", "l", "d"),
		key.WithHelp("→/l/d", "running pane"),
	),
	Run: key.NewBinding(
		key.WithKeys("enter", " "),
		key.WithHelp("enter/space", "run/select"),
	),
	Copy: key.NewBinding(
		key.WithKeys("c"),
		key.WithHelp("c", "copy profile"),
	),
	Delete: key.NewBinding(
		key.WithKeys("delete"),
		key.WithHelp("del", "delete profile (press twice)"),
	),
	Logs: key.NewBinding(
		key.WithKeys("e"),
		key.WithHelp("e", "view logs"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
}
