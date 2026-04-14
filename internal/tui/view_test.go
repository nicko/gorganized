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

// tabMsg sends a tab keypress.
func tabMsg() tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyTab}
}

func TestTab_SwitchesToAllView(t *testing.T) {
	a := setupApp(t, nil)
	if a.activeView != viewToday {
		t.Fatal("default view should be Today")
	}

	a2, _ := a.Update(tabMsg())
	app2 := a2.(app)

	if app2.activeView != viewAll {
		t.Errorf("after tab: expected All view, got %v", app2.activeView)
	}
}

func TestTab_TogglesBackToToday(t *testing.T) {
	a := setupApp(t, nil)
	a2, _ := a.Update(tabMsg())
	a3, _ := a2.(app).Update(tabMsg())
	app3 := a3.(app)

	if app3.activeView != viewToday {
		t.Errorf("after two tabs: expected Today view, got %v", app3.activeView)
	}
}

func TestAllView_IncludesDoneTasks(t *testing.T) {
	yesterday := time.Now().AddDate(0, 0, -1)
	tasks := []model.Task{
		{ID: 1, Title: "Done Yesterday", State: model.StateDone, DoneAt: &yesterday,
			CreatedAt: yesterday, UpdatedAt: yesterday},
		{ID: 2, Title: "Todo Task", State: model.StateTodo,
			CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}
	a := setupApp(t, tasks)

	// Today view: done-yesterday task should NOT appear
	hasDoneInToday := false
	for _, e := range a.entries {
		if !e.isHeader && e.task.ID == 1 {
			hasDoneInToday = true
		}
	}
	if hasDoneInToday {
		t.Error("Today view should not include tasks done yesterday")
	}

	// All view: done-yesterday task SHOULD appear
	a2, _ := a.Update(tabMsg())
	app2 := a2.(app)
	hasDoneInAll := false
	for _, e := range app2.entries {
		if !e.isHeader && e.task.ID == 1 {
			hasDoneInAll = true
		}
	}
	if !hasDoneInAll {
		t.Error("All view should include tasks done yesterday")
	}
}

func TestTodayView_IncludesTodayDone(t *testing.T) {
	now := time.Now()
	tasks := []model.Task{
		{ID: 1, Title: "Done Today", State: model.StateDone, DoneAt: &now,
			CreatedAt: now, UpdatedAt: now},
	}
	a := setupApp(t, tasks)

	// Today view should include task done today
	found := false
	for _, e := range a.entries {
		if !e.isHeader && e.task.ID == 1 {
			found = true
		}
	}
	if !found {
		t.Error("Today view should include tasks done today")
	}
}

func TestAllView_ShowsDoneGroupHeader(t *testing.T) {
	yesterday := time.Now().AddDate(0, 0, -1)
	tasks := []model.Task{
		{ID: 1, Title: "Old done", State: model.StateDone, DoneAt: &yesterday,
			CreatedAt: yesterday, UpdatedAt: yesterday},
	}
	a := setupApp(t, tasks)

	// Tab to All view
	a2, _ := a.Update(tabMsg())
	app2 := a2.(app)

	hasDoneHeader := false
	for _, e := range app2.entries {
		if e.isHeader && e.state == model.StateDone {
			hasDoneHeader = true
		}
	}
	if !hasDoneHeader {
		t.Error("All view should show Done group header")
	}
}

func TestTabToggle_RebuildsCursor(t *testing.T) {
	// Cursor should land on a task entry (not a header) after tab.
	yesterday := time.Now().AddDate(0, 0, -1)
	tasks := []model.Task{
		{ID: 1, Title: "Done Yesterday", State: model.StateDone, DoneAt: &yesterday,
			CreatedAt: yesterday, UpdatedAt: yesterday},
		{ID: 2, Title: "Todo Task", State: model.StateTodo,
			CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}
	a := setupApp(t, tasks)

	a2, _ := a.Update(tabMsg()) // → All view
	app2 := a2.(app)

	if app2.cursor >= len(app2.entries) {
		t.Fatalf("cursor %d out of range (%d entries)", app2.cursor, len(app2.entries))
	}
	if app2.entries[app2.cursor].isHeader {
		t.Errorf("cursor should point to a task, not a header (cursor=%d)", app2.cursor)
	}
}

func TestTabToggle_MarkDoneThenViewInAll(t *testing.T) {
	// Integration: mark a task done today, verify it shows in both views.
	dir := t.TempDir()
	tasksDir := filepath.Join(dir, "tasks")
	if err := os.MkdirAll(tasksDir, 0o755); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	task := model.Task{ID: 1, Title: "To finish", State: model.StateActive,
		CreatedAt: now, UpdatedAt: now}
	if err := storage.Write(tasksDir, task); err != nil {
		t.Fatal(err)
	}

	a, _ := loadApp(dir)
	// Mark done (active → done via d key)
	a2, _ := a.Update(keyMsg("d"))

	// Should still be in Today view and task visible
	app2 := a2.(app)
	inToday := false
	for _, e := range app2.entries {
		if !e.isHeader && e.task.ID == 1 {
			inToday = true
		}
	}
	if !inToday {
		t.Error("task done today should still be visible in Today view")
	}

	// Switch to All — still visible
	a3, _ := app2.Update(tabMsg())
	app3 := a3.(app)
	inAll := false
	for _, e := range app3.entries {
		if !e.isHeader && e.task.ID == 1 {
			inAll = true
		}
	}
	if !inAll {
		t.Error("task done today should be visible in All view")
	}
}
