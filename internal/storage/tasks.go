package storage

import "github.com/nicko/gorganized/internal/model"

// LoadAll reads all task markdown files from dir.
func LoadAll(dir string) ([]model.Task, error) {
	return nil, nil
}

// Write marshals a task to <dir>/<id>.md.
func Write(dir string, t model.Task) error {
	return nil
}

// Read parses a single task markdown file.
func Read(path string) (model.Task, error) {
	return model.Task{}, nil
}

// NextID returns the next available task ID (max existing + 1, starting at 1).
func NextID(dir string) (int, error) {
	return 1, nil
}
