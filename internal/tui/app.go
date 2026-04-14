package tui

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nicko/gorganized/internal/model"
	"github.com/nicko/gorganized/internal/pomodoro"
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
	gorganDir  string
	tasks      []model.Task
	entries    []flatEntry
	cursor     int
	activeView view
	timer      *pomodoro.Timer
	timerAlert bool // work interval just expired
	adding     bool
	input      textinput.Model
	width      int
	height     int
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

	// Resume timer if there is already an active task.
	for _, t := range tasks {
		if t.State == model.StateActive {
			a.timer.Start(time.Now())
			break
		}
	}
	return a, nil
}

func newApp(gorganDir string) app {
	ti := textinput.New()
	ti.Placeholder = "Task title…"
	ti.CharLimit = 200
	return app{
		gorganDir: gorganDir,
		timer:     pomodoro.New(25*time.Minute, 5*time.Minute),
		input:     ti,
	}
}

// rebuildEntries regenerates the flat entry list from current tasks and view.
// Preserves the cursor on the same task ID when possible.
func (a *app) rebuildEntries() {
	// Remember which task is currently selected.
	var selectedID int
	if a.cursor < len(a.entries) && !a.entries[a.cursor].isHeader {
		selectedID = a.entries[a.cursor].task.ID
	}

	var groups []stateGroup
	if a.activeView == viewToday {
		groups = groupForToday(a.tasks, today())
	} else {
		groups = groupForAll(a.tasks)
	}
	a.entries = flattenGroups(groups)

	// Try to restore cursor to the same task.
	if selectedID != 0 {
		for i, e := range a.entries {
			if !e.isHeader && e.task.ID == selectedID {
				a.cursor = i
				return
			}
		}
	}

	// Fall back to first task entry.
	if idx := firstTaskIndex(a.entries); idx >= 0 {
		a.cursor = idx
	} else {
		a.cursor = 0
	}
}

// selectedTask returns the task at the cursor, and whether one exists.
func (a *app) selectedTask() (model.Task, bool) {
	if a.cursor < len(a.entries) && !a.entries[a.cursor].isHeader {
		return a.entries[a.cursor].task, true
	}
	return model.Task{}, false
}

// updateTask updates a task in a.tasks by ID and writes it to disk.
func (a *app) updateTask(t model.Task) error {
	tasksDir := filepath.Join(a.gorganDir, "tasks")
	if err := storage.Write(tasksDir, t); err != nil {
		return err
	}
	for i, existing := range a.tasks {
		if existing.ID == t.ID {
			a.tasks[i] = t
			return nil
		}
	}
	return nil
}

func (a app) Init() tea.Cmd {
	if a.timer.IsRunning() {
		return tickCmd()
	}
	return nil
}

func (a app) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height

	case tickMsg:
		_, _, complete := a.timer.Tick(time.Time(msg))
		if complete && a.timer.CurrentPhase() == pomodoro.PhaseBreak {
			a.timerAlert = true
		} else {
			a.timerAlert = false
		}
		if a.timer.IsRunning() {
			return a, tickCmd()
		}

	case tea.KeyMsg:
		// Adding mode intercepts most keys.
		if a.adding {
			switch msg.Type {
			case tea.KeyEsc:
				a.adding = false
				a.input.Reset()
				return a, nil
			case tea.KeyEnter:
				return a.handleAddConfirm()
			default:
				var cmd tea.Cmd
				a.input, cmd = a.input.Update(msg)
				return a, cmd
			}
		}

		switch {
		case key.Matches(msg, keys.Quit):
			return a, tea.Quit

		case key.Matches(msg, keys.Down):
			a.cursor = nextTaskIndex(a.entries, a.cursor)

		case key.Matches(msg, keys.Up):
			a.cursor = prevTaskIndex(a.entries, a.cursor)

		case key.Matches(msg, keys.Enter):
			return a.handleEnter()

		case key.Matches(msg, keys.Done):
			return a.handleDone()

		case key.Matches(msg, keys.Add):
			a.adding = true
			a.input.Reset()
			a.input.Focus()
			return a, textinput.Blink
		}
	}
	return a, nil
}

// handleEnter cycles the selected task: todo→active, active→inactive, inactive→active.
func (a app) handleEnter() (tea.Model, tea.Cmd) {
	task, ok := a.selectedTask()
	if !ok {
		return a, nil
	}

	var to model.State
	switch task.State {
	case model.StateTodo:
		to = model.StateActive
	case model.StateActive:
		to = model.StateInactive
	case model.StateInactive:
		to = model.StateActive
	default:
		return a, nil
	}

	if err := task.Transition(to); err != nil {
		return a, nil
	}
	if err := a.updateTask(task); err != nil {
		return a, nil
	}
	a.rebuildEntries()

	return a.applyTimerForState(to)
}

// handleDone marks the selected task as done.
func (a app) handleDone() (tea.Model, tea.Cmd) {
	task, ok := a.selectedTask()
	if !ok {
		return a, nil
	}
	if task.State == model.StateDone {
		return a, nil
	}

	if err := task.Transition(model.StateDone); err != nil {
		return a, nil
	}
	if err := a.updateTask(task); err != nil {
		return a, nil
	}
	a.rebuildEntries()

	return a.applyTimerForState(model.StateDone)
}

// handleAddConfirm creates a new task from the input field and saves it.
func (a app) handleAddConfirm() (tea.Model, tea.Cmd) {
	title := strings.TrimSpace(a.input.Value())
	a.adding = false
	a.input.Reset()

	if title == "" {
		return a, nil
	}

	tasksDir := filepath.Join(a.gorganDir, "tasks")
	id, err := storage.NextID(tasksDir)
	if err != nil {
		return a, nil
	}
	now := time.Now()
	task := model.Task{
		ID:        id,
		Title:     title,
		State:     model.StateTodo,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := storage.Write(tasksDir, task); err != nil {
		return a, nil
	}
	a.tasks = append(a.tasks, task)
	a.rebuildEntries()

	// Position cursor on the newly added task.
	for i, e := range a.entries {
		if !e.isHeader && e.task.ID == task.ID {
			a.cursor = i
			break
		}
	}

	var cmd tea.Cmd
	if a.timer.IsRunning() {
		cmd = tickCmd()
	}
	return a, cmd
}

// applyTimerForState starts or resets the timer based on the new state.
func (a app) applyTimerForState(state model.State) (tea.Model, tea.Cmd) {
	switch state {
	case model.StateActive:
		a.timer.Start(time.Now())
		a.timerAlert = false
		return a, tickCmd()
	default:
		a.timer.Reset()
		a.timerAlert = false
		return a, nil
	}
}

func (a app) View() string {
	var sb strings.Builder

	// Header / status bar
	viewName := "Today"
	if a.activeView == viewAll {
		viewName = "All"
	}
	timerStr := timerStatus(a.timer, time.Now())
	barStyle := styleStatusBar
	if a.timerAlert {
		barStyle = styleStatusBarAlert
	}
	header := barStyle.Width(a.width).Render(
		fmt.Sprintf(" gorganized  [%s]%s  tab: switch view  q: quit", viewName, timerStr),
	)
	sb.WriteString(header)
	sb.WriteString("\n")

	// Inline add prompt
	if a.adding {
		sb.WriteString("\n  New task: ")
		sb.WriteString(a.input.View())
		sb.WriteString("\n")
	}

	if len(a.entries) == 0 && !a.adding {
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

		line := fmt.Sprintf("  %s", e.task.Title)
		if i == a.cursor {
			line = fmt.Sprintf("> %s", e.task.Title)
			sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Bold(true).Render(line))
		} else {
			sb.WriteString(line)
		}
		sb.WriteString("\n")
	}

	return sb.String()
}
