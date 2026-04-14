# SPEC.md — gorganized

## 1. Objective

A local-first, keyboard-driven TUI application for personal task management, knowledge capture, and time allocation. The primary user is the developer themselves. All data lives on disk in a single `.gorgan/` directory — no network access, no accounts, no cloud sync.

The core workflow:
1. Review today's tasks
2. Select one → mark **active** → Pomodoro timer starts
3. Work until a break; mark **inactive** → timer resets
4. Pick the next task; repeat
5. Mark tasks **done** → notes are automatically indexed in the knowledge base

---

## 2. Commands

```bash
go mod init github.com/nicko/gorganized   # one-time setup
go build -o gorgan ./cmd/gorgan           # build binary
./gorgan                                  # run TUI in current directory
go test ./...                             # run all tests
go test ./... -run TestName               # run a single test
go vet ./...                              # static analysis
golangci-lint run                         # lint
```

The binary is named `gorgan` for brevity.

---

## 3. Project Structure

```
cmd/gorgan/
  main.go                   # entry point: init .gorgan dir, launch TUI

internal/
  model/
    task.go                 # Task struct, State type, state transitions
    note.go                 # Note struct (attached to a Task)
  storage/
    tasks.go                # read/write tasks as markdown files
    markdown.go             # frontmatter parsing (YAML) + body as notes
  tui/
    app.go                  # root Bubble Tea model, view switching
    tasklist.go             # task list view (grouped by state)
    noteeditor.go           # inline note editor overlay
    pomodoro.go             # pomodoro timer widget
    keys.go                 # keybindings (keyboard map)
    styles.go               # Lip Gloss styles
  kb/
    index.go                # SQLite-backed knowledge base: insert + search

  pomodoro/
    timer.go                # pure timer logic (no TUI dependency)

.gorgan/                    # created in CWD on first run (user-prompted)
  tasks/                    # one .md file per task
  kb.db                     # SQLite knowledge base index
```

---

## 4. Tech Stack

| Concern | Library |
|---|---|
| TUI framework | [Bubble Tea](https://github.com/charmbracelet/bubbletea) |
| Styling | [Lip Gloss](https://github.com/charmbracelet/lipgloss) |
| Reusable TUI components | [Bubbles](https://github.com/charmbracelet/bubbles) (textarea, spinner, list) |
| Markdown frontmatter | `gopkg.in/yaml.v3` |
| SQLite | `modernc.org/sqlite` (pure Go, no CGo) |
| Unique IDs | `github.com/google/uuid` |

---

## 5. Data Model

### Task (stored as `.gorgan/tasks/<id>.md`)

```markdown
---
id: <uuid>
title: "Buy groceries"
state: todo          # todo | active | inactive | done
created_at: 2026-04-14T10:00:00Z
updated_at: 2026-04-14T10:00:00Z
done_at: null        # set when state → done
---

Notes body goes here in plain markdown.
Multiple paragraphs supported.
```

### States

| State | Description |
|---|---|
| `todo` | Not yet started |
| `active` | Actively being worked on; Pomodoro timer running |
| `inactive` | Work interrupted; timer reset |
| `done` | Completed; notes indexed into KB |

### State Transitions

```
todo     → active
active   → inactive
active   → done
inactive → active
inactive → done
done     → (no transition; read-only)
```

### Knowledge Base (SQLite `kb.db`)

Full-text search (FTS5) over task titles and notes. FTS5 is SQLite's built-in full-text search engine, implemented as a virtual table — this is a SQLite concept meaning it manages an inverted index internally. The `kb.db` file is persisted on disk in `.gorgan/` and is only appended to when tasks are marked done; it is never regenerated from scratch on startup.

```sql
CREATE VIRTUAL TABLE IF NOT EXISTS kb USING fts5(task_id, title, body);
```

---

## 6. TUI Design

### Views

| Key | Action |
|---|---|
| `tab` | Toggle between **Today** view and **All** view |
| `j` / `↓` | Move cursor down |
| `k` / `↑` | Move cursor up |
| `enter` | Cycle state: todo → active → inactive |
| `n` | Open note editor for selected task |
| `esc` | Close note editor (auto-save) |
| `a` | Add new task (prompt for title inline) |
| `d` | Mark selected task done |
| `/` | Search knowledge base |
| `q` | Quit |

### Today View

Shows tasks grouped and separated by state in this order:
1. **Active**
2. **Inactive**
3. **Todo**

Done tasks are included in **Today** view until midnight of the day they were completed, then only appear in **All** view.

### All View

Shows all tasks including **Done**, grouped by state:
1. Active
2. Inactive
3. Todo
4. Done

### Note Editor

Opened as a full-screen overlay using a `bubbles/textarea`. Auto-saves to the task's markdown file on `esc`. Does not require an explicit save command.

### Pomodoro Timer

Displayed in the header/status bar when a task is **active**.
- Work interval: 25 minutes
- Short break: 5 minutes
- Shows: `[Pomodoro 1] 18:42 remaining` or `[Break] 4:12 remaining`
- When the work interval ends, a visual alert is shown; the timer does not auto-transition the task state
- Timer resets when the task is marked inactive or done

---

## 7. Startup Behaviour

On launch, `gorgan` checks for `.gorgan/` in the current working directory:

- If absent: prompt the user `".gorgan/ not found. Create it here? [y/N]"`. Exit if denied.
- If present: load tasks from `.gorgan/tasks/` and open the KB connection.

---

## 8. Testing Strategy

- **Unit tests** for `internal/model` (state machine transitions)
- **Unit tests** for `internal/pomodoro` (timer tick logic, interval boundaries)
- **Integration tests** for `internal/storage` (write task → read back, frontmatter round-trip)
- **Integration tests** for `internal/kb` (index task → FTS query returns it)
- No tests for TUI rendering (Bubble Tea models are tested via `Update()` message dispatch if needed, not visual output)

---

## 9. Boundaries

| Rule | Detail |
|---|---|
| No network access | All I/O is local filesystem + SQLite only |
| Scoped to CWD | All reads and writes go to `.gorgan/` in the directory where `gorgan` is invoked |
| Prompt before creating dirs | Never silently create `.gorgan/`; always ask first |
| Non-destructive done | Marking done never deletes the task file; it updates frontmatter and indexes notes |
| No auto-state changes | Pomodoro expiry does not automatically transition a task; it only notifies |
