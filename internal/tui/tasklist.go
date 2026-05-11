package tui

import (
	"sort"
	"time"

	"github.com/nicko/gorganized/internal/model"
)

// stateGroup holds the tasks for one state bucket.
type stateGroup struct {
	state model.State
	tasks []model.Task
}

// flatEntry is one rendered row in the list — either a group header or a task.
type flatEntry struct {
	isHeader bool
	state    model.State // set when isHeader == true
	task     model.Task  // set when isHeader == false
}

// today returns midnight of the current day in local time.
func today() time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
}

// sortByOrder sorts tasks ascending by Order field (lower = higher priority).
func sortByOrder(tasks []model.Task) {
	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].Order < tasks[j].Order
	})
}

// sortByDoneAtDesc sorts tasks by DoneAt descending (most recently done first).
func sortByDoneAtDesc(tasks []model.Task) {
	sort.Slice(tasks, func(i, j int) bool {
		if tasks[i].DoneAt == nil {
			return false
		}
		if tasks[j].DoneAt == nil {
			return true
		}
		return tasks[i].DoneAt.After(*tasks[j].DoneAt)
	})
}

// groupForToday groups tasks for the Today view.
// Order: Active → Inactive → Todo → Done (only tasks done today).
// Empty groups are omitted. Tasks within non-done groups are sorted by Order.
func groupForToday(tasks []model.Task, midnight time.Time) []stateGroup {
	buckets := map[model.State][]model.Task{
		model.StateActive:   {},
		model.StateInactive: {},
		model.StateTodo:     {},
		model.StateDone:     {},
	}

	for _, t := range tasks {
		switch t.State {
		case model.StateDone:
			// Include only tasks completed today (after midnight).
			if t.DoneAt != nil && !t.DoneAt.Before(midnight) {
				buckets[model.StateDone] = append(buckets[model.StateDone], t)
			}
		default:
			buckets[t.State] = append(buckets[t.State], t)
		}
	}

	order := []model.State{
		model.StateActive,
		model.StateInactive,
		model.StateTodo,
		model.StateDone,
	}

	var groups []stateGroup
	for _, s := range order {
		if len(buckets[s]) > 0 {
			if s == model.StateDone {
				sortByDoneAtDesc(buckets[s])
			} else {
				sortByOrder(buckets[s])
			}
			groups = append(groups, stateGroup{state: s, tasks: buckets[s]})
		}
	}
	return groups
}

// groupForAll groups tasks for the All view.
// Order: Active → Inactive → Todo → Done.
// Empty groups are omitted. Tasks within non-done groups are sorted by Order.
func groupForAll(tasks []model.Task) []stateGroup {
	buckets := map[model.State][]model.Task{
		model.StateActive:   {},
		model.StateInactive: {},
		model.StateTodo:     {},
		model.StateDone:     {},
	}
	for _, t := range tasks {
		buckets[t.State] = append(buckets[t.State], t)
	}

	order := []model.State{
		model.StateActive,
		model.StateInactive,
		model.StateTodo,
		model.StateDone,
	}

	var groups []stateGroup
	for _, s := range order {
		if len(buckets[s]) > 0 {
			if s == model.StateDone {
				sortByDoneAtDesc(buckets[s])
			} else {
				sortByOrder(buckets[s])
			}
			groups = append(groups, stateGroup{state: s, tasks: buckets[s]})
		}
	}
	return groups
}

// flattenGroups converts groups into a flat slice of entries (headers + tasks).
func flattenGroups(groups []stateGroup) []flatEntry {
	var entries []flatEntry
	for _, g := range groups {
		entries = append(entries, flatEntry{isHeader: true, state: g.state})
		for _, t := range g.tasks {
			entries = append(entries, flatEntry{task: t})
		}
	}
	return entries
}

// headerLabel returns the display label for a state group header.
func headerLabel(s model.State) string {
	switch s {
	case model.StateActive:
		return "● Active"
	case model.StateInactive:
		return "◌ Inactive"
	case model.StateTodo:
		return "○ Todo"
	case model.StateDone:
		return "✓ Done"
	}
	return string(s)
}

// headerStyle returns the Lip Gloss style for a state group header.
func headerStyle(s model.State) interface{ Render(...string) string } {
	switch s {
	case model.StateActive:
		return styleActiveHeader
	case model.StateInactive:
		return styleInactiveHeader
	case model.StateTodo:
		return styleTodoHeader
	case model.StateDone:
		return styleDoneHeader
	}
	return styleHeader
}

// firstTaskIndex returns the index of the first non-header entry, or -1.
func firstTaskIndex(entries []flatEntry) int {
	for i, e := range entries {
		if !e.isHeader {
			return i
		}
	}
	return -1
}

// nextTaskIndex returns the next non-header entry index after cur, or cur if none.
func nextTaskIndex(entries []flatEntry, cur int) int {
	for i := cur + 1; i < len(entries); i++ {
		if !entries[i].isHeader {
			return i
		}
	}
	return cur
}

// prevTaskIndex returns the previous non-header entry index before cur, or cur if none.
func prevTaskIndex(entries []flatEntry, cur int) int {
	for i := cur - 1; i >= 0; i-- {
		if !entries[i].isHeader {
			return i
		}
	}
	return cur
}

// maxOrderForState returns the highest Order value among tasks with the given state, or 0.
func maxOrderForState(tasks []model.Task, state model.State) int {
	max := 0
	for _, t := range tasks {
		if t.State == state && t.Order > max {
			max = t.Order
		}
	}
	return max
}

// tasksInSameGroup returns all tasks sharing the same state as the entry at cursor.
// Returns nil if cursor is on a header or out of bounds.
func tasksInSameGroup(entries []flatEntry, cursor int) []model.Task {
	if cursor < 0 || cursor >= len(entries) || entries[cursor].isHeader {
		return nil
	}
	state := entries[cursor].task.State
	var group []model.Task
	for _, e := range entries {
		if !e.isHeader && e.task.State == state {
			group = append(group, e.task)
		}
	}
	return group
}

// hasDuplicateOrders returns true if any two tasks in the slice share an Order value,
// or if any task has Order == 0.
func hasDuplicateOrders(tasks []model.Task) bool {
	seen := make(map[int]bool)
	for _, t := range tasks {
		if t.Order == 0 || seen[t.Order] {
			return true
		}
		seen[t.Order] = true
	}
	return false
}

// normaliseOrders assigns sequential 1-based Order values to tasks sorted by their
// current Order (or by CreatedAt if all are zero). Returns the updated slice.
func normaliseOrders(tasks []model.Task) []model.Task {
	// Sort by existing order first; fall back to created_at for ties/zeros.
	sort.Slice(tasks, func(i, j int) bool {
		if tasks[i].Order != tasks[j].Order {
			return tasks[i].Order < tasks[j].Order
		}
		return tasks[i].CreatedAt.Before(tasks[j].CreatedAt)
	})
	for i := range tasks {
		tasks[i].Order = i + 1
	}
	return tasks
}
