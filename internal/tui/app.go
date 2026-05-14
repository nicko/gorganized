package tui

import (
	"fmt"
	"path/filepath"
	"sort"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/nicko/gorganized/internal/kb"
	"github.com/nicko/gorganized/internal/model"
	"github.com/nicko/gorganized/internal/pomodoro"
	"github.com/nicko/gorganized/internal/storage"
	"github.com/nicko/gorganized/internal/timerstate"
)

// Run starts the Bubble Tea application.
func Run(gorganDir string) error {
	m, err := loadApp(gorganDir)
	if err != nil {
		return err
	}
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err = p.Run()
	// Clear the state file so gorgan-bar knows gorgan has exited.
	_ = timerstate.Clear(timerstate.FilePath(gorganDir))
	return err
}

// view represents which list view is active.
type view int

const (
	viewToday view = iota
	viewAll
)

type app struct {
	gorganDir      string
	tasks          []model.Task
	entries        []flatEntry
	cursor         int
	activeView     view
	timer          *pomodoro.Timer
	timerAlert     bool // work interval just expired
	previewVisible bool
	adding         bool
	input          textinput.Model
	editingNote    bool
	noteEditor     textarea.Model
	noteTaskID     int // ID of the task whose notes are open
	kbIndex        *kb.Index
	searching      bool
	searchQuery    string
	searchResults  []kb.Result
	searchCursor   int
	width          int
	height         int
}

func loadApp(gorganDir string) (app, error) {
	tasksDir := filepath.Join(gorganDir, "tasks")
	tasks, err := storage.LoadAll(tasksDir)
	if err != nil {
		return app{}, fmt.Errorf("load tasks: %w", err)
	}

	// Normalise order for tasks whose order is 0 (created before this field existed).
	tasks = normaliseAllOrders(gorganDir, tasks)

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

	// Open knowledge base.
	dbPath := filepath.Join(gorganDir, "kb.db")
	kbIdx, err := kb.Open(dbPath)
	if err != nil {
		return app{}, fmt.Errorf("open kb: %w", err)
	}
	a.kbIndex = kbIdx

	return a, nil
}

// normaliseAllOrders assigns sequential orders within each non-done state group
// if any task in the group has Order == 0. Writes updated tasks to disk.
// Returns the updated task slice.
func normaliseAllOrders(gorganDir string, tasks []model.Task) []model.Task {
	tasksDir := filepath.Join(gorganDir, "tasks")
	states := []model.State{model.StateActive, model.StateInactive, model.StateTodo}
	for _, s := range states {
		var group []model.Task
		for _, t := range tasks {
			if t.State == s {
				group = append(group, t)
			}
		}
		if len(group) == 0 || !hasDuplicateOrders(group) {
			continue
		}
		// Sort by created_at before assigning sequential orders.
		sort.Slice(group, func(i, j int) bool {
			return group[i].CreatedAt.Before(group[j].CreatedAt)
		})
		for i := range group {
			group[i].Order = i + 1
		}
		// Write and update master slice.
		for _, updated := range group {
			_ = storage.Write(tasksDir, updated)
			for j, t := range tasks {
				if t.ID == updated.ID {
					tasks[j] = updated
					break
				}
			}
		}
	}
	return tasks
}

func newApp(gorganDir string) app {
	ti := textinput.New()
	ti.Placeholder = "Task title…"
	ti.CharLimit = 200
	return app{
		gorganDir:      gorganDir,
		timer:          pomodoro.New(25*time.Minute, 5*time.Minute),
		input:          ti,
		noteEditor:     newNoteEditor(),
		previewVisible: true,
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
		if complete {
			a.timerAlert = true
		}
		a.writeTimerState()
		if a.timer.IsRunning() {
			return a, tickCmd()
		}

	case tea.KeyMsg:
		// Search mode intercepts all keys.
		if a.searching {
			return a.handleSearchKey(msg)
		}

		// Note editing mode intercepts all keys.
		if a.editingNote {
			if msg.Type == tea.KeyEsc {
				return a.handleNoteClose()
			}
			var cmd tea.Cmd
			a.noteEditor, cmd = a.noteEditor.Update(msg)
			return a, cmd
		}

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

		case key.Matches(msg, keys.MoveUp):
			return a.handleMoveUp()

		case key.Matches(msg, keys.MoveDown):
			return a.handleMoveDown()

		case key.Matches(msg, keys.Enter):
			if a.timerAlert {
				// Timer paused at phase boundary — advance to next phase.
				a.timer.Advance(time.Now())
				a.timerAlert = false
				a.writeTimerState()
				return a, tickCmd()
			}
			return a.handleEnter()

		case key.Matches(msg, keys.Done):
			return a.handleDone()

		case key.Matches(msg, keys.Add):
			a.adding = true
			a.input.Reset()
			a.input.Focus()
			return a, textinput.Blink

		case key.Matches(msg, keys.Note):
			return a.handleNoteOpen()

		case key.Matches(msg, keys.Preview):
			a.previewVisible = !a.previewVisible

		case key.Matches(msg, keys.Tab):
			if a.activeView == viewToday {
				a.activeView = viewAll
			} else {
				a.activeView = viewToday
			}
			a.rebuildEntries()

		case key.Matches(msg, keys.Search):
			a.searching = true
			a.searchQuery = ""
			a.searchResults = nil
			a.searchCursor = 0
		}
	}
	return a, nil
}
