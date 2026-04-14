package tui

import "github.com/charmbracelet/lipgloss"

var (
	styleHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("243")).
			MarginTop(1)

	styleActiveHeader = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("10")) // green

	styleInactiveHeader = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("11")) // yellow

	styleTodoHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("12")) // blue

	styleDoneHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("243")) // grey

	styleTask = lipgloss.NewStyle().
			PaddingLeft(2)

	styleCursor = lipgloss.NewStyle().
			PaddingLeft(2).
			Foreground(lipgloss.Color("15")).
			Bold(true)

	styleStatusBar = lipgloss.NewStyle().
			Background(lipgloss.Color("236")).
			Foreground(lipgloss.Color("252")).
			PaddingLeft(1).
			PaddingRight(1)
)
