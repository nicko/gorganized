package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nicko/gorganized/internal/model"
)

func makeTasks() []model.Task {
	now := time.Now()
	return []model.Task{
		{ID: 1, Title: "Todo A", State: model.StateTodo, CreatedAt: now, UpdatedAt: now},
		{ID: 2, Title: "Active B", State: model.StateActive, CreatedAt: now, UpdatedAt: now},
		{ID: 3, Title: "Inactive C", State: model.StateInactive, CreatedAt: now, UpdatedAt: now},
		{ID: 4, Title: "Todo D", State: model.StateTodo, CreatedAt: now, UpdatedAt: now},
		{ID: 5, Title: "Done E", State: model.StateDone, CreatedAt: now, UpdatedAt: now},
	}
}

func TestGroupForToday_Order(t *testing.T) {
	tasks := makeTasks()
	groups := groupForToday(tasks, today())

	if len(groups) != 3 {
		t.Fatalf("expected 3 groups (active, inactive, todo), got %d", len(groups))
	}
	if groups[0].state != model.StateActive {
		t.Errorf("group[0]: want active, got %s", groups[0].state)
	}
	if groups[1].state != model.StateInactive {
		t.Errorf("group[1]: want inactive, got %s", groups[1].state)
	}
	if groups[2].state != model.StateTodo {
		t.Errorf("group[2]: want todo, got %s", groups[2].state)
	}
}

func TestGroupForToday_ExcludesDonePreviousDays(t *testing.T) {
	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)
	tasks := []model.Task{
		{ID: 1, Title: "Done Yesterday", State: model.StateDone, DoneAt: &yesterday, CreatedAt: now, UpdatedAt: now},
		{ID: 2, Title: "Active Now", State: model.StateActive, CreatedAt: now, UpdatedAt: now},
	}
	groups := groupForToday(tasks, today())
	for _, g := range groups {
		if g.state == model.StateDone {
			t.Error("today view should not include done tasks from previous days")
		}
	}
}

func TestGroupForToday_IncludesTodayDone(t *testing.T) {
	now := time.Now()
	tasks := []model.Task{
		{ID: 1, Title: "Done Today", State: model.StateDone, DoneAt: &now, CreatedAt: now, UpdatedAt: now},
	}
	groups := groupForToday(tasks, today())
	found := false
	for _, g := range groups {
		if g.state == model.StateDone {
			found = true
			if len(g.tasks) != 1 {
				t.Errorf("expected 1 done task today, got %d", len(g.tasks))
			}
		}
	}
	if !found {
		t.Error("today view should include done tasks completed today")
	}
}

func TestGroupForToday_EmptyGroupsOmitted(t *testing.T) {
	now := time.Now()
	tasks := []model.Task{
		{ID: 1, Title: "Only Todo", State: model.StateTodo, CreatedAt: now, UpdatedAt: now},
	}
	groups := groupForToday(tasks, today())
	for _, g := range groups {
		if len(g.tasks) == 0 {
			t.Errorf("empty group %s should not be included", g.state)
		}
	}
}

func TestFlattenGroups(t *testing.T) {
	tasks := makeTasks() // active, inactive, todo (done excluded)
	groups := groupForToday(tasks, today())
	flat := flattenGroups(groups)

	// flat contains group-header entries + task entries, all task IDs in state order
	taskEntries := make([]flatEntry, 0)
	for _, e := range flat {
		if !e.isHeader {
			taskEntries = append(taskEntries, e)
		}
	}

	// Active first, then inactive, then todo (done excluded from today without today's done_at)
	if taskEntries[0].task.State != model.StateActive {
		t.Errorf("first task entry should be active, got %s", taskEntries[0].task.State)
	}
	if taskEntries[1].task.State != model.StateInactive {
		t.Errorf("second task entry should be inactive, got %s", taskEntries[1].task.State)
	}
}

func TestNavigation_CursorDown(t *testing.T) {
	a := newApp("/tmp")
	a.entries = []flatEntry{
		{isHeader: true},
		{isHeader: false, task: model.Task{ID: 1}},
		{isHeader: false, task: model.Task{ID: 2}},
	}
	a.cursor = 1 // on first task

	a2, _ := a.Update(keyMsg("j"))
	app2 := a2.(app)
	if app2.cursor != 2 {
		t.Errorf("cursor after j: got %d, want 2", app2.cursor)
	}
}

func TestNavigation_CursorUp(t *testing.T) {
	a := newApp("/tmp")
	a.entries = []flatEntry{
		{isHeader: true},
		{isHeader: false, task: model.Task{ID: 1}},
		{isHeader: false, task: model.Task{ID: 2}},
	}
	a.cursor = 2

	a2, _ := a.Update(keyMsg("k"))
	app2 := a2.(app)
	if app2.cursor != 1 {
		t.Errorf("cursor after k: got %d, want 1", app2.cursor)
	}
}

func TestNavigation_SkipsHeaders(t *testing.T) {
	a := newApp("/tmp")
	a.entries = []flatEntry{
		{isHeader: true},                          // 0 — header, skipped
		{isHeader: false, task: model.Task{ID: 1}}, // 1
		{isHeader: true},                          // 2 — header, skipped
		{isHeader: false, task: model.Task{ID: 2}}, // 3
	}
	a.cursor = 1

	a2, _ := a.Update(keyMsg("j"))
	app2 := a2.(app)
	if app2.cursor != 3 {
		t.Errorf("cursor should skip header at 2, got %d", app2.cursor)
	}
}

func TestNavigation_NoWrapAtBottom(t *testing.T) {
	a := newApp("/tmp")
	a.entries = []flatEntry{
		{isHeader: false, task: model.Task{ID: 1}},
	}
	a.cursor = 0

	a2, _ := a.Update(keyMsg("j"))
	app2 := a2.(app)
	if app2.cursor != 0 {
		t.Errorf("cursor should not wrap at bottom, got %d", app2.cursor)
	}
}

func TestQuit(t *testing.T) {
	a := newApp("/tmp")
	_, cmd := a.Update(keyMsg("q"))
	if cmd == nil {
		t.Fatal("q should return a quit command")
	}
}

// helpers

func keyMsg(key string) tea.KeyMsg {
	r := []rune(key)
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: r, Alt: false}
}
