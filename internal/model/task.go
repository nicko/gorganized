package model

import (
	"fmt"
	"time"
)

// State represents the lifecycle state of a task.
type State string

const (
	StateTodo     State = "todo"
	StateActive   State = "active"
	StateInactive State = "inactive"
	StateDone     State = "done"
)

// validTransitions lists every allowed state change.
var validTransitions = map[State][]State{
	StateTodo:     {StateActive},
	StateActive:   {StateInactive, StateDone},
	StateInactive: {StateActive, StateDone},
	StateDone:     {StateInactive}, // undo
}

// Task is the central domain object.
type Task struct {
	ID          int
	Title       string
	State       State
	Notes       string
	Order       int        // position within state group; lower = higher priority
	TimeSpent   int        // accumulated seconds across all active sessions
	ActiveSince *time.Time // set when state → active; cleared otherwise
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DoneAt      *time.Time
}

// Transition moves the task to the given state, enforcing allowed transitions.
// It updates time-tracking fields, UpdatedAt, DoneAt, and ActiveSince as appropriate.
func (t *Task) Transition(to State) error {
	allowed := validTransitions[t.State]
	for _, s := range allowed {
		if s == to {
			now := time.Now()

			// Leaving active: flush elapsed time into TimeSpent.
			if t.State == StateActive && t.ActiveSince != nil {
				elapsed := int(now.Sub(*t.ActiveSince).Seconds())
				if elapsed > 0 {
					t.TimeSpent += elapsed
				}
				t.ActiveSince = nil
			}

			t.State = to
			t.UpdatedAt = now

			switch to {
			case StateActive:
				t.ActiveSince = &now
			case StateDone:
				t.DoneAt = &now
			case StateInactive:
				// Clears DoneAt on undo (done → inactive).
				t.DoneAt = nil
			}

			return nil
		}
	}
	return fmt.Errorf("invalid transition: %s → %s", t.State, to)
}

// FormatID returns the zero-padded 5-digit string representation of a task ID.
func FormatID(id int) string {
	return fmt.Sprintf("%05d", id)
}
