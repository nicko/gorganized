package tui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nicko/gorganized/internal/pomodoro"
)

// tickMsg is sent every second when the pomodoro timer is running.
type tickMsg time.Time

// tickCmd schedules the next tick one second from now.
func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// timerStatus formats the current timer state for the status bar.
// Returns an empty string when the timer is not running.
func timerStatus(tm *pomodoro.Timer, now time.Time) string {
	if !tm.IsRunning() {
		return ""
	}
	remaining, phase := tm.Remaining(now)
	if phase == pomodoro.PhaseWork {
		return fmt.Sprintf("[Pomodoro %d] %s remaining", tm.Count()+1, pomodoro.FormatRemaining(remaining))
	}
	return fmt.Sprintf("[Break] %s remaining", pomodoro.FormatRemaining(remaining))
}
