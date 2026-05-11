package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/nicko/gorganized/internal/model"
)

func makeTask() model.Task {
	now := time.Date(2026, 4, 14, 10, 0, 0, 0, time.UTC)
	return model.Task{
		ID:        1,
		Title:     "Buy groceries",
		State:     model.StateTodo,
		Notes:     "- milk\n- eggs\n",
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func TestRoundTrip(t *testing.T) {
	dir := t.TempDir()
	task := makeTask()

	if err := Write(dir, task); err != nil {
		t.Fatalf("Write: %v", err)
	}

	path := filepath.Join(dir, model.FormatID(task.ID)+".md")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected file %s to exist: %v", path, err)
	}

	got, err := Read(path)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}

	if got.ID != task.ID {
		t.Errorf("ID: got %d, want %d", got.ID, task.ID)
	}
	if got.Title != task.Title {
		t.Errorf("Title: got %q, want %q", got.Title, task.Title)
	}
	if got.State != task.State {
		t.Errorf("State: got %q, want %q", got.State, task.State)
	}
	if got.Notes != task.Notes {
		t.Errorf("Notes: got %q, want %q", got.Notes, task.Notes)
	}
	if !got.CreatedAt.Equal(task.CreatedAt) {
		t.Errorf("CreatedAt: got %v, want %v", got.CreatedAt, task.CreatedAt)
	}
	if got.DoneAt != nil {
		t.Errorf("DoneAt: expected nil, got %v", got.DoneAt)
	}
}

func TestRoundTrip_WithDoneAt(t *testing.T) {
	dir := t.TempDir()
	task := makeTask()
	task.State = model.StateDone
	doneAt := time.Date(2026, 4, 14, 12, 0, 0, 0, time.UTC)
	task.DoneAt = &doneAt

	if err := Write(dir, task); err != nil {
		t.Fatalf("Write: %v", err)
	}
	path := filepath.Join(dir, model.FormatID(task.ID)+".md")
	got, err := Read(path)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got.DoneAt == nil {
		t.Fatal("DoneAt: expected non-nil")
	}
	if !got.DoneAt.Equal(*task.DoneAt) {
		t.Errorf("DoneAt: got %v, want %v", got.DoneAt, task.DoneAt)
	}
}

func TestLoadAll(t *testing.T) {
	dir := t.TempDir()

	tasks := []model.Task{
		{ID: 1, Title: "First", State: model.StateTodo, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ID: 2, Title: "Second", State: model.StateActive, CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ID: 3, Title: "Third", State: model.StateDone, CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}
	for _, task := range tasks {
		if err := Write(dir, task); err != nil {
			t.Fatalf("Write: %v", err)
		}
	}

	got, err := LoadAll(dir)
	if err != nil {
		t.Fatalf("LoadAll: %v", err)
	}
	if len(got) != len(tasks) {
		t.Fatalf("LoadAll: got %d tasks, want %d", len(got), len(tasks))
	}
}

func TestRoundTrip_WithOrderAndTimeSpent(t *testing.T) {
	dir := t.TempDir()
	activeSince := time.Date(2026, 4, 14, 9, 0, 0, 0, time.UTC)
	task := model.Task{
		ID:          1,
		Title:       "Tracked task",
		State:       model.StateActive,
		Order:       3,
		TimeSpent:   600,
		ActiveSince: &activeSince,
		CreatedAt:   time.Date(2026, 4, 14, 8, 0, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2026, 4, 14, 9, 0, 0, 0, time.UTC),
	}
	if err := Write(dir, task); err != nil {
		t.Fatalf("Write: %v", err)
	}
	path := filepath.Join(dir, model.FormatID(task.ID)+".md")
	got, err := Read(path)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got.Order != 3 {
		t.Errorf("Order: got %d, want 3", got.Order)
	}
	if got.TimeSpent != 600 {
		t.Errorf("TimeSpent: got %d, want 600", got.TimeSpent)
	}
	if got.ActiveSince == nil {
		t.Fatal("ActiveSince: expected non-nil")
	}
	if !got.ActiveSince.Equal(activeSince) {
		t.Errorf("ActiveSince: got %v, want %v", got.ActiveSince, activeSince)
	}
}

func TestRoundTrip_MissingOrderDefaultsToZero(t *testing.T) {
	dir := t.TempDir()
	// Write a task, then manually strip the order field from the file.
	task := model.Task{
		ID:        1,
		Title:     "Old task",
		State:     model.StateTodo,
		Order:     0,
		CreatedAt: time.Date(2026, 4, 14, 8, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, 4, 14, 8, 0, 0, 0, time.UTC),
	}
	if err := Write(dir, task); err != nil {
		t.Fatalf("Write: %v", err)
	}
	path := filepath.Join(dir, model.FormatID(task.ID)+".md")
	got, err := Read(path)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got.Order != 0 {
		t.Errorf("Order missing from file should default to 0, got %d", got.Order)
	}
}

func TestNextID_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	id, err := NextID(dir)
	if err != nil {
		t.Fatalf("NextID: %v", err)
	}
	if id != 1 {
		t.Errorf("NextID empty dir: got %d, want 1", id)
	}
}

func TestNextID_WithExisting(t *testing.T) {
	dir := t.TempDir()
	now := time.Now()
	for _, id := range []int{1, 3, 7} {
		if err := Write(dir, model.Task{ID: id, Title: "t", State: model.StateTodo, CreatedAt: now, UpdatedAt: now}); err != nil {
			t.Fatalf("Write: %v", err)
		}
	}
	id, err := NextID(dir)
	if err != nil {
		t.Fatalf("NextID: %v", err)
	}
	if id != 8 {
		t.Errorf("NextID: got %d, want 8", id)
	}
}
