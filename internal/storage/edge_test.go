package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/nicko/gorganized/internal/model"
)

func TestLoadAll_SkipsNonMarkdownFiles(t *testing.T) {
	dir := t.TempDir()
	now := time.Now()

	// Write one real task
	task := model.Task{ID: 1, Title: "Real task", State: model.StateTodo, CreatedAt: now, UpdatedAt: now}
	if err := Write(dir, task); err != nil {
		t.Fatal(err)
	}

	// Write some noise files that should be ignored
	for _, name := range []string{".gitkeep", "README.txt", "notes.json", ".DS_Store"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("noise"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	tasks, err := LoadAll(dir)
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if len(tasks) != 1 {
		t.Errorf("LoadAll should return 1 task, got %d", len(tasks))
	}
}

func TestNextID_SkipsNonNumericFilenames(t *testing.T) {
	dir := t.TempDir()
	now := time.Now()

	// Write a real task with ID 3
	if err := Write(dir, model.Task{ID: 3, Title: "t", State: model.StateTodo, CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatal(err)
	}

	// Drop a non-numeric .md file
	if err := os.WriteFile(filepath.Join(dir, "notes.md"), []byte("# notes"), 0o644); err != nil {
		t.Fatal(err)
	}

	id, err := NextID(dir)
	if err != nil {
		t.Fatalf("NextID: %v", err)
	}
	// Should skip "notes.md" and return max(3)+1 = 4
	if id != 4 {
		t.Errorf("NextID: got %d, want 4", id)
	}
}

func TestWrite_UpdatesExistingFile(t *testing.T) {
	dir := t.TempDir()
	now := time.Now()

	task := model.Task{ID: 1, Title: "Original", State: model.StateTodo, CreatedAt: now, UpdatedAt: now}
	if err := Write(dir, task); err != nil {
		t.Fatal(err)
	}

	// Update the task
	task.Title = "Updated"
	task.State = model.StateActive
	task.Notes = "some notes"
	if err := Write(dir, task); err != nil {
		t.Fatalf("Write (update): %v", err)
	}

	// Read back
	got, err := Read(filepath.Join(dir, "00001.md"))
	if err != nil {
		t.Fatalf("Read after update: %v", err)
	}
	if got.Title != "Updated" {
		t.Errorf("Title after update: got %q, want %q", got.Title, "Updated")
	}
	if got.State != model.StateActive {
		t.Errorf("State after update: got %q, want %q", got.State, model.StateActive)
	}
	if got.Notes != "some notes" {
		t.Errorf("Notes after update: got %q, want %q", got.Notes, "some notes")
	}
}

func TestRead_MissingFrontmatter(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.md")
	if err := os.WriteFile(path, []byte("no frontmatter here"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Read(path)
	if err == nil {
		t.Error("Read with missing frontmatter should return error")
	}
}

func TestRead_NonexistentFile(t *testing.T) {
	_, err := Read("/tmp/does-not-exist-gorgan.md")
	if err == nil {
		t.Error("Read of nonexistent file should return error")
	}
}
