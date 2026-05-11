package timerstate

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// State is the timer snapshot written by the TUI and read by gorgan-bar.
type State struct {
	Running   bool   `json:"running"`
	Phase     string `json:"phase"`     // "work" or "break"
	Pomodoro  int    `json:"pomodoro"`  // Count()+1, meaningful during work
	Remaining int    `json:"remaining"` // seconds remaining
	Alert     bool   `json:"alert"`     // paused at phase boundary, waiting for user
}

// Write atomically writes s to path via a temp file and rename.
func Write(path string, s State) error {
	data, err := json.Marshal(s)
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// Read reads State from path. Returns a zero State (Running=false) if the
// file does not exist.
func Read(path string) (State, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return State{}, nil
	}
	if err != nil {
		return State{}, err
	}
	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		return State{}, err
	}
	return s, nil
}

// Clear writes a stopped state to path, signalling that gorgan has exited.
func Clear(path string) error {
	return Write(path, State{})
}

// FilePath returns the canonical timer state file path for a .gorgan directory.
func FilePath(gorganDir string) string {
	return filepath.Join(gorganDir, "timer.state")
}
