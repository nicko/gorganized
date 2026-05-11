package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/nicko/gorganized/internal/model"
	"gopkg.in/yaml.v3"
)

// frontmatter is the YAML header stored at the top of each task file.
type frontmatter struct {
	ID          int     `yaml:"id"`
	Title       string  `yaml:"title"`
	State       string  `yaml:"state"`
	Order       int     `yaml:"order"`
	TimeSpent   int     `yaml:"time_spent"`
	ActiveSince *string `yaml:"active_since,omitempty"`
	CreatedAt   string  `yaml:"created_at"`
	UpdatedAt   string  `yaml:"updated_at"`
	DoneAt      *string `yaml:"done_at,omitempty"`
}

const timeLayout = "2006-01-02T15:04:05Z07:00"

// Write marshals a task to <dir>/<id>.md.
func Write(dir string, t model.Task) error {
	var doneAtStr *string
	if t.DoneAt != nil {
		s := t.DoneAt.UTC().Format(timeLayout)
		doneAtStr = &s
	}
	var activeSinceStr *string
	if t.ActiveSince != nil {
		s := t.ActiveSince.UTC().Format(timeLayout)
		activeSinceStr = &s
	}

	fm := frontmatter{
		ID:          t.ID,
		Title:       t.Title,
		State:       string(t.State),
		Order:       t.Order,
		TimeSpent:   t.TimeSpent,
		ActiveSince: activeSinceStr,
		CreatedAt:   t.CreatedAt.UTC().Format(timeLayout),
		UpdatedAt:   t.UpdatedAt.UTC().Format(timeLayout),
		DoneAt:      doneAtStr,
	}

	fmBytes, err := yaml.Marshal(fm)
	if err != nil {
		return fmt.Errorf("marshal frontmatter: %w", err)
	}

	content := "---\n" + string(fmBytes) + "---\n" + t.Notes

	path := filepath.Join(dir, model.FormatID(t.ID)+".md")
	return os.WriteFile(path, []byte(content), 0o644)
}

// Read parses a single task markdown file.
func Read(path string) (model.Task, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return model.Task{}, fmt.Errorf("read file: %w", err)
	}

	content := string(data)

	// Strip leading "---\n", find closing "---\n"
	if !strings.HasPrefix(content, "---\n") {
		return model.Task{}, fmt.Errorf("missing frontmatter in %s", path)
	}
	rest := content[4:]
	end := strings.Index(rest, "---\n")
	if end == -1 {
		return model.Task{}, fmt.Errorf("unclosed frontmatter in %s", path)
	}

	fmContent := rest[:end]
	notes := rest[end+4:]

	var fm frontmatter
	if err := yaml.Unmarshal([]byte(fmContent), &fm); err != nil {
		return model.Task{}, fmt.Errorf("unmarshal frontmatter: %w", err)
	}

	createdAt, err := time.Parse(timeLayout, fm.CreatedAt)
	if err != nil {
		return model.Task{}, fmt.Errorf("parse created_at: %w", err)
	}
	updatedAt, err := time.Parse(timeLayout, fm.UpdatedAt)
	if err != nil {
		return model.Task{}, fmt.Errorf("parse updated_at: %w", err)
	}

	task := model.Task{
		ID:        fm.ID,
		Title:     fm.Title,
		State:     model.State(fm.State),
		Notes:     notes,
		Order:     fm.Order,
		TimeSpent: fm.TimeSpent,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}

	if fm.DoneAt != nil {
		t, err := time.Parse(timeLayout, *fm.DoneAt)
		if err != nil {
			return model.Task{}, fmt.Errorf("parse done_at: %w", err)
		}
		task.DoneAt = &t
	}

	if fm.ActiveSince != nil {
		t, err := time.Parse(timeLayout, *fm.ActiveSince)
		if err != nil {
			return model.Task{}, fmt.Errorf("parse active_since: %w", err)
		}
		task.ActiveSince = &t
	}

	return task, nil
}

// LoadAll reads all .md files from dir and returns the parsed tasks.
func LoadAll(dir string) ([]model.Task, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read dir: %w", err)
	}

	var tasks []model.Task
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		task, err := Read(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	return tasks, nil
}

// NextID returns the next available task ID (max existing ID + 1, starting at 1).
func NextID(dir string) (int, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, fmt.Errorf("read dir: %w", err)
	}

	max := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".md")
		n, err := strconv.Atoi(name)
		if err != nil {
			continue // skip non-numeric filenames
		}
		if n > max {
			max = n
		}
	}
	return max + 1, nil
}
