package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nicko/gorganized/internal/model"
)

// --- Undo done (done → inactive) ---

func TestUKey_DoneBecomesInactive(t *testing.T) {
	now := time.Now()
	tasks := []model.Task{
		{ID: 1, Title: "Finished", State: model.StateDone, DoneAt: &now,
			CreatedAt: now, UpdatedAt: now},
	}
	a := setupAppClean(t, tasks)

	// Switch to All view so done tasks are visible.
	a2, _ := a.Update(tea.KeyMsg{Type: tea.KeyTab})
	app2 := a2.(app)

	// Position cursor on the done task.
	for i, e := range app2.entries {
		if !e.isHeader && e.task.ID == 1 {
			app2.cursor = i
			break
		}
	}

	a3, _ := app2.Update(tea.KeyMsg{Type: tea.KeySpace})
	app3 := a3.(app)

	for _, tk := range app3.tasks {
		if tk.ID == 1 {
			if tk.State != model.StateInactive {
				t.Errorf("after space on done: state = %s, want inactive", tk.State)
			}
			if tk.DoneAt != nil {
				t.Error("DoneAt should be cleared after undo")
			}
			return
		}
	}
	t.Fatal("task 1 not found")
}

func TestEnterKey_DoneBecomesInactive(t *testing.T) {
	now := time.Now()
	tasks := []model.Task{
		{ID: 1, Title: "Finished", State: model.StateDone, DoneAt: &now,
			CreatedAt: now, UpdatedAt: now},
	}
	a := setupAppClean(t, tasks)

	a2, _ := a.Update(tea.KeyMsg{Type: tea.KeyTab})
	app2 := a2.(app)
	for i, e := range app2.entries {
		if !e.isHeader && e.task.ID == 1 {
			app2.cursor = i
			break
		}
	}

	a3, _ := app2.Update(tea.KeyMsg{Type: tea.KeySpace})
	app3 := a3.(app)

	for _, tk := range app3.tasks {
		if tk.ID == 1 && tk.State != model.StateInactive {
			t.Errorf("space on done task should undo to inactive; got %s", tk.State)
		}
	}
}

// --- Reorder: ctrl+up / ctrl+down ---

func TestReorder_MoveUp_SwapsOrder(t *testing.T) {
	now := time.Now()
	tasks := []model.Task{
		{ID: 1, Title: "First", State: model.StateTodo, Order: 1, CreatedAt: now, UpdatedAt: now},
		{ID: 2, Title: "Second", State: model.StateTodo, Order: 2, CreatedAt: now, UpdatedAt: now},
	}
	a := setupAppClean(t, tasks)

	// Find entry index of task 2 (second todo).
	for i, e := range a.entries {
		if !e.isHeader && e.task.ID == 2 {
			a.cursor = i
			break
		}
	}

	a2, _ := a.Update(tea.KeyMsg{Type: tea.KeyShiftUp})
	app2 := a2.(app)

	orderOf := func(id int) int {
		for _, tk := range app2.tasks {
			if tk.ID == id {
				return tk.Order
			}
		}
		return -1
	}
	if orderOf(2) >= orderOf(1) {
		t.Errorf("after ctrl+up: task 2 order (%d) should be less than task 1 order (%d)",
			orderOf(2), orderOf(1))
	}
}

func TestReorder_MoveDown_SwapsOrder(t *testing.T) {
	now := time.Now()
	tasks := []model.Task{
		{ID: 1, Title: "First", State: model.StateTodo, Order: 1, CreatedAt: now, UpdatedAt: now},
		{ID: 2, Title: "Second", State: model.StateTodo, Order: 2, CreatedAt: now, UpdatedAt: now},
	}
	a := setupAppClean(t, tasks)

	// Find task 1 in entries.
	for i, e := range a.entries {
		if !e.isHeader && e.task.ID == 1 {
			a.cursor = i
			break
		}
	}

	a2, _ := a.Update(tea.KeyMsg{Type: tea.KeyShiftDown})
	app2 := a2.(app)

	orderOf := func(id int) int {
		for _, tk := range app2.tasks {
			if tk.ID == id {
				return tk.Order
			}
		}
		return -1
	}
	if orderOf(1) <= orderOf(2) {
		t.Errorf("after ctrl+down: task 1 order (%d) should be greater than task 2 order (%d)",
			orderOf(1), orderOf(2))
	}
}

func TestReorder_MoveUpAtTop_NoOp(t *testing.T) {
	now := time.Now()
	tasks := []model.Task{
		{ID: 1, Title: "Only", State: model.StateTodo, Order: 1, CreatedAt: now, UpdatedAt: now},
	}
	a := setupAppClean(t, tasks)
	for i, e := range a.entries {
		if !e.isHeader && e.task.ID == 1 {
			a.cursor = i
			break
		}
	}
	origOrder := a.tasks[0].Order
	a2, _ := a.Update(tea.KeyMsg{Type: tea.KeyShiftUp})
	app2 := a2.(app)
	for _, tk := range app2.tasks {
		if tk.ID == 1 && tk.Order != origOrder {
			t.Errorf("ctrl+up at top should be no-op; order changed from %d to %d", origOrder, tk.Order)
		}
	}
}

func TestReorder_CrossesGroupBoundary_NoOp(t *testing.T) {
	now := time.Now()
	tasks := []model.Task{
		{ID: 1, Title: "Active one", State: model.StateActive, Order: 1, CreatedAt: now, UpdatedAt: now},
		{ID: 2, Title: "Todo one", State: model.StateTodo, Order: 1, CreatedAt: now, UpdatedAt: now},
	}
	a := setupAppClean(t, tasks)

	// Move cursor to the todo task.
	for i, e := range a.entries {
		if !e.isHeader && e.task.ID == 2 {
			a.cursor = i
			break
		}
	}

	a2, _ := a.Update(tea.KeyMsg{Type: tea.KeyShiftUp})
	app2 := a2.(app)

	for _, tk := range app2.tasks {
		if tk.ID == 2 && tk.State != model.StateTodo {
			t.Errorf("ctrl+up should not cross group boundary; task 2 state = %s", tk.State)
		}
	}
}

// --- Preview toggle ---

func TestPreviewToggle(t *testing.T) {
	tasks := []model.Task{
		{ID: 1, Title: "A", State: model.StateTodo, CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}
	a := setupAppClean(t, tasks)

	if !a.previewVisible {
		t.Fatal("preview should be visible by default")
	}

	a2, _ := a.Update(keyMsg("p"))
	if a2.(app).previewVisible {
		t.Error("preview should be hidden after p")
	}

	a3, _ := a2.(app).Update(keyMsg("p"))
	if !a3.(app).previewVisible {
		t.Error("preview should be visible after second p")
	}
}

// --- Order normalisation on load ---

func TestLoadApp_NormalisesTasksWithNoOrder(t *testing.T) {
	// Tasks created without order field (Order == 0).
	now := time.Now()
	tasks := []model.Task{
		{ID: 1, Title: "A", State: model.StateTodo, Order: 0, CreatedAt: now, UpdatedAt: now},
		{ID: 2, Title: "B", State: model.StateTodo, Order: 0, CreatedAt: now.Add(time.Second), UpdatedAt: now},
	}
	a := setupAppClean(t, tasks)

	for _, tk := range a.tasks {
		if tk.Order == 0 {
			t.Errorf("task %d should have order assigned after load, got 0", tk.ID)
		}
	}
}

// --- Order assigned on new task ---

func TestAddTask_GetsOrderBeyondExisting(t *testing.T) {
	now := time.Now()
	tasks := []model.Task{
		{ID: 1, Title: "Existing", State: model.StateTodo, Order: 5, CreatedAt: now, UpdatedAt: now},
	}
	a := setupAppClean(t, tasks)

	// Add a new task.
	a2, _ := a.Update(keyMsg("n"))
	// Type a title.
	for _, r := range "New task" {
		a3, _ := a2.(app).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		a2 = a3
	}
	a4, _ := a2.(app).Update(tea.KeyMsg{Type: tea.KeyEnter})
	app4 := a4.(app)

	for _, tk := range app4.tasks {
		if tk.Title == "New task" && tk.Order <= 5 {
			t.Errorf("new task order (%d) should be > 5 (max existing)", tk.Order)
		}
	}
}

// --- State transition assigns order ---

func TestStateTransition_AssignsOrderInNewGroup(t *testing.T) {
	now := time.Now()
	tasks := []model.Task{
		{ID: 1, Title: "Task", State: model.StateTodo, Order: 3, CreatedAt: now, UpdatedAt: now},
	}
	a := setupAppClean(t, tasks)

	// todo → active
	a2, _ := a.Update(tea.KeyMsg{Type: tea.KeySpace})
	app2 := a2.(app)

	for _, tk := range app2.tasks {
		if tk.ID == 1 && tk.State == model.StateActive && tk.Order == 0 {
			t.Error("task should have order assigned after transitioning to active group")
		}
	}
}
