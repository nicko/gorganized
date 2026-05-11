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

// setupApp creates an app with tasks written to a temp .gorgan dir.
func setupApp(t *testing.T, tasks []model.Task) app {
	t.Helper()
	dir := t.TempDir()
	tasksDir := filepath.Join(dir, "tasks")
	if err := createDir(tasksDir); err != nil {
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

func now() time.Time { return time.Now() }

func TestEnterKey_TodoBecomesActive(t *testing.T) {
	tasks := []model.Task{
		{ID: 1, Title: "Task A", State: model.StateTodo, CreatedAt: now(), UpdatedAt: now()},
	}
	a := setupApp(t, tasks)

	// cursor should be on the only task
	a2, _ := a.Update(tea.KeyMsg{Type: tea.KeySpace})
	app2 := a2.(app)

	// find the task in app2's task list
	found := false
	for _, tk := range app2.tasks {
		if tk.ID == 1 {
			found = true
			if tk.State != model.StateActive {
				t.Errorf("after space on todo: state = %s, want active", tk.State)
			}
		}
	}
	if !found {
		t.Fatal("task ID 1 not found after transition")
	}
}

func TestEnterKey_ActiveBecomesInactive(t *testing.T) {
	tasks := []model.Task{
		{ID: 1, Title: "Task A", State: model.StateActive, CreatedAt: now(), UpdatedAt: now()},
	}
	a := setupApp(t, tasks)

	a2, _ := a.Update(tea.KeyMsg{Type: tea.KeySpace})
	app2 := a2.(app)

	for _, tk := range app2.tasks {
		if tk.ID == 1 && tk.State != model.StateInactive {
			t.Errorf("after space on active: state = %s, want inactive", tk.State)
		}
	}
}

func TestDKey_MarksDone(t *testing.T) {
	tasks := []model.Task{
		{ID: 1, Title: "Task A", State: model.StateTodo, CreatedAt: now(), UpdatedAt: now()},
	}
	a := setupApp(t, tasks)
	// First make it active so d is valid (todo → active → done, but spec allows inactive → done)
	// Actually from spec: active → done and inactive → done.
	// Let's put it in active first.
	a2, _ := a.Update(tea.KeyMsg{Type: tea.KeySpace}) // todo → active
	a3, _ := a2.(app).Update(keyMsg("d"))
	app3 := a3.(app)

	for _, tk := range app3.tasks {
		if tk.ID == 1 && tk.State != model.StateDone {
			t.Errorf("after d: state = %s, want done", tk.State)
		}
	}
}

func TestStateChange_RebuildsList(t *testing.T) {
	tasks := []model.Task{
		{ID: 1, Title: "Task A", State: model.StateTodo, CreatedAt: now(), UpdatedAt: now()},
		{ID: 2, Title: "Task B", State: model.StateTodo, CreatedAt: now(), UpdatedAt: now()},
	}
	a := setupApp(t, tasks)

	// Before: no active group
	for _, e := range a.entries {
		if e.isHeader && e.state == model.StateActive {
			t.Fatal("should not have active group before any transition")
		}
	}

	a2, _ := a.Update(tea.KeyMsg{Type: tea.KeySpace}) // task 1: todo → active
	app2 := a2.(app)

	// After: active group should exist
	hasActive := false
	for _, e := range app2.entries {
		if e.isHeader && e.state == model.StateActive {
			hasActive = true
		}
	}
	if !hasActive {
		t.Error("active group should appear after task transitions to active")
	}
}

func TestActiveTask_StartsTimer(t *testing.T) {
	tasks := []model.Task{
		{ID: 1, Title: "Task A", State: model.StateTodo, CreatedAt: now(), UpdatedAt: now()},
	}
	a := setupApp(t, tasks)

	a2, _ := a.Update(tea.KeyMsg{Type: tea.KeySpace}) // todo → active
	app2 := a2.(app)

	if !app2.timer.IsRunning() {
		t.Error("timer should be running after task goes active")
	}
}

func TestInactiveTask_ResetsTimer(t *testing.T) {
	tasks := []model.Task{
		{ID: 1, Title: "Task A", State: model.StateActive, CreatedAt: now(), UpdatedAt: now()},
	}
	a := setupApp(t, tasks)
	// Manually start the timer as loadApp would do for an existing active task
	a.timer.Start(time.Now())

	a2, _ := a.Update(tea.KeyMsg{Type: tea.KeySpace}) // active → inactive
	app2 := a2.(app)

	if app2.timer.IsRunning() {
		t.Error("timer should not be running after task goes inactive")
	}
}

func TestDoneTask_ResetsTimer(t *testing.T) {
	tasks := []model.Task{
		{ID: 1, Title: "Task A", State: model.StateActive, CreatedAt: now(), UpdatedAt: now()},
	}
	a := setupApp(t, tasks)
	a.timer.Start(time.Now())

	a2, _ := a.Update(keyMsg("d"))
	app2 := a2.(app)

	if app2.timer.IsRunning() {
		t.Error("timer should not be running after task is done")
	}
}

func createDir(path string) error {
	return os.MkdirAll(path, 0o755)
}
