package tui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nicko/gorganized/internal/model"
	"github.com/nicko/gorganized/internal/timerstate"
)

func (a app) applyTimerForState(state model.State) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	if state == model.StateActive {
		a.timer.Start(time.Now())
		cmd = tickCmd()
	} else {
		a.timer.Reset()
	}
	a.timerAlert = false
	a.writeTimerState()
	return a, cmd
}

func (a app) writeTimerState() {
	remaining, _ := a.timer.Remaining(time.Now())
	s := timerstate.State{
		Running:   a.timer.IsRunning(),
		Phase:     a.timer.CurrentPhase().String(),
		Pomodoro:  a.timer.Count() + 1,
		Remaining: int(remaining.Seconds()),
		Alert:     a.timerAlert,
	}
	_ = timerstate.Write(timerstate.FilePath(a.gorganDir), s)
}

func formatDuration(seconds int) string {
	if seconds <= 0 {
		return "< 1m"
	}
	h := seconds / 3600
	m := (seconds % 3600) / 60
	if h > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	if m == 0 {
		return "< 1m"
	}
	return fmt.Sprintf("%dm", m)
}
