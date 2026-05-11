package model

import (
	"testing"
	"time"
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
		{StateDone, StateDone},
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

func TestTransition_DoneToInactive(t *testing.T) {
	now := time.Now()
	task := Task{State: StateDone, DoneAt: &now}
	if err := task.Transition(StateInactive); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if task.State != StateInactive {
		t.Errorf("state: got %s, want inactive", task.State)
	}
	if task.DoneAt != nil {
		t.Error("DoneAt should be nil after undo")
	}
}

func TestTransition_DoneToInactive_PreservesTimeSpent(t *testing.T) {
	now := time.Now()
	task := Task{State: StateDone, TimeSpent: 500, DoneAt: &now}
	if err := task.Transition(StateInactive); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if task.TimeSpent != 500 {
		t.Errorf("TimeSpent should be preserved on undo: got %d, want 500", task.TimeSpent)
	}
}

func TestTransition_DoneIsStillInvalid(t *testing.T) {
	invalid := []State{StateTodo, StateActive, StateDone}
	for _, to := range invalid {
		task := Task{State: StateDone}
		if err := task.Transition(to); err == nil {
			t.Errorf("Transition(done → %s): expected error, got nil", to)
		}
	}
}

func TestTransition_ActiveSince_SetOnActive(t *testing.T) {
	task := Task{State: StateTodo}
	if err := task.Transition(StateActive); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if task.ActiveSince == nil {
		t.Error("ActiveSince should be set when transitioning to active")
	}
}

func TestTransition_ActiveSince_ClearedOnInactive(t *testing.T) {
	now := time.Now()
	task := Task{State: StateActive, ActiveSince: &now}
	if err := task.Transition(StateInactive); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if task.ActiveSince != nil {
		t.Error("ActiveSince should be cleared when transitioning to inactive")
	}
}

func TestTransition_TimeSpent_AccumulatesOnInactive(t *testing.T) {
	fiveMinAgo := time.Now().Add(-5 * time.Minute)
	task := Task{State: StateActive, TimeSpent: 100, ActiveSince: &fiveMinAgo}
	if err := task.Transition(StateInactive); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// 100 base + ~300s elapsed; accept 395–430 to allow for slow CI
	if task.TimeSpent < 395 {
		t.Errorf("TimeSpent: got %d, want >= 395 (100 base + ~300 elapsed)", task.TimeSpent)
	}
	if task.TimeSpent > 430 {
		t.Errorf("TimeSpent: got %d, seems too large", task.TimeSpent)
	}
}

func TestTransition_TimeSpent_AccumulatesOnDone(t *testing.T) {
	threeMinAgo := time.Now().Add(-3 * time.Minute)
	task := Task{State: StateActive, TimeSpent: 0, ActiveSince: &threeMinAgo}
	if err := task.Transition(StateDone); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if task.TimeSpent < 175 { // ~180 - 5s tolerance
		t.Errorf("TimeSpent: got %d, want >= 175", task.TimeSpent)
	}
	if task.ActiveSince != nil {
		t.Error("ActiveSince should be cleared when transitioning to done")
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
