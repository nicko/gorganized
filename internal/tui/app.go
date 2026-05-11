package tui

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

// handleEnter cycles the selected task state:
// todo→active, active→inactive, inactive→active, done→inactive (undo).
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
	case model.StateDone:
		to = model.StateInactive
	default:
		return a, nil
	}

	if err := task.Transition(to); err != nil {
		return a, nil
	}
	task.Order = maxOrderForState(a.tasks, to) + 1
	if err := a.updateTask(task); err != nil {
		return a, nil
	}

	// Remove from KB when undoing done.
	if to == model.StateInactive && a.kbIndex != nil {
		_ = a.kbIndex.Remove(task.ID)
	}

	a.rebuildEntries()
	return a.applyTimerForState(to)
}

// handleDone marks the selected task as done and indexes it in the KB.
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

	// Index in knowledge base (best-effort; don't block on error).
	if a.kbIndex != nil {
		_ = a.kbIndex.Index(task)
	}

	return a.applyTimerForState(model.StateDone)
}

// handleMoveUp swaps the selected task's order with the previous task in the same group.
func (a app) handleMoveUp() (tea.Model, tea.Cmd) {
	task, ok := a.selectedTask()
	if !ok || task.State == model.StateDone {
		return a, nil
	}

	// Find the previous task in the same state group from entries.
	prevIdx := -1
	for i := a.cursor - 1; i >= 0; i-- {
		if a.entries[i].isHeader {
			break // reached group boundary
		}
		prevIdx = i
		break
	}
	if prevIdx < 0 {
		return a, nil // already at top of group
	}

	prev := a.entries[prevIdx].task
	a.swapOrders(task, prev)
	return a, nil
}

// handleMoveDown swaps the selected task's order with the next task in the same group.
func (a app) handleMoveDown() (tea.Model, tea.Cmd) {
	task, ok := a.selectedTask()
	if !ok || task.State == model.StateDone {
		return a, nil
	}

	// Find the next task in the same state group from entries.
	nextIdx := -1
	for i := a.cursor + 1; i < len(a.entries); i++ {
		if a.entries[i].isHeader {
			break // reached group boundary
		}
		nextIdx = i
		break
	}
	if nextIdx < 0 {
		return a, nil // already at bottom of group
	}

	next := a.entries[nextIdx].task
	a.swapOrders(task, next)
	return a, nil
}

// swapOrders swaps the Order fields of two tasks, normalising the group first if needed.
func (a *app) swapOrders(t1, t2 model.Task) {
	// Collect the group for potential normalisation.
	group := tasksInSameGroup(a.entries, a.cursor)
	if hasDuplicateOrders(group) {
		group = normaliseOrders(group)
		tasksDir := filepath.Join(a.gorganDir, "tasks")
		updated := make(map[int]model.Task, len(group))
		for _, u := range group {
			_ = storage.Write(tasksDir, u)
			updated[u.ID] = u
		}
		for j, t := range a.tasks {
			if u, ok := updated[t.ID]; ok {
				a.tasks[j] = u
			}
		}
		if u, ok := updated[t1.ID]; ok {
			t1 = u
		}
		if u, ok := updated[t2.ID]; ok {
			t2 = u
		}
	}

	// Swap the two orders.
	t1.Order, t2.Order = t2.Order, t1.Order
	_ = a.updateTask(t1)
	_ = a.updateTask(t2)
	a.rebuildEntries()

	// Keep cursor on t1 after swap (it moved to a different entry).
	if idx := a.findEntryIndex(t1.ID); idx >= 0 {
		a.cursor = idx
	}
}

// findEntryIndex returns the index of the task with the given ID in a.entries,
// or -1 if not found.
func (a *app) findEntryIndex(taskID int) int {
	for i, e := range a.entries {
		if !e.isHeader && e.task.ID == taskID {
			return i
		}
	}
	return -1
}

func (a *app) clearSearch() {
	a.searching = false
	a.searchQuery = ""
	a.searchResults = nil
	a.searchCursor = 0
}

// handleSearchKey processes keypresses while the search overlay is open.
func (a app) handleSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		a.clearSearch()
		return a, nil

	case tea.KeyEnter:
		if len(a.searchResults) > 0 && a.searchCursor < len(a.searchResults) {
			targetID := a.searchResults[a.searchCursor].TaskID
			a.clearSearch()
			// Navigate to the task in the current view.
			if idx := a.findEntryIndex(targetID); idx >= 0 {
				a.cursor = idx
				return a, nil
			}
			// Task not in current view — try switching to All.
			if a.activeView == viewToday {
				a.activeView = viewAll
				a.rebuildEntries()
				if idx := a.findEntryIndex(targetID); idx >= 0 {
					a.cursor = idx
				}
			}
		}
		return a, nil

	case tea.KeyUp:
		if a.searchCursor > 0 {
			a.searchCursor--
		}
		return a, nil

	case tea.KeyDown:
		if a.searchCursor < len(a.searchResults)-1 {
			a.searchCursor++
		}
		return a, nil

	case tea.KeyBackspace, tea.KeyDelete:
		if len(a.searchQuery) > 0 {
			a.searchQuery = a.searchQuery[:len([]rune(a.searchQuery))-1]
			a.runSearch()
		}
		return a, nil

	default:
		if len(msg.Runes) > 0 {
			a.searchQuery += string(msg.Runes)
			a.runSearch()
		}
		return a, nil
	}
}

// runSearch executes the current query against the KB index.
func (a *app) runSearch() {
	if a.kbIndex == nil {
		return
	}
	results, err := a.kbIndex.Search(a.searchQuery)
	if err != nil {
		a.searchResults = nil
		return
	}
	a.searchResults = results
	a.searchCursor = 0
}

// handleNoteOpen opens the note editor for the selected task.
func (a app) handleNoteOpen() (tea.Model, tea.Cmd) {
	task, ok := a.selectedTask()
	if !ok {
		return a, nil
	}
	a.editingNote = true
	a.noteTaskID = task.ID
	a.noteEditor = newNoteEditor()
	a.noteEditor.SetValue(task.Notes)
	a.noteEditor.SetWidth(a.width - 6)
	a.noteEditor.SetHeight(a.height - 8)
	a.noteEditor.Focus()
	return a, textarea.Blink
}

// handleNoteClose saves the note editor content and closes the overlay.
func (a app) handleNoteClose() (tea.Model, tea.Cmd) {
	notes := a.noteEditor.Value()
	a.editingNote = false
	a.noteEditor.Blur()

	// Find and update the task.
	tasksDir := filepath.Join(a.gorganDir, "tasks")
	for i, task := range a.tasks {
		if task.ID == a.noteTaskID {
			a.tasks[i].Notes = notes
			a.tasks[i].UpdatedAt = time.Now()
			_ = storage.Write(tasksDir, a.tasks[i])
			// Refresh the entry so the in-memory copy matches.
			a.rebuildEntries()
			break
		}
	}

	var cmd tea.Cmd
	if a.timer.IsRunning() {
		cmd = tickCmd()
	}
	return a, cmd
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
		Order:     maxOrderForState(a.tasks, model.StateTodo) + 1,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := storage.Write(tasksDir, task); err != nil {
		return a, nil
	}
	a.tasks = append(a.tasks, task)
	a.rebuildEntries()

	// Position cursor on the newly added task.
	if idx := a.findEntryIndex(task.ID); idx >= 0 {
		a.cursor = idx
	}

	var cmd tea.Cmd
	if a.timer.IsRunning() {
		cmd = tickCmd()
	}
	return a, cmd
}

// applyTimerForState starts or resets the timer based on the new state.
func (a app) applyTimerForState(state model.State) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	if state == model.StateActive {
		a.timer.Start(time.Now())
		cmd = tickCmd()
	} else {
		a.timer.Reset()
	}
	a.timerAlert = false
	a.writeTimerState()
	return a, cmd
}

// writeTimerState writes the current timer snapshot to .gorgan/timer.state.
// Errors are silently ignored — the state file is best-effort.
func (a app) writeTimerState() {
	remaining, _ := a.timer.Remaining(time.Now())
	s := timerstate.State{
		Running:   a.timer.IsRunning(),
		Phase:     a.timer.CurrentPhase().String(),
		Pomodoro:  a.timer.Count() + 1,
		Remaining: int(remaining.Seconds()),
		Alert:     a.timerAlert,
	}
	_ = timerstate.Write(timerstate.FilePath(a.gorganDir), s)
}

// formatDuration formats accumulated seconds as "Xh Ym", "Ym", or "< 1m".
func formatDuration(seconds int) string {
	if seconds <= 0 {
		return "< 1m"
	}
	h := seconds / 3600
	m := (seconds % 3600) / 60
	if h > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	if m == 0 {
		return "< 1m"
	}
	return fmt.Sprintf("%dm", m)
}

func (a app) View() string {
	var sb strings.Builder

	// Header / status bar
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
	if a.timerAlert {
		barStyle = styleStatusBarAlert
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

	// Search overlay
	if a.searching {
		sb.WriteString(renderSearchOverlay(a.searchQuery, a.searchResults, a.searchCursor))
		sb.WriteString(helpBar)
		return sb.String()
	}

	// Full-screen note editor overlay
	if a.editingNote {
		var noteSB strings.Builder
		// Find task title for the header
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

	// Inline add prompt
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

	// Task list (full width)
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

// viewWithPreview renders the 60/40 split layout.
func (a app) viewWithPreview() string {
	if a.width == 0 {
		return a.renderTaskList(0)
	}
	listWidth := (a.width * 60) / 100
	previewWidth := a.width - listWidth - 1 // -1 for the divider

	list := a.renderTaskList(listWidth)
	preview := renderPreview(a.selectedTaskForPreview(), previewWidth, a.height-2)

	return lipgloss.JoinHorizontal(lipgloss.Top, list,
		lipgloss.NewStyle().
			BorderLeft(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240")).
			Width(previewWidth).
			Render(preview),
	)
}

// selectedTaskForPreview returns the selected task, or zero value if none.
func (a app) selectedTaskForPreview() model.Task {
	t, _ := a.selectedTask()
	return t
}

// renderTaskList renders the task list at the given width.
func (a app) renderTaskList(_ int) string {
	var sb strings.Builder

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

		title := e.task.Title
		if i == a.cursor {
			line := "> " + title
			sb.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Bold(true).Render(line))
		} else {
			sb.WriteString("  " + title)
		}
		sb.WriteString(timeStr)
		sb.WriteString("\n")
	}
	return sb.String()
}
