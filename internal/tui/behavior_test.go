package tui

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nicko/gorganized/internal/model"
	"github.com/nicko/gorganized/internal/storage"
)

// setupApp now registers cleanup for the KB connection to prevent resource leaks.
// NOTE: This shadows the existing setupApp in app_test.go to add the t.Cleanup.
// We can't redefine it there, so this file documents the fix.
// The actual fix is applied below via a wrapper used in new tests.
func setupAppClean(t *testing.T, tasks []model.Task) app {
	t.Helper()
	dir := t.TempDir()
	tasksDir := filepath.Join(dir, "tasks")
	if err := os.MkdirAll(tasksDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, task := range tasks {
		if err := storage.Write(tasksDir, task); err != nil {
			t.Fatalf("Write task: %v", err)
		}
	}
	a, err := loadApp(dir)
	if err != nil {
		t.Fatalf("loadApp: %v", err)
	}
	t.Cleanup(func() {
		if a.kbIndex != nil {
			_ = a.kbIndex.Close()
		}
	})
	return a
}

// --- Inactive → Active transition ---

func TestEnterKey_InactiveBecomesActive(t *testing.T) {
	tasks := []model.Task{
		{ID: 1, Title: "Paused work", State: model.StateInactive,
			CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}
	a := setupAppClean(t, tasks)

	a2, _ := a.Update(tea.KeyMsg{Type: tea.KeySpace})
	app2 := a2.(app)

	for _, tk := range app2.tasks {
		if tk.ID == 1 && tk.State != model.StateActive {
			t.Errorf("after space on inactive: state = %s, want active", tk.State)
		}
	}
}

func TestEnterKey_InactiveBecomesActive_StartsTimer(t *testing.T) {
	tasks := []model.Task{
		{ID: 1, Title: "Paused work", State: model.StateInactive,
			CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}
	a := setupAppClean(t, tasks)

	a2, _ := a.Update(tea.KeyMsg{Type: tea.KeySpace})
	app2 := a2.(app)

	if !app2.timer.IsRunning() {
		t.Error("timer should start when inactive task becomes active")
	}
}

// --- Done task: enter undoes to inactive ---

func TestEnterKey_DoneTask_Undoes(t *testing.T) {
	now := time.Now()
	tasks := []model.Task{
		{ID: 1, Title: "Already done", State: model.StateDone, DoneAt: &now,
			CreatedAt: now, UpdatedAt: now},
	}
	a := setupAppClean(t, tasks)

	// Switch to All view so the done task is visible.
	a2, _ := a.Update(tea.KeyMsg{Type: tea.KeyTab})
	app2 := a2.(app)

	doneIdx := -1
	for i, e := range app2.entries {
		if !e.isHeader && e.task.ID == 1 {
			doneIdx = i
			break
		}
	}
	if doneIdx < 0 {
		t.Skip("done task not in All view entries")
	}
	app2.cursor = doneIdx

	a3, _ := app2.Update(tea.KeyMsg{Type: tea.KeySpace})
	app3 := a3.(app)

	for _, tk := range app3.tasks {
		if tk.ID == 1 && tk.State != model.StateInactive {
			t.Errorf("space on done task should undo to inactive; state = %s", tk.State)
		}
	}
}

func TestDKey_DoneTask_NoOp(t *testing.T) {
	now := time.Now()
	tasks := []model.Task{
		{ID: 1, Title: "Already done", State: model.StateDone, DoneAt: &now,
			CreatedAt: now, UpdatedAt: now},
	}
	a := setupAppClean(t, tasks)

	// Position cursor on the done task via All view
	a2, _ := a.Update(tea.KeyMsg{Type: tea.KeyTab})
	app2 := a2.(app)
	for i, e := range app2.entries {
		if !e.isHeader && e.task.ID == 1 {
			app2.cursor = i
			break
		}
	}

	a3, _ := app2.Update(keyMsg("d"))
	app3 := a3.(app)

	for _, tk := range app3.tasks {
		if tk.ID == 1 && tk.State != model.StateDone {
			t.Errorf("d on already-done task should be no-op; got state %s", tk.State)
		}
	}
}

// --- loadApp auto-starts timer for active task ---

func TestLoadApp_AutoStartsTimerForActiveTask(t *testing.T) {
	dir := t.TempDir()
	tasksDir := filepath.Join(dir, "tasks")
	if err := os.MkdirAll(tasksDir, 0o755); err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	active := model.Task{ID: 1, Title: "In progress", State: model.StateActive,
		CreatedAt: now, UpdatedAt: now}
	if err := storage.Write(tasksDir, active); err != nil {
		t.Fatal(err)
	}

	a, err := loadApp(dir)
	if err != nil {
		t.Fatalf("loadApp: %v", err)
	}
	t.Cleanup(func() {
		if a.kbIndex != nil {
			_ = a.kbIndex.Close()
		}
	})

	if !a.timer.IsRunning() {
		t.Error("loadApp should auto-start pomodoro timer when an active task exists")
	}
}

func TestLoadApp_NoAutoStartForTodoOnly(t *testing.T) {
	dir := t.TempDir()
	tasksDir := filepath.Join(dir, "tasks")
	if err := os.MkdirAll(tasksDir, 0o755); err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	todo := model.Task{ID: 1, Title: "Not started", State: model.StateTodo,
		CreatedAt: now, UpdatedAt: now}
	if err := storage.Write(tasksDir, todo); err != nil {
		t.Fatal(err)
	}

	a, err := loadApp(dir)
	if err != nil {
		t.Fatalf("loadApp: %v", err)
	}
	t.Cleanup(func() {
		if a.kbIndex != nil {
			_ = a.kbIndex.Close()
		}
	})

	if a.timer.IsRunning() {
		t.Error("loadApp should not start timer when there are only todo tasks")
	}
}

// --- Cursor preservation after note save ---

func TestNoteClose_PreservesCursor(t *testing.T) {
	now := time.Now()
	tasks := []model.Task{
		{ID: 1, Title: "Task A", State: model.StateTodo, CreatedAt: now, UpdatedAt: now},
		{ID: 2, Title: "Task B", State: model.StateTodo, CreatedAt: now, UpdatedAt: now},
	}
	a := setupAppClean(t, tasks)

	// Move cursor to task 2
	a2, _ := a.Update(keyMsg("j"))
	app2 := a2.(app)

	cursorTaskID := 0
	if app2.cursor < len(app2.entries) && !app2.entries[app2.cursor].isHeader {
		cursorTaskID = app2.entries[app2.cursor].task.ID
	}

	// Open notes, close without editing
	a3, _ := app2.Update(keyMsg("enter"))
	a4, _ := a3.(app).Update(tea.KeyMsg{Type: tea.KeyEsc})
	app4 := a4.(app)

	cursorAfterID := 0
	if app4.cursor < len(app4.entries) && !app4.entries[app4.cursor].isHeader {
		cursorAfterID = app4.entries[app4.cursor].task.ID
	}

	if cursorTaskID != cursorAfterID {
		t.Errorf("cursor task changed after note save: was task %d, now task %d",
			cursorTaskID, cursorAfterID)
	}
}

// --- No-op keys while in adding mode do not affect list ---

func TestAddingMode_NavigationKeysIgnored(t *testing.T) {
	tasks := []model.Task{
		{ID: 1, Title: "Task A", State: model.StateTodo, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ID: 2, Title: "Task B", State: model.StateTodo, CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}
	a := setupAppClean(t, tasks)
	cursorBefore := a.cursor

	// Enter adding mode
	a2, _ := a.Update(keyMsg("n"))
	// Send j — should go to textinput, not move list cursor
	a3, _ := a2.(app).Update(keyMsg("j"))
	app3 := a3.(app)

	if app3.cursor != cursorBefore {
		t.Errorf("j in adding mode should not move list cursor: was %d, got %d",
			cursorBefore, app3.cursor)
	}
}
