package tui

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nicko/gorganized/internal/model"
	"github.com/nicko/gorganized/internal/storage"
)

func makeTaskWithNotes(id int, notes string) model.Task {
	now := time.Now()
	return model.Task{
		ID:        id,
		Title:     "Task with notes",
		State:     model.StateTodo,
		Notes:     notes,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func TestNoteKey_EntersEditingMode(t *testing.T) {
	tasks := []model.Task{makeTaskWithNotes(1, "existing notes")}
	a := setupApp(t, tasks)

	a2, _ := a.Update(keyMsg("n"))
	app2 := a2.(app)

	if !app2.editingNote {
		t.Error("after pressing 'n', app should be in note editing mode")
	}
}

func TestNoteEditor_PrefilledWithExistingNotes(t *testing.T) {
	notes := "- item one\n- item two\n"
	tasks := []model.Task{makeTaskWithNotes(1, notes)}
	a := setupApp(t, tasks)

	a2, _ := a.Update(keyMsg("n"))
	app2 := a2.(app)

	if app2.noteEditor.Value() != notes {
		t.Errorf("note editor content: got %q, want %q", app2.noteEditor.Value(), notes)
	}
}

func TestNoteEditor_EscSavesAndCloses(t *testing.T) {
	dir := t.TempDir()
	tasksDir := filepath.Join(dir, "tasks")
	if err := os.MkdirAll(tasksDir, 0o755); err != nil {
		t.Fatal(err)
	}

	task := makeTaskWithNotes(1, "original notes")
	if err := storage.Write(tasksDir, task); err != nil {
		t.Fatal(err)
	}

	a, err := loadApp(dir)
	if err != nil {
		t.Fatal(err)
	}

	// Open notes
	a2, _ := a.Update(keyMsg("n"))

	// Type additional content
	a3 := a2.(app)
	// Set the value directly via the textarea's SetValue — we test the save path
	// by sending esc which triggers the save
	a3.noteEditor.SetValue("updated notes content")

	// Close with esc
	a4, _ := a3.Update(tea.KeyMsg{Type: tea.KeyEsc})
	app4 := a4.(app)

	if app4.editingNote {
		t.Error("after esc, note editing should be closed")
	}

	// Notes should be updated in memory
	found := false
	for _, tk := range app4.tasks {
		if tk.ID == 1 {
			found = true
			if tk.Notes != "updated notes content" {
				t.Errorf("in-memory notes: got %q, want %q", tk.Notes, "updated notes content")
			}
		}
	}
	if !found {
		t.Fatal("task not found after note save")
	}

	// Notes should be persisted to disk
	saved, err := storage.Read(filepath.Join(tasksDir, "00001.md"))
	if err != nil {
		t.Fatalf("Read after save: %v", err)
	}
	if saved.Notes != "updated notes content" {
		t.Errorf("disk notes: got %q, want %q", saved.Notes, "updated notes content")
	}
}

func TestNoteEditor_NoTaskSelected_DoesNothing(t *testing.T) {
	// App with no tasks — no task to open notes for
	a := setupApp(t, nil)

	a2, _ := a.Update(keyMsg("n"))
	app2 := a2.(app)

	if app2.editingNote {
		t.Error("n with no task selected should not enter editing mode")
	}
}

func TestNoteEditor_EscOnEmptyNotesStillSaves(t *testing.T) {
	dir := t.TempDir()
	tasksDir := filepath.Join(dir, "tasks")
	if err := os.MkdirAll(tasksDir, 0o755); err != nil {
		t.Fatal(err)
	}

	task := makeTaskWithNotes(1, "some notes")
	if err := storage.Write(tasksDir, task); err != nil {
		t.Fatal(err)
	}

	a, _ := loadApp(dir)
	a2, _ := a.Update(keyMsg("n"))

	a3 := a2.(app)
	a3.noteEditor.SetValue("") // clear notes

	a4, _ := a3.Update(tea.KeyMsg{Type: tea.KeyEsc})
	app4 := a4.(app)

	for _, tk := range app4.tasks {
		if tk.ID == 1 && tk.Notes != "" {
			t.Errorf("expected empty notes after clearing, got %q", tk.Notes)
		}
	}
}

func TestNoteEditor_OtherKeysForwardedToTextarea(t *testing.T) {
	tasks := []model.Task{makeTaskWithNotes(1, "")}
	a := setupApp(t, tasks)

	a2, _ := a.Update(keyMsg("n"))
	app2 := a2.(app)

	// Type a character — should go to the textarea, not trigger list navigation
	m, _ := app2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	app3 := m.(app)

	if !app3.editingNote {
		t.Error("should still be in editing mode after typing")
	}
}
