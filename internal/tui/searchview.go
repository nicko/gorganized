package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/nicko/gorganized/internal/kb"
)

var (
	styleSearchOverlay = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("99")).
				Padding(1, 2)

	styleSearchHeader = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("15")).
				MarginBottom(1)

	styleSearchResult = lipgloss.NewStyle().
				PaddingLeft(1)

	styleSearchResultSelected = lipgloss.NewStyle().
					PaddingLeft(1).
					Foreground(lipgloss.Color("15")).
					Bold(true)

	styleSearchSnippet = lipgloss.NewStyle().
				Foreground(lipgloss.Color("243")).
				PaddingLeft(3)

	styleSearchHint = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243"))
)

// renderSearchOverlay renders the search overlay as a string.
func renderSearchOverlay(query string, results []kb.Result, cursor int) string {
	var sb strings.Builder

	sb.WriteString(styleSearchHeader.Render("Search knowledge base"))
	sb.WriteString("\n")
	sb.WriteString("  / ")
	sb.WriteString(query)
	sb.WriteString("█\n") // block cursor

	if len(results) == 0 && query != "" {
		sb.WriteString(styleSearchHint.Render("\n  No results."))
	}

	for i, r := range results {
		sb.WriteString("\n")
		line := fmt.Sprintf("%d. %s", r.TaskID, r.Title)
		if i == cursor {
			sb.WriteString(styleSearchResultSelected.Render("> " + line))
		} else {
			sb.WriteString(styleSearchResult.Render("  " + line))
		}
		if r.Snippet != "" {
			sb.WriteString("\n")
			sb.WriteString(styleSearchSnippet.Render(r.Snippet))
		}
	}

	sb.WriteString("\n\n")
	sb.WriteString(styleSearchHint.Render("  enter: go to task  ↑/↓: navigate  esc: close"))

	return styleSearchOverlay.Render(sb.String())
}
