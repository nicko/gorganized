package pomodoro

import (
	"testing"
	"time"
)

func TestRemaining_NotRunning(t *testing.T) {
	tm := New(25*time.Minute, 5*time.Minute)
	remaining, phase := tm.Remaining(time.Now())
	if remaining != 0 {
		t.Errorf("Remaining when not running: got %v, want 0", remaining)
	}
	if phase != PhaseWork {
		t.Errorf("Phase when not running: got %v, want PhaseWork", phase)
	}
}

func TestRemaining_DuringWork(t *testing.T) {
	tm := New(25*time.Minute, 5*time.Minute)
	start := time.Now()
	tm.Start(start)

	elapsed := 10 * time.Minute
	remaining, phase := tm.Remaining(start.Add(elapsed))

	want := 25*time.Minute - elapsed
	if remaining != want {
		t.Errorf("Remaining during work: got %v, want %v", remaining, want)
	}
	if phase != PhaseWork {
		t.Errorf("Phase during work: got %v, want PhaseWork", phase)
	}
}

func TestRemaining_DuringBreak(t *testing.T) {
	tm := New(25*time.Minute, 5*time.Minute)
	start := time.Now()
	tm.Start(start)

	// Advance past the work interval to trigger a transition, then user advances to break
	boundary := start.Add(25 * time.Minute)
	tm.Tick(boundary)
	tm.Advance(boundary) // simulate user pressing space to start break

	// Now in break phase; check remaining
	elapsed := 2 * time.Minute
	remaining, phase := tm.Remaining(boundary.Add(elapsed))

	want := 5*time.Minute - elapsed
	if remaining != want {
		t.Errorf("Remaining during break: got %v, want %v", remaining, want)
	}
	if phase != PhaseBreak {
		t.Errorf("Phase during break: got %v, want PhaseBreak", phase)
	}
}

func TestRemaining_ClampedAtZero(t *testing.T) {
	tm := New(25*time.Minute, 5*time.Minute)
	start := time.Now()
	tm.Start(start)

	// Way past the work interval without ticking (so no phase transition yet)
	remaining, _ := tm.Remaining(start.Add(60 * time.Minute))
	if remaining != 0 {
		t.Errorf("Remaining past end (no tick): got %v, want 0", remaining)
	}
}

func TestCurrentPhase_InitiallyWork(t *testing.T) {
	tm := New(25*time.Minute, 5*time.Minute)
	if tm.CurrentPhase() != PhaseWork {
		t.Errorf("initial phase: got %v, want PhaseWork", tm.CurrentPhase())
	}
}

func TestCurrentPhase_AfterWorkComplete(t *testing.T) {
	tm := New(25*time.Minute, 5*time.Minute)
	start := time.Now()
	tm.Start(start)
	tm.Tick(start.Add(25 * time.Minute))

	if tm.CurrentPhase() != PhaseBreak {
		t.Errorf("phase after work complete: got %v, want PhaseBreak", tm.CurrentPhase())
	}
}

func TestFormatRemaining_Zero(t *testing.T) {
	got := FormatRemaining(0)
	if got != "00:00" {
		t.Errorf("FormatRemaining(0): got %q, want %q", got, "00:00")
	}
}

func TestFormatRemaining_Negative(t *testing.T) {
	got := FormatRemaining(-5 * time.Second)
	if got != "00:00" {
		t.Errorf("FormatRemaining(negative): got %q, want %q", got, "00:00")
	}
}
