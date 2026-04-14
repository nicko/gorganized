package model

import (
	"testing"
)

func TestTransition_ValidPaths(t *testing.T) {
	cases := []struct {
		from State
		to   State
	}{
		{StateTodo, StateActive},
		{StateActive, StateInactive},
		{StateActive, StateDone},
		{StateInactive, StateActive},
		{StateInactive, StateDone},
	}
	for _, tc := range cases {
		task := Task{State: tc.from}
		if err := task.Transition(tc.to); err != nil {
			t.Errorf("Transition(%s → %s) unexpected error: %v", tc.from, tc.to, err)
		}
		if task.State != tc.to {
			t.Errorf("Transition(%s → %s): state is %s", tc.from, tc.to, task.State)
		}
	}
}

func TestTransition_InvalidPaths(t *testing.T) {
	cases := []struct {
		from State
		to   State
	}{
		{StateTodo, StateInactive},
		{StateTodo, StateDone},
		{StateDone, StateTodo},
		{StateDone, StateActive},
		{StateDone, StateInactive},
	}
	for _, tc := range cases {
		task := Task{State: tc.from}
		if err := task.Transition(tc.to); err == nil {
			t.Errorf("Transition(%s → %s): expected error, got nil", tc.from, tc.to)
		}
		// State must not change on invalid transition
		if task.State != tc.from {
			t.Errorf("Transition(%s → %s): state mutated to %s on invalid transition", tc.from, tc.to, task.State)
		}
	}
}

func TestTransition_DoneAt(t *testing.T) {
	task := Task{State: StateActive}
	if err := task.Transition(StateDone); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if task.DoneAt == nil {
		t.Error("DoneAt should be set when transitioning to done")
	}
	if task.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should be set on transition")
	}
}

func TestTransition_UpdatedAt(t *testing.T) {
	task := Task{State: StateTodo}
	if err := task.Transition(StateActive); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if task.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should be set on transition")
	}
}

func TestFormatID(t *testing.T) {
	cases := []struct {
		id       int
		expected string
	}{
		{1, "00001"},
		{42, "00042"},
		{99999, "99999"},
	}
	for _, tc := range cases {
		got := FormatID(tc.id)
		if got != tc.expected {
			t.Errorf("FormatID(%d) = %q, want %q", tc.id, got, tc.expected)
		}
	}
}
