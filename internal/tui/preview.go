package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/nicko/gorganized/internal/model"
)

var (
	stylePreviewTitle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("15"))

	stylePreviewMeta = lipgloss.NewStyle().
				Foreground(lipgloss.Color("243"))

	stylePreviewDivider = lipgloss.NewStyle().
				Foreground(lipgloss.Color("240"))

	stylePreviewPlaceholder = lipgloss.NewStyle().
					Foreground(lipgloss.Color("243")).
					Italic(true)

	stylePreviewStateActive   = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
	stylePreviewStateInactive = lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Bold(true)
	stylePreviewStateTodo     = lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Bold(true)
	stylePreviewStateDone     = lipgloss.NewStyle().Foreground(lipgloss.Color("243")).Bold(true)
)

// renderPreview renders the right-hand preview panel for the given task.
// pomodoroCount is the number of completed work intervals (only meaningful when task is active).
// width is the usable column count; height is the terminal height minus the status bar.
func renderPreview(task model.Task, pomodoroCount int, width, height int) string {
	if task.ID == 0 {
		return stylePreviewPlaceholder.Render("  No task selected.")
	}

	var sb strings.Builder

	// Title
	sb.WriteString(stylePreviewTitle.Render(task.Title))
	sb.WriteString("\n")

	// State (coloured)
	stateStr := stateStyle(task.State).Render(string(task.State))
	sb.WriteString(stylePreviewMeta.Render("State: ") + stateStr)
	sb.WriteString("\n")

	// Time spent
	totalSecs := task.TimeSpent
	if task.State == model.StateActive && task.ActiveSince != nil {
		totalSecs += int(time.Since(*task.ActiveSince).Seconds())
	}
	sb.WriteString(stylePreviewMeta.Render("Time:  "+formatDuration(totalSecs)))
	sb.WriteString("\n")

	// Created
	sb.WriteString(stylePreviewMeta.Render("Created: "+task.CreatedAt.Local().Format("2006-01-02 15:04")))
	sb.WriteString("\n")

	// Pomodoros — only shown when task is active
	if task.State == model.StateActive {
		sb.WriteString(stylePreviewMeta.Render(fmt.Sprintf("Pomodoros: %d done", pomodoroCount)))
		sb.WriteString("\n")
	}

	// Divider
	sb.WriteString(stylePreviewDivider.Render(strings.Repeat("─", width-2)))
	sb.WriteString("\n")

	// Notes
	notes := strings.TrimSpace(task.Notes)
	if notes == "" {
		sb.WriteString(stylePreviewPlaceholder.Render("No notes yet. Press n to add."))
	} else {
		rendered := renderMarkdown(notes, width)
		sb.WriteString(rendered)
	}

	// Trim to height so the panel doesn't scroll.
	lines := strings.Split(sb.String(), "\n")
	maxLines := height - 1
	if maxLines > 0 && len(lines) > maxLines {
		lines = lines[:maxLines]
	}
	return strings.Join(lines, "\n")
}

// stateStyle returns the coloured lipgloss style for a state label.
func stateStyle(s model.State) lipgloss.Style {
	switch s {
	case model.StateActive:
		return stylePreviewStateActive
	case model.StateInactive:
		return stylePreviewStateInactive
	case model.StateTodo:
		return stylePreviewStateTodo
	case model.StateDone:
		return stylePreviewStateDone
	}
	return lipgloss.NewStyle()
}

// renderMarkdown renders markdown text for the preview panel.
// Falls back to plain text if glamour is unavailable.
func renderMarkdown(text string, width int) string {
	rendered, err := glamourRender(text, width)
	if err != nil {
		return text
	}
	return rendered
}

// glamourRender renders markdown text using glamour's dark terminal style.
func glamourRender(text string, width int) (string, error) {
	if width < 20 {
		width = 20
	}
	r, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle("dark"),
		glamour.WithWordWrap(width-2),
	)
	if err != nil {
		return text, nil
	}
	out, err := r.Render(text)
	if err != nil {
		return text, nil
	}
	return out, nil
}
