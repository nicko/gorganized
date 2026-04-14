package tui

import (
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

// groupForToday groups tasks for the Today view.
// Order: Active → Inactive → Todo → Done (only tasks done today).
// Empty groups are omitted.
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
			groups = append(groups, stateGroup{state: s, tasks: buckets[s]})
		}
	}
	return groups
}

// groupForAll groups tasks for the All view.
// Order: Active → Inactive → Todo → Done.
// Empty groups are omitted.
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
