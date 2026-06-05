# gorganized

A local-first, keyboard-driven TUI for personal task management, knowledge capture, and time allocation. All data lives in a single `.gorgan/` directory — no network, no accounts, no cloud sync.

Core workflow:
1. Review today's tasks
2. Select one → `space` to activate → Pomodoro timer starts
3. Work until break; `space` again → inactive → timer resets
4. Pick the next task; repeat
5. `d` to mark done → notes indexed in the knowledge base

---

## Commands

```bash
go build -o gorgan ./cmd/gorgan   # build binary
./gorgan                          # run TUI in current directory
go test ./...                     # run all tests
go test ./... -run TestName       # run a single test
go vet ./...                      # static analysis
golangci-lint run                 # lint
```

---

## Project Structure

```
cmd/gorgan/
  main.go               # entry point: init .gorgan/, launch TUI

internal/
  model/
    task.go             # Task struct, State type, state transitions, time tracking
  storage/
    tasks.go            # read/write tasks as markdown files (YAML frontmatter + body)
  tui/
    app.go              # root Bubble Tea model, key dispatch
    tasklist.go         # task list view (grouped by state, ordered)
    view.go             # status bar rendering
    noteeditor.go       # note editor overlay
    preview.go          # preview panel (right-hand side)
    pomodoro.go         # pomodoro tick message + timer status formatting
    searchview.go       # search overlay rendering
    keys.go             # keybindings
    styles.go           # Lip Gloss styles
  kb/
    index.go            # SQLite FTS5 knowledge base: insert, search, remove
  pomodoro/
    timer.go            # pure timer logic (no TUI dependency)

.gorgan/
  tasks/                # one .md file per task
  kb.db                 # SQLite FTS5 knowledge base
```

---

## Tech Stack

| Concern | Library |
|---|---|
| TUI framework | `github.com/charmbracelet/bubbletea` |
| Styling | `github.com/charmbracelet/lipgloss` |
| TUI components | `github.com/charmbracelet/bubbles` (textarea, textinput) |
| Markdown rendering | `github.com/charmbracelet/glamour` |
| Markdown frontmatter | `gopkg.in/yaml.v3` |
| SQLite | `modernc.org/sqlite` (pure Go, no CGo) |

---

## Data Model

### Task file (`.gorgan/tasks/<id>.md`)

```markdown
---
id: 1
title: "Buy groceries"
state: todo              # todo | active | inactive | done
order: 3                 # position within state group (lower = higher priority)
time_spent: 5400         # accumulated seconds across all active sessions
active_since: null       # RFC3339; set when state → active, cleared otherwise
created_at: 2026-04-14T10:00:00Z
updated_at: 2026-04-14T10:00:00Z
done_at: null            # set when state → done; cleared on undo
---

Notes body in plain markdown.
```

### State machine

```
todo     → active
active   → inactive
active   → done
inactive → active
inactive → done
done     → inactive   ← undo (space); clears done_at, removes from KB
```

### Knowledge base (`.gorgan/kb.db`)

```sql
CREATE VIRTUAL TABLE IF NOT EXISTS kb USING fts5(task_id UNINDEXED, title, body);
```

Populated on `→ done`. Entry removed on `done → inactive` (undo).

---

## Keyboard Reference

| Key | Action |
|---|---|
| `j` / `↓` | Move cursor down |
| `k` / `↑` | Move cursor up |
| `shift+↑` / `shift+↓` | Move selected task up/down within its state group |
| `space` | Cycle state; undo done |
| `d` | Mark selected task done |
| `n` | Add new task (inline title prompt) |
| `enter` | Open note editor |
| `esc` | Close note editor / search overlay (auto-saves) |
| `p` | Toggle preview panel |
| `tab` | Toggle Today / All view |
| `/` | Open search overlay |
| `q` | Quit |

---

## Layout

**Without preview panel:**
```
┌─────────────────────────────────────────┐
│ status bar (view name, timer, time)     │
├─────────────────────────────────────────┤
│  task list (full width)                 │
└─────────────────────────────────────────┘
```

**With preview panel (default):**
```
┌───────────────────────┬─────────────────┐
│ status bar                              │
├───────────────────────┼─────────────────┤
│                       │ Title           │
│  task list (60%)      │ State / Time    │
│                       │ Created         │
│                       │ Pomodoros       │
│                       │ ─────────────── │
│                       │ Notes (glamour) │
└───────────────────────┴─────────────────┘
```

---

## Testing Strategy

- **Unit tests** for `internal/model`: state transitions, time accumulation, order normalisation
- **Unit tests** for `internal/pomodoro`: tick, interval boundaries, `Remaining`
- **Integration tests** for `internal/storage`: round-trips including all fields
- **Integration tests** for `internal/kb`: index, search, remove (undo path)
- **TUI behavioural tests** via `Update()` dispatch: reorder, undo, preview toggle, time display, search
- No tests for `View()` rendering output, glamour output, or terminal layout

---

## Constraints

| Rule | Detail |
|---|---|
| No network access | All I/O is local filesystem + SQLite only |
| Scoped to CWD | All reads/writes go to `.gorgan/` in the invocation directory |
| Prompt before creating | Never silently create `.gorgan/` |
| Non-destructive done | Marking done updates frontmatter; never deletes the file |
| Undo removes from KB | `done → inactive` removes the task from `kb.db` |
| No auto-state changes | Pomodoro expiry notifies only; never transitions task state automatically |
| Order on transition | Task appended to new state group always gets the lowest priority in that group |
| Time on undo | Undoing done preserves `time_spent` |
