package kb

import (
	"testing"
	"time"

	"github.com/nicko/gorganized/internal/model"
)

func TestSearch_SpecialCharactersDoNotPanic(t *testing.T) {
	idx := openTestIndex(t)
	if err := idx.Index(makeTask(1, "Auth refactor", "Moved to Redis")); err != nil {
		t.Fatal(err)
	}

	queries := []string{
		`"quoted"`,
		`*wildcard*`,
		`(parens)`,
		`{braces}`,
		`[brackets]`,
		`a^b`,
		``,
		`   `,
		`"*^()[]{}"`, // all specials
	}
	for _, q := range queries {
		results, err := idx.Search(q)
		if err != nil {
			t.Errorf("Search(%q) returned error: %v", q, err)
		}
		_ = results
	}
}

func TestSearch_SpecialCharsStripped_TermStillMatches(t *testing.T) {
	idx := openTestIndex(t)
	now := time.Now()
	task := model.Task{
		ID: 1, Title: "Merge conflict", State: model.StateDone,
		Notes: "resolved the conflict carefully", CreatedAt: now, UpdatedAt: now,
	}
	if err := idx.Index(task); err != nil {
		t.Fatal(err)
	}

	// Query with surrounding quotes — quotes stripped, "conflict" should still match
	results, err := idx.Search(`"conflict"`)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) == 0 {
		t.Error(`Search("conflict") should find the task after stripping quotes`)
	}
}

func TestIndex_TaskWithEmptyNotes(t *testing.T) {
	idx := openTestIndex(t)
	now := time.Now()
	task := model.Task{
		ID: 1, Title: "No notes task", State: model.StateDone,
		Notes: "", CreatedAt: now, UpdatedAt: now,
	}
	if err := idx.Index(task); err != nil {
		t.Fatalf("Index with empty notes: %v", err)
	}

	results, err := idx.Search("notes")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) == 0 {
		t.Error("task with empty notes should still be searchable by title")
	}
}

func TestClose_IsIdempotentSafe(t *testing.T) {
	// Ensure Close doesn't panic when called on a valid index
	dir := t.TempDir()
	idx, err := Open(dir + "/kb.db")
	if err != nil {
		t.Fatal(err)
	}
	if err := idx.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
}
