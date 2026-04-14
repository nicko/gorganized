package tui

import (
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/lipgloss"
)

// newNoteEditor creates a configured textarea for note editing.
func newNoteEditor() textarea.Model {
	ta := textarea.New()
	ta.Placeholder = "Add notes here…"
	ta.ShowLineNumbers = false
	ta.SetWidth(80)
	ta.SetHeight(20)
	return ta
}

var styleNoteOverlay = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color("62")).
	Padding(1, 2)

var styleNoteHeader = lipgloss.NewStyle().
	Bold(true).
	Foreground(lipgloss.Color("15")).
	MarginBottom(1)
