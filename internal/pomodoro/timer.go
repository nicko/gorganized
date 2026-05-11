package pomodoro

import (
	"fmt"
	"time"
)

// Phase indicates whether the timer is in a work or break interval.
type Phase int

const (
	PhaseWork  Phase = iota
	PhaseBreak
)

// Timer tracks pomodoro work/break intervals.
// Call Start to begin, Tick on each clock tick, Reset to stop.
type Timer struct {
	WorkDuration  time.Duration
	BreakDuration time.Duration

	running   bool
	phase     Phase
	count     int       // completed work intervals
	startedAt time.Time
	lastTick  time.Time // last time that triggered a completion, to prevent double-fire
}

// currentDuration returns the duration for the active phase.
func (t *Timer) currentDuration() time.Duration {
	if t.phase == PhaseBreak {
		return t.BreakDuration
	}
	return t.WorkDuration
}

// String returns "work" or "break" for use in state files and display.
func (p Phase) String() string {
	if p == PhaseBreak {
		return "break"
	}
	return "work"
}

// New creates a Timer with the given work and break durations.
func New(work, breakDur time.Duration) *Timer {
	return &Timer{
		WorkDuration:  work,
		BreakDuration: breakDur,
	}
}

// Start begins the timer in the work phase at the given time.
func (t *Timer) Start(now time.Time) {
	t.running = true
	t.phase = PhaseWork
	t.startedAt = now
	t.lastTick = time.Time{}
}

// Reset stops the timer and clears all state.
func (t *Timer) Reset() {
	t.running = false
	t.phase = PhaseWork
	t.count = 0
	t.startedAt = time.Time{}
	t.lastTick = time.Time{}
}

// IsRunning reports whether the timer is active.
func (t *Timer) IsRunning() bool { return t.running }

// Count returns the number of completed work intervals.
func (t *Timer) Count() int { return t.count }

// CurrentPhase returns the current phase.
func (t *Timer) CurrentPhase() Phase { return t.phase }

// Remaining returns the remaining duration and current phase without advancing state.
func (t *Timer) Remaining(now time.Time) (remaining time.Duration, phase Phase) {
	if !t.running {
		return 0, t.phase
	}
	remaining = t.currentDuration() - now.Sub(t.startedAt)
	if remaining < 0 {
		remaining = 0
	}
	return remaining, t.phase
}

// Tick advances the timer to now.
// Returns remaining duration in the current interval, the current phase,
// and whether an interval just completed (fires once per boundary crossing).
func (t *Timer) Tick(now time.Time) (remaining time.Duration, phase Phase, justCompleted bool) {
	if !t.running {
		return 0, t.phase, false
	}

	elapsed := now.Sub(t.startedAt)
	remaining = t.currentDuration() - elapsed

	if remaining <= 0 && now != t.lastTick {
		// Interval just crossed the boundary — stop and wait for user to advance.
		t.lastTick = now
		if t.phase == PhaseWork {
			t.count++
			t.phase = PhaseBreak
		} else {
			t.phase = PhaseWork
		}
		t.running = false
		return 0, t.phase, true
	}

	if remaining < 0 {
		remaining = 0
	}
	return remaining, t.phase, false
}

// Advance starts the pending phase. Call after Tick returns justCompleted=true
// to begin the next interval when the user is ready.
func (t *Timer) Advance(now time.Time) {
	t.running = true
	t.startedAt = now
	t.lastTick = time.Time{}
}

// FormatRemaining formats a duration as "MM:SS".
func FormatRemaining(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	total := int(d.Seconds())
	m := total / 60
	s := total % 60
	return fmt.Sprintf("%02d:%02d", m, s)
}
