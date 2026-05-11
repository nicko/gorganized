package kb

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/nicko/gorganized/internal/model"
)

func openTestIndex(t *testing.T) *Index {
	t.Helper()
	dir := t.TempDir()
	idx, err := Open(filepath.Join(dir, "kb.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { idx.Close() })
	return idx
}

func makeTask(id int, title, notes string) model.Task {
	now := time.Now()
	return model.Task{
		ID:        id,
		Title:     title,
		State:     model.StateDone,
		Notes:     notes,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func TestOpen_CreatesDatabase(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "kb.db")

	idx, err := Open(dbPath)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer idx.Close()

	if _, err := os.Stat(dbPath); err != nil {
		t.Errorf("db file should exist after Open: %v", err)
	}
}

func TestIndex_SearchFindsIndexedTask(t *testing.T) {
	idx := openTestIndex(t)
	task := makeTask(1, "Buy groceries", "- milk\n- eggs\n- bread\n")

	if err := idx.Index(task); err != nil {
		t.Fatalf("Index: %v", err)
	}

	results, err := idx.Search("milk")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected at least one result for 'milk'")
	}
	if results[0].TaskID != 1 {
		t.Errorf("result TaskID: got %d, want 1", results[0].TaskID)
	}
	if results[0].Title != "Buy groceries" {
		t.Errorf("result Title: got %q, want 'Buy groceries'", results[0].Title)
	}
}

func TestIndex_SearchByTitle(t *testing.T) {
	idx := openTestIndex(t)
	task := makeTask(2, "Design sprint", "Reviewed wireframes")

	if err := idx.Index(task); err != nil {
		t.Fatalf("Index: %v", err)
	}

	results, err := idx.Search("sprint")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected result for title word 'sprint'")
	}
	if results[0].TaskID != 2 {
		t.Errorf("result TaskID: got %d, want 2", results[0].TaskID)
	}
}

func TestIndex_NoResultsForUnknownTerm(t *testing.T) {
	idx := openTestIndex(t)
	if err := idx.Index(makeTask(1, "Something", "Some notes")); err != nil {
		t.Fatal(err)
	}

	results, err := idx.Search("zzznomatch")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected no results, got %d", len(results))
	}
}

func TestIndex_MultipleTasksSearchable(t *testing.T) {
	idx := openTestIndex(t)

	tasks := []model.Task{
		makeTask(1, "Task one", "alpha beta gamma"),
		makeTask(2, "Task two", "delta epsilon"),
		makeTask(3, "Task three", "beta zeta"),
	}
	for _, task := range tasks {
		if err := idx.Index(task); err != nil {
			t.Fatalf("Index: %v", err)
		}
	}

	results, err := idx.Search("beta")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results for 'beta', got %d", len(results))
	}
}

func TestIndex_UpsertUpdatesExistingEntry(t *testing.T) {
	idx := openTestIndex(t)

	task := makeTask(1, "Original title", "original notes")
	if err := idx.Index(task); err != nil {
		t.Fatal(err)
	}

	// Re-index with updated content
	task.Title = "Updated title"
	task.Notes = "completely different content"
	if err := idx.Index(task); err != nil {
		t.Fatal(err)
	}

	// Old term should no longer match
	results, err := idx.Search("original")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 0 {
		t.Errorf("after upsert, old content should not match; got %d results", len(results))
	}

	// New term should match
	results, err = idx.Search("different")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Errorf("after upsert, new content should match; got %d results", len(results))
	}
}

func TestIndex_EmptyQueryReturnsNothing(t *testing.T) {
	idx := openTestIndex(t)
	if err := idx.Index(makeTask(1, "Title", "Notes")); err != nil {
		t.Fatal(err)
	}

	results, err := idx.Search("")
	if err != nil {
		t.Fatalf("Search empty: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("empty query should return no results, got %d", len(results))
	}
}

func TestRemove_TaskNoLongerSearchable(t *testing.T) {
	idx := openTestIndex(t)
	task := makeTask(1, "Redis migration", "Moved session tokens to Redis")
	if err := idx.Index(task); err != nil {
		t.Fatal(err)
	}
	// Confirm it's findable.
	results, err := idx.Search("Redis")
	if err != nil || len(results) == 0 {
		t.Fatalf("task should be searchable before Remove: %v, %v", err, results)
	}
	// Remove it.
	if err := idx.Remove(1); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	// Confirm it's gone.
	results, err = idx.Search("Redis")
	if err != nil {
		t.Fatalf("Search after Remove: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("after Remove, task should not be searchable; got %d results", len(results))
	}
}

func TestRemove_NonExistentIsNoOp(t *testing.T) {
	idx := openTestIndex(t)
	// Should not error even if the task was never indexed.
	if err := idx.Remove(999); err != nil {
		t.Errorf("Remove of non-existent task should not error: %v", err)
	}
}

func TestResult_HasSnippet(t *testing.T) {
	idx := openTestIndex(t)
	if err := idx.Index(makeTask(1, "Refactor auth", "Moved the session token storage to Redis for compliance")); err != nil {
		t.Fatal(err)
	}

	results, err := idx.Search("Redis")
	if err != nil {
		t.Fatal(err)
	}
	if len(results) == 0 {
		t.Fatal("no results")
	}
	if results[0].Snippet == "" {
		t.Error("result should have a non-empty snippet")
	}
}
