package tui

import "github.com/charmbracelet/bubbles/key"

// keyMap defines the keybindings for the main TUI screen.
type keyMap struct {
	Up     key.Binding
	Down   key.Binding
	Left   key.Binding
	Right  key.Binding
	Run    key.Binding
	Stop   key.Binding
	Delete key.Binding
	Logs   key.Binding
	Quit   key.Binding
}

var keys = keyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
	Left: key.NewBinding(
		key.WithKeys("left", "h"),
		key.WithHelp("←/h", "models pane"),
	),
	Right: key.NewBinding(
		key.WithKeys("right", "l"),
		key.WithHelp("→/l", "running pane"),
	),
	Run: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "run profile"),
	),
	Stop: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "stop"),
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
