package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/nicko/gorganized/internal/model"
	"github.com/nicko/gorganized/internal/pomodoro"
)

func (a app) View() string {
	var sb strings.Builder

	viewName := "Today"
	if a.activeView == viewAll {
		viewName = "All"
	}
	timer := timerStatus(a.timer, time.Now())
	if a.timerAlert && timer == "" {
		if a.timer.CurrentPhase() == pomodoro.PhaseBreak {
			timer = "[Work done]  space: start break"
		} else {
			timer = "[Break done]  space: resume work"
		}
	}
	barStyle := styleStatusBar
	switch {
	case a.timerAlert:
		barStyle = styleStatusBarAlert
	case a.timer.IsRunning() && a.timer.CurrentPhase() == pomodoro.PhaseBreak:
		barStyle = styleStatusBarBreak
	case a.timer.IsRunning():
		barStyle = styleStatusBarWork
	}
	left := fmt.Sprintf(" gorganized  [%s]", viewName)
	var barContent string
	if timer == "" || a.width == 0 {
		barContent = left
	} else {
		pad := (a.width-lipgloss.Width(timer))/2 - lipgloss.Width(left)
		if pad < 1 {
			pad = 1
		}
		barContent = left + strings.Repeat(" ", pad) + timer
	}
	header := barStyle.Width(a.width).Render(barContent)
	sb.WriteString(header)
	sb.WriteString("\n")

	helpBar := "\n" + a.renderHelpBar(a.width)

	if a.searching {
		sb.WriteString(renderSearchOverlay(a.searchQuery, a.searchResults, a.searchCursor))
		sb.WriteString(helpBar)
		return sb.String()
	}

	if a.editingNote {
		var noteSB strings.Builder
		taskTitle := "Notes"
		for _, task := range a.tasks {
			if task.ID == a.noteTaskID {
				taskTitle = task.Title
				break
			}
		}
		noteSB.WriteString(styleNoteHeader.Render("Notes: " + taskTitle))
		noteSB.WriteString("\n")
		noteSB.WriteString(a.noteEditor.View())
		noteSB.WriteString("\n")
		noteSB.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("243")).Render("  esc: save & close"))
		sb.WriteString(styleNoteOverlay.Render(noteSB.String()))
		sb.WriteString(helpBar)
		return sb.String()
	}

	if a.adding {
		sb.WriteString("\n  New task: ")
		sb.WriteString(a.input.View())
		sb.WriteString("\n")
	}

	if len(a.entries) == 0 && !a.adding {
		sb.WriteString("\n  No tasks. Press 'n' to add one.\n")
		sb.WriteString(helpBar)
		return sb.String()
	}

	if a.previewVisible {
		return sb.String() + a.viewWithPreview() + helpBar
	}

	sb.WriteString(a.renderTaskList(a.width))
	sb.WriteString(helpBar)
	return sb.String()
}

func (a app) renderHelpBar(width int) string {
	hints := strings.Join([]string{
		keys.Enter.Help().Key + ": " + keys.Enter.Help().Desc,
		keys.Note.Help().Key + ": " + keys.Note.Help().Desc,
		keys.Add.Help().Key + ": " + keys.Add.Help().Desc,
		keys.Done.Help().Key + ": " + keys.Done.Help().Desc,
		keys.MoveUp.Help().Key + "/" + keys.MoveDown.Help().Key + ": move",
		keys.Preview.Help().Key + ": " + keys.Preview.Help().Desc,
		keys.Tab.Help().Key + ": " + keys.Tab.Help().Desc,
		keys.Search.Help().Key + ": " + keys.Search.Help().Desc,
		keys.Up.Help().Key + "/" + keys.Down.Help().Key + ": navigate",
		keys.Quit.Help().Key + ": " + keys.Quit.Help().Desc,
	}, "  ")
	return styleHelpBar.Width(width).Render(hints)
}

func (a app) viewWithPreview() string {
	if a.width == 0 {
		return a.renderTaskList(0)
	}
	listWidth := a.width / 2
	previewWidth := a.width - listWidth - 1

	list := a.renderTaskList(listWidth)
	task := a.selectedTaskForPreview()
	pomCount := 0
	if task.State == model.StateActive {
		pomCount = a.timer.Count()
	}
	preview := renderPreview(task, pomCount, previewWidth, a.height-2)

	return lipgloss.JoinHorizontal(lipgloss.Top, list,
		lipgloss.NewStyle().
			BorderLeft(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240")).
			Width(previewWidth).
			Render(preview),
	)
}

func (a app) selectedTaskForPreview() model.Task {
	t, _ := a.selectedTask()
	return t
}

func (a app) renderTaskList(width int) string {
	var sb strings.Builder
	wrapWidth := width - 2 // subtract prefix "  " or "> "

	for i, e := range a.entries {
		if e.isHeader {
			sb.WriteString("\n")
			style := headerStyle(e.state)
			sb.WriteString(style.Render("  " + headerLabel(e.state)))
			sb.WriteString("\n")
			continue
		}

		timeStr := ""
		totalSecs := e.task.TimeSpent
		if e.task.State == model.StateActive && e.task.ActiveSince != nil {
			totalSecs += int(time.Since(*e.task.ActiveSince).Seconds())
		}
		if totalSecs > 0 {
			timeStr = "  " + lipgloss.NewStyle().Foreground(lipgloss.Color("243")).Render(formatDuration(totalSecs))
		}

		isCursor := i == a.cursor
		lines := wrapWords(e.task.Title, wrapWidth)
		for j, line := range lines {
			isLast := j == len(lines)-1
			if j == 0 {
				if isCursor {
					sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Bold(true).Render("> " + line))
				} else {
					sb.WriteString("  " + line)
				}
			} else {
				sb.WriteString("  ↳ " + line)
			}
			if isLast {
				sb.WriteString(timeStr)
			}
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

// wrapWords wraps text at word boundaries to fit within width runes.
// Returns at least one element. When width <= 0, returns the original text unsplit.
func wrapWords(text string, width int) []string {
	if width <= 0 || lipgloss.Width(text) <= width {
		return []string{text}
	}
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{text}
	}
	var lines []string
	current := words[0]
	for _, word := range words[1:] {
		candidate := current + " " + word
		if lipgloss.Width(candidate) <= width {
			current = candidate
		} else {
			lines = append(lines, current)
			current = word
		}
	}
	return append(lines, current)
}
