package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nicko/gorganized/internal/model"
	"github.com/nicko/gorganized/internal/storage"
)

func TestAddKey_EntersAddingMode(t *testing.T) {
	a := setupApp(t, nil)

	a2, _ := a.Update(keyMsg("a"))
	app2 := a2.(app)

	if !app2.adding {
		t.Error("after pressing 'a', app should be in adding mode")
	}
}

func TestEsc_CancelsAddingMode(t *testing.T) {
	a := setupApp(t, nil)
	a2, _ := a.Update(keyMsg("a"))

	a3, _ := a2.(app).Update(tea.KeyMsg{Type: tea.KeyEsc})
	app3 := a3.(app)

	if app3.adding {
		t.Error("after esc, adding mode should be cancelled")
	}
}

func TestAddTask_CreatesTaskOnEnter(t *testing.T) {
	dir := t.TempDir()
	tasksDir := filepath.Join(dir, "tasks")
	if err := os.MkdirAll(tasksDir, 0o755); err != nil {
		t.Fatal(err)
	}
	a, err := loadApp(dir)
	if err != nil {
		t.Fatal(err)
	}

	// Enter adding mode
	a2, _ := a.Update(keyMsg("a"))

	// Type a title character by character
	a3 := a2.(app)
	for _, ch := range "Buy milk" {
		m, _ := a3.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		a3 = m.(app)
	}

	// Confirm with enter
	a4, _ := a3.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app4 := a4.(app)

	if app4.adding {
		t.Error("after enter, adding mode should be closed")
	}

	// Task should be in the list
	found := false
	for _, task := range app4.tasks {
		if task.Title == "Buy milk" {
			found = true
			if task.State != model.StateTodo {
				t.Errorf("new task state: got %s, want todo", task.State)
			}
		}
	}
	if !found {
		t.Errorf("task 'Buy milk' not found in app.tasks; tasks: %v", app4.tasks)
	}

	// File should exist on disk
	tasks, err := storage.LoadAll(tasksDir)
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	diskFound := false
	for _, task := range tasks {
		if task.Title == "Buy milk" {
			diskFound = true
		}
	}
	if !diskFound {
		t.Error("task 'Buy milk' not found on disk")
	}
}

func TestAddTask_EmptyTitleIgnored(t *testing.T) {
	a := setupApp(t, nil)
	a2, _ := a.Update(keyMsg("a"))

	// Confirm immediately with no title typed
	a3, _ := a2.(app).Update(tea.KeyMsg{Type: tea.KeyEnter})
	app3 := a3.(app)

	if app3.adding {
		t.Error("enter on empty title should close adding mode")
	}
	if len(app3.tasks) != 0 {
		t.Errorf("empty title should not create a task; got %d tasks", len(app3.tasks))
	}
}

func TestAddTask_AppearsInTodoGroup(t *testing.T) {
	dir := t.TempDir()
	tasksDir := filepath.Join(dir, "tasks")
	if err := os.MkdirAll(tasksDir, 0o755); err != nil {
		t.Fatal(err)
	}
	a, _ := loadApp(dir)

	a2, _ := a.Update(keyMsg("a"))
	a3 := a2.(app)
	for _, ch := range "New task" {
		m, _ := a3.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		a3 = m.(app)
	}
	a4, _ := a3.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app4 := a4.(app)

	// Should appear in entries under the todo group
	foundInEntries := false
	for _, e := range app4.entries {
		if !e.isHeader && e.task.Title == "New task" {
			foundInEntries = true
		}
	}
	if !foundInEntries {
		t.Error("new task should appear in the entry list immediately")
	}
}

func TestAddTask_IdsIncrement(t *testing.T) {
	dir := t.TempDir()
	tasksDir := filepath.Join(dir, "tasks")
	if err := os.MkdirAll(tasksDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Seed one task with ID 1
	seed := model.Task{ID: 1, Title: "Existing", State: model.StateTodo, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := storage.Write(tasksDir, seed); err != nil {
		t.Fatal(err)
	}

	a, _ := loadApp(dir)
	a2, _ := a.Update(keyMsg("a"))
	a3 := a2.(app)
	for _, ch := range "Second task" {
		m, _ := a3.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		a3 = m.(app)
	}
	a4, _ := a3.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app4 := a4.(app)

	for _, task := range app4.tasks {
		if task.Title == "Second task" {
			if task.ID != 2 {
				t.Errorf("second task ID: got %d, want 2", task.ID)
			}
		}
	}
}

// inputValue returns the trimmed value typed into the add input so far.
// Used as a helper to inspect app state without accessing private fields directly.
func inputValue(a app) string {
	return strings.TrimSpace(a.input.Value())
}
