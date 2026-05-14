package tui

import (
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/nicko/gorganized/internal/model"
	"github.com/nicko/gorganized/internal/storage"
)

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

	if to == model.StateInactive && a.kbIndex != nil {
		_ = a.kbIndex.Remove(task.ID)
	}

	a.rebuildEntries()
	return a.applyTimerForState(to)
}

func (a app) handleDone() (tea.Model, tea.Cmd) {
	task, ok := a.selectedTask()
	if !ok || task.State == model.StateDone {
		return a, nil
	}

	if err := task.Transition(model.StateDone); err != nil {
		return a, nil
	}
	if err := a.updateTask(task); err != nil {
		return a, nil
	}
	a.rebuildEntries()

	if a.kbIndex != nil {
		_ = a.kbIndex.Index(task)
	}

	return a.applyTimerForState(model.StateDone)
}

func (a app) handleMoveUp() (tea.Model, tea.Cmd) {
	task, ok := a.selectedTask()
	if !ok || task.State == model.StateDone {
		return a, nil
	}

	prevIdx := -1
	for i := a.cursor - 1; i >= 0; i-- {
		if a.entries[i].isHeader {
			break
		}
		prevIdx = i
		break
	}
	if prevIdx < 0 {
		return a, nil
	}

	prev := a.entries[prevIdx].task
	a.swapOrders(task, prev)
	return a, nil
}

func (a app) handleMoveDown() (tea.Model, tea.Cmd) {
	task, ok := a.selectedTask()
	if !ok || task.State == model.StateDone {
		return a, nil
	}

	nextIdx := -1
	for i := a.cursor + 1; i < len(a.entries); i++ {
		if a.entries[i].isHeader {
			break
		}
		nextIdx = i
		break
	}
	if nextIdx < 0 {
		return a, nil
	}

	next := a.entries[nextIdx].task
	a.swapOrders(task, next)
	return a, nil
}

func (a *app) swapOrders(t1, t2 model.Task) {
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

	t1.Order, t2.Order = t2.Order, t1.Order
	_ = a.updateTask(t1)
	_ = a.updateTask(t2)
	a.rebuildEntries()

	if idx := a.findEntryIndex(t1.ID); idx >= 0 {
		a.cursor = idx
	}
}

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

func (a app) handleSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		a.clearSearch()
		return a, nil

	case tea.KeyEnter:
		if len(a.searchResults) > 0 && a.searchCursor < len(a.searchResults) {
			targetID := a.searchResults[a.searchCursor].TaskID
			a.clearSearch()
			if idx := a.findEntryIndex(targetID); idx >= 0 {
				a.cursor = idx
				return a, nil
			}
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

func (a app) handleNoteClose() (tea.Model, tea.Cmd) {
	notes := a.noteEditor.Value()
	a.editingNote = false
	a.noteEditor.Blur()

	tasksDir := filepath.Join(a.gorganDir, "tasks")
	for i, task := range a.tasks {
		if task.ID == a.noteTaskID {
			a.tasks[i].Notes = notes
			a.tasks[i].UpdatedAt = time.Now()
			_ = storage.Write(tasksDir, a.tasks[i])
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

	if idx := a.findEntryIndex(task.ID); idx >= 0 {
		a.cursor = idx
	}

	var cmd tea.Cmd
	if a.timer.IsRunning() {
		cmd = tickCmd()
	}
	return a, cmd
}
