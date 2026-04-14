package pomodoro

import (
	"testing"
	"time"
)

var (
	work  = 25 * time.Minute
	short = 5 * time.Minute
)

func newTimer() *Timer {
	return New(work, short)
}

func TestTimer_NotRunningInitially(t *testing.T) {
	tm := newTimer()
	if tm.IsRunning() {
		t.Error("timer should not be running before Start()")
	}
}

func TestTimer_RunningAfterStart(t *testing.T) {
	tm := newTimer()
	now := time.Now()
	tm.Start(now)
	if !tm.IsRunning() {
		t.Error("timer should be running after Start()")
	}
}

func TestTimer_NotRunningAfterReset(t *testing.T) {
	tm := newTimer()
	tm.Start(time.Now())
	tm.Reset()
	if tm.IsRunning() {
		t.Error("timer should not be running after Reset()")
	}
}

func TestTimer_RemainingDuringWork(t *testing.T) {
	tm := newTimer()
	start := time.Now()
	tm.Start(start)

	elapsed := 10 * time.Minute
	remaining, phase, _ := tm.Tick(start.Add(elapsed))

	want := work - elapsed
	if remaining != want {
		t.Errorf("remaining: got %v, want %v", remaining, want)
	}
	if phase != PhaseWork {
		t.Errorf("phase: got %v, want PhaseWork", phase)
	}
}

func TestTimer_RemainingJustBeforeExpiry(t *testing.T) {
	tm := newTimer()
	start := time.Now()
	tm.Start(start)

	tick := start.Add(work - time.Second)
	remaining, phase, complete := tm.Tick(tick)

	if complete {
		t.Error("should not be complete 1s before expiry")
	}
	if phase != PhaseWork {
		t.Errorf("phase: want PhaseWork, got %v", phase)
	}
	if remaining != time.Second {
		t.Errorf("remaining: got %v, want 1s", remaining)
	}
}

func TestTimer_IntervalComplete_TransitionsToBreak(t *testing.T) {
	tm := newTimer()
	start := time.Now()
	tm.Start(start)

	// Tick at exactly the work duration boundary
	_, phase, complete := tm.Tick(start.Add(work))

	if !complete {
		t.Error("interval should be complete at work boundary")
	}
	if phase != PhaseBreak {
		t.Errorf("phase after work complete: want PhaseBreak, got %v", phase)
	}
}

func TestTimer_CountIncrementsAfterWorkInterval(t *testing.T) {
	tm := newTimer()
	start := time.Now()
	tm.Start(start)

	tm.Tick(start.Add(work)) // complete work interval

	if tm.Count() != 1 {
		t.Errorf("count after 1 work interval: got %d, want 1", tm.Count())
	}
}

func TestTimer_BreakTransitionsBackToWork(t *testing.T) {
	tm := newTimer()
	start := time.Now()
	tm.Start(start)

	tm.Tick(start.Add(work))                    // complete work → break starts
	_, phase, complete := tm.Tick(start.Add(work + short)) // complete break

	if !complete {
		t.Error("break interval should be complete")
	}
	if phase != PhaseWork {
		t.Errorf("phase after break: want PhaseWork, got %v", phase)
	}
}

func TestTimer_CompleteOnlyFiredOnce(t *testing.T) {
	tm := newTimer()
	start := time.Now()
	tm.Start(start)

	_, _, complete1 := tm.Tick(start.Add(work))
	_, _, complete2 := tm.Tick(start.Add(work + time.Second))

	if !complete1 {
		t.Error("first tick past boundary should be complete")
	}
	if complete2 {
		t.Error("subsequent ticks should not re-fire complete")
	}
}

func TestTimer_ResetClearsCount(t *testing.T) {
	tm := newTimer()
	start := time.Now()
	tm.Start(start)
	tm.Tick(start.Add(work))
	tm.Reset()

	if tm.Count() != 0 {
		t.Errorf("count after reset: got %d, want 0", tm.Count())
	}
	if tm.IsRunning() {
		t.Error("should not be running after reset")
	}
}

func TestFormatRemaining(t *testing.T) {
	cases := []struct {
		d    time.Duration
		want string
	}{
		{25*time.Minute + 0*time.Second, "25:00"},
		{18*time.Minute + 42*time.Second, "18:42"},
		{0*time.Minute + 5*time.Second, "00:05"},
	}
	for _, tc := range cases {
		got := FormatRemaining(tc.d)
		if got != tc.want {
			t.Errorf("FormatRemaining(%v) = %q, want %q", tc.d, got, tc.want)
		}
	}
}
