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
	StateDone:     {},
}

// Task is the central domain object.
type Task struct {
	ID        int
	Title     string
	State     State
	Notes     string
	CreatedAt time.Time
	UpdatedAt time.Time
	DoneAt    *time.Time
}

// Transition moves the task to the given state, enforcing allowed transitions.
// It updates UpdatedAt (and DoneAt when transitioning to done) on success.
func (t *Task) Transition(to State) error {
	allowed := validTransitions[t.State]
	for _, s := range allowed {
		if s == to {
			now := time.Now()
			t.State = to
			t.UpdatedAt = now
			if to == StateDone {
				t.DoneAt = &now
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
