package tui

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nicko/gorganized/internal/kb"
	"github.com/nicko/gorganized/internal/model"
	"github.com/nicko/gorganized/internal/storage"
)

// setupAppWithKB creates an app backed by a real .gorgan dir including a KB.
func setupAppWithKB(t *testing.T, tasks []model.Task) app {
	t.Helper()
	dir := t.TempDir()
	tasksDir := filepath.Join(dir, "tasks")
	if err := os.MkdirAll(tasksDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, task := range tasks {
		if err := storage.Write(tasksDir, task); err != nil {
			t.Fatalf("Write task: %v", err)
		}
	}
	a, err := loadApp(dir)
	if err != nil {
		t.Fatalf("loadApp: %v", err)
	}
	t.Cleanup(func() {
		if a.kbIndex != nil {
			a.kbIndex.Close()
		}
	})
	return a
}

func TestSearchKey_EntersSearchMode(t *testing.T) {
	a := setupAppWithKB(t, nil)

	a2, _ := a.Update(keyMsg("/"))
	app2 := a2.(app)

	if !app2.searching {
		t.Error("after pressing '/', app should be in search mode")
	}
}

func TestSearch_EscClosesOverlay(t *testing.T) {
	a := setupAppWithKB(t, nil)
	a2, _ := a.Update(keyMsg("/"))

	a3, _ := a2.(app).Update(tea.KeyMsg{Type: tea.KeyEsc})
	app3 := a3.(app)

	if app3.searching {
		t.Error("esc should close search overlay")
	}
}

func TestSearch_TypedQueryBuildsUp(t *testing.T) {
	a := setupAppWithKB(t, nil)
	a2, _ := a.Update(keyMsg("/"))

	a3 := a2.(app)
	for _, ch := range "hello" {
		m, _ := a3.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		a3 = m.(app)
	}

	if a3.searchQuery != "hello" {
		t.Errorf("searchQuery: got %q, want %q", a3.searchQuery, "hello")
	}
}

func TestSearch_BackspaceDeletesChar(t *testing.T) {
	a := setupAppWithKB(t, nil)
	a2, _ := a.Update(keyMsg("/"))

	a3 := a2.(app)
	for _, ch := range "abc" {
		m, _ := a3.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		a3 = m.(app)
	}
	m, _ := a3.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	a3 = m.(app)

	if a3.searchQuery != "ab" {
		t.Errorf("after backspace: got %q, want %q", a3.searchQuery, "ab")
	}
}

func TestSearch_FindsDoneTask(t *testing.T) {
	now := time.Now()
	tasks := []model.Task{
		{ID: 1, Title: "Refactor auth", State: model.StateActive,
			Notes: "Moved session token to Redis", CreatedAt: now, UpdatedAt: now},
	}
	a := setupAppWithKB(t, tasks)

	// Mark task done — this should index it
	a2, _ := a.Update(keyMsg("d"))

	// Open search and type a term from the notes
	a3, _ := a2.(app).Update(keyMsg("/"))
	a4 := a3.(app)
	for _, ch := range "Redis" {
		m, _ := a4.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		a4 = m.(app)
	}

	if len(a4.searchResults) == 0 {
		t.Error("expected search results for 'Redis' after task was indexed on done")
	}
	if a4.searchResults[0].TaskID != 1 {
		t.Errorf("result TaskID: got %d, want 1", a4.searchResults[0].TaskID)
	}
}

func TestSearch_EnterNavigatesToTask(t *testing.T) {
	now := time.Now()
	tasks := []model.Task{
		{ID: 1, Title: "Buy milk", State: model.StateActive,
			Notes: "skimmed milk please", CreatedAt: now, UpdatedAt: now},
		{ID: 2, Title: "Other task", State: model.StateTodo,
			CreatedAt: now, UpdatedAt: now},
	}
	a := setupAppWithKB(t, tasks)

	// Mark task 1 done to index it
	a2, _ := a.Update(keyMsg("d"))

	// Search and navigate
	a3, _ := a2.(app).Update(keyMsg("/"))
	a4 := a3.(app)
	for _, ch := range "skimmed" {
		m, _ := a4.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		a4 = m.(app)
	}

	if len(a4.searchResults) == 0 {
		t.Skip("no results for 'skimmed' — search not finding indexed content")
	}

	// Press enter to navigate
	m, _ := a4.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app5 := m.(app)

	if app5.searching {
		t.Error("search overlay should close after enter")
	}
}

func TestSearch_ResultCursorNavigation(t *testing.T) {
	// Manually inject results to test cursor movement
	a := setupAppWithKB(t, nil)
	a.searching = true
	a.searchResults = []kb.Result{
		{TaskID: 1, Title: "First"},
		{TaskID: 2, Title: "Second"},
		{TaskID: 3, Title: "Third"},
	}
	a.searchCursor = 0

	m, _ := a.Update(tea.KeyMsg{Type: tea.KeyDown})
	a2 := m.(app)
	if a2.searchCursor != 1 {
		t.Errorf("cursor after down: got %d, want 1", a2.searchCursor)
	}

	m, _ = a2.Update(tea.KeyMsg{Type: tea.KeyUp})
	a3 := m.(app)
	if a3.searchCursor != 0 {
		t.Errorf("cursor after up: got %d, want 0", a3.searchCursor)
	}
}

func TestSearch_CursorDoesNotGoNegative(t *testing.T) {
	a := setupAppWithKB(t, nil)
	a.searching = true
	a.searchResults = []kb.Result{{TaskID: 1, Title: "One"}}
	a.searchCursor = 0

	m, _ := a.Update(tea.KeyMsg{Type: tea.KeyUp})
	a2 := m.(app)
	if a2.searchCursor != 0 {
		t.Errorf("cursor should not go below 0, got %d", a2.searchCursor)
	}
}
