# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

**gorganized** is a simple organizer written in Go. The project is in early development — no Go module or source files exist yet.

## Common Commands

```bash
go build -o gorgan ./cmd/gorgan   # Build binary
go build ./...                    # Build all packages
go test ./...                     # Run all tests
go test ./... -run Foo            # Run a single test by name
go vet ./...                      # Static analysis
golangci-lint run                 # Run all configured linters
```

## Architecture

See `SPEC.md` for the full specification. High-level package layout:

- `cmd/gorgan/` — entry point; handles `.gorgan/` init prompt, launches TUI
- `internal/model/` — `Task` struct, `State` type, state transition rules
- `internal/storage/` — read/write tasks as YAML-frontmatter markdown files; `NextID` for sequential IDs
- `internal/tui/` — Bubble Tea root model + views (task list, note editor, pomodoro widget, search)
- `internal/pomodoro/` — pure timer logic (no TUI dependency)
- `internal/kb/` — SQLite FTS5 knowledge base (opened from `.gorgan/kb.db`)

### Data storage

Tasks are stored as `.gorgan/tasks/00001.md` (zero-padded 5-digit integer IDs). Each file has YAML frontmatter (`id`, `title`, `state`, `created_at`, `updated_at`, `done_at`) and a markdown body for notes.

The knowledge base is a single SQLite file at `.gorgan/kb.db` using an FTS5 virtual table. It is populated (append-only) when a task is marked `done`.
