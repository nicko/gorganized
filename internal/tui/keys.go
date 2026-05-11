package tui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Up       key.Binding
	Down     key.Binding
	Quit     key.Binding
	Enter    key.Binding
	Done     key.Binding
	Add      key.Binding
	Note     key.Binding
	Tab      key.Binding
	Search   key.Binding
	Preview  key.Binding
	MoveUp   key.Binding
	MoveDown key.Binding
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
	Enter: key.NewBinding(
		key.WithKeys(" "),
		key.WithHelp("space", "cycle state"),
	),
	Done: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "mark done"),
	),
	Add: key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "add task"),
	),
	Note: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "edit notes"),
	),
	Tab: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "switch view"),
	),
	Search: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "search KB"),
	),
	Preview: key.NewBinding(
		key.WithKeys("p"),
		key.WithHelp("p", "toggle preview"),
	),
	MoveUp: key.NewBinding(
		key.WithKeys("shift+up"),
		key.WithHelp("shift+↑", "move task up"),
	),
	MoveDown: key.NewBinding(
		key.WithKeys("shift+down"),
		key.WithHelp("shift+↓", "move task down"),
	),
}
