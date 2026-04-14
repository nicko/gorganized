package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nicko/gorganized/internal/model"
	"github.com/nicko/gorganized/internal/storage"
)

// Run starts the Bubble Tea application.
func Run(gorganDir string) error {
	m, err := loadApp(gorganDir)
	if err != nil {
		return err
	}
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err = p.Run()
	return err
}

// view represents which list view is active.
type view int

const (
	viewToday view = iota
	viewAll
)

type app struct {
	gorganDir string
	tasks     []model.Task
	entries   []flatEntry
	cursor    int
	activeView view
	width     int
	height    int
}

func loadApp(gorganDir string) (app, error) {
	tasksDir := filepath.Join(gorganDir, "tasks")
	tasks, err := storage.LoadAll(tasksDir)
	if err != nil {
		return app{}, fmt.Errorf("load tasks: %w", err)
	}
	a := newApp(gorganDir)
	a.tasks = tasks
	a.rebuildEntries()
	return a, nil
}

func newApp(gorganDir string) app {
	return app{gorganDir: gorganDir}
}

// rebuildEntries regenerates the flat entry list from current tasks and view.
func (a *app) rebuildEntries() {
	var groups []stateGroup
	if a.activeView == viewToday {
		groups = groupForToday(a.tasks, today())
	} else {
		groups = groupForAll(a.tasks)
	}
	a.entries = flattenGroups(groups)

	// Reset cursor to first task entry.
	if idx := firstTaskIndex(a.entries); idx >= 0 {
		a.cursor = idx
	} else {
		a.cursor = 0
	}
}

func (a app) Init() tea.Cmd {
	return nil
}

func (a app) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Quit):
			return a, tea.Quit
		case key.Matches(msg, keys.Down):
			a.cursor = nextTaskIndex(a.entries, a.cursor)
		case key.Matches(msg, keys.Up):
			a.cursor = prevTaskIndex(a.entries, a.cursor)
		}
	}
	return a, nil
}

func (a app) View() string {
	var sb strings.Builder

	// Header bar
	viewName := "Today"
	if a.activeView == viewAll {
		viewName = "All"
	}
	header := styleStatusBar.Width(a.width).Render(
		fmt.Sprintf(" gorganized  [%s]  tab: switch view  q: quit", viewName),
	)
	sb.WriteString(header)
	sb.WriteString("\n")

	if len(a.entries) == 0 {
		sb.WriteString("\n  No tasks. Press 'a' to add one.\n")
		return sb.String()
	}

	// Task list
	for i, e := range a.entries {
		if e.isHeader {
			sb.WriteString("\n")
			style := headerStyle(e.state)
			sb.WriteString(style.Render("  " + headerLabel(e.state)))
			sb.WriteString("\n")
			continue
		}

		prefix := "  "
		style := styleTask
		if i == a.cursor {
			prefix = "> "
			style = styleCursor
			_ = style
		}

		line := fmt.Sprintf("%s%s", prefix, e.task.Title)
		if i == a.cursor {
			sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Bold(true).Render(line))
		} else {
			sb.WriteString(line)
		}
		sb.WriteString("\n")
	}

	return sb.String()
}
