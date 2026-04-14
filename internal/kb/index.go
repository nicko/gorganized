package kb

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/nicko/gorganized/internal/model"
	_ "modernc.org/sqlite"
)

// Result is a single search hit from the knowledge base.
type Result struct {
	TaskID  int
	Title   string
	Snippet string
}

// Index manages the SQLite FTS5 knowledge base.
type Index struct {
	db *sql.DB
}

// Open opens (or creates) the SQLite knowledge base at the given path.
func Open(path string) (*Index, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	_, err = db.Exec(`CREATE VIRTUAL TABLE IF NOT EXISTS kb
		USING fts5(task_id UNINDEXED, title, body)`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("create fts5 table: %w", err)
	}

	return &Index{db: db}, nil
}

// Close closes the database connection.
func (idx *Index) Close() error {
	return idx.db.Close()
}

// Index upserts a task's title and notes into the FTS5 table.
// Called when a task transitions to done.
func (idx *Index) Index(t model.Task) error {
	// FTS5 doesn't support UPDATE directly; delete then insert for upsert.
	_, err := idx.db.Exec(`DELETE FROM kb WHERE task_id = ?`, t.ID)
	if err != nil {
		return fmt.Errorf("delete before upsert: %w", err)
	}
	_, err = idx.db.Exec(`INSERT INTO kb(task_id, title, body) VALUES (?, ?, ?)`,
		t.ID, t.Title, t.Notes)
	if err != nil {
		return fmt.Errorf("insert: %w", err)
	}
	return nil
}

// Search performs a full-text search and returns matching results.
// Returns an empty slice (not an error) for no matches.
// Empty query returns no results.
func (idx *Index) Search(query string) ([]Result, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, nil
	}

	// Wrap each word with prefix matching and escape special FTS5 characters.
	safeQuery := sanitizeFTS5(query)
	if safeQuery == "" {
		return nil, nil
	}

	rows, err := idx.db.Query(`
		SELECT task_id, title, snippet(kb, 2, '[', ']', '…', 8)
		FROM kb
		WHERE kb MATCH ?
		ORDER BY rank`,
		safeQuery)
	if err != nil {
		// FTS5 can return an error for malformed queries; treat as no results.
		return nil, nil
	}
	defer rows.Close()

	var results []Result
	for rows.Next() {
		var r Result
		if err := rows.Scan(&r.TaskID, &r.Title, &r.Snippet); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

// sanitizeFTS5 escapes a user query for safe use in an FTS5 MATCH expression.
// It removes FTS5 special characters and returns a quoted phrase.
func sanitizeFTS5(query string) string {
	// Remove characters that have special meaning in FTS5.
	replacer := strings.NewReplacer(
		`"`, ``,
		`*`, ``,
		`^`, ``,
		`(`, ``,
		`)`, ``,
		`{`, ``,
		`}`, ``,
		`[`, ``,
		`]`, ``,
	)
	clean := strings.TrimSpace(replacer.Replace(query))
	if clean == "" {
		return ""
	}
	// Quote the entire query so it's treated as a phrase / safe literal.
	return `"` + clean + `"`
}
