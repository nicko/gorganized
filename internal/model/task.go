package model

import "time"

// State represents the lifecycle state of a task.
type State string

const (
	StateTodo     State = "todo"
	StateActive   State = "active"
	StateInactive State = "inactive"
	StateDone     State = "done"
)

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
