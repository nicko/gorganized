package tui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Up   key.Binding
	Down key.Binding
	Quit key.Binding
}

var keys = keyMap{
	Up: key.NewBinding(
		key.WithKeys("k", "up"),
		key.WithHelp("k/↑", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("j", "down"),
		key.WithHelp("j/↓", "down"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
}
