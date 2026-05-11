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
  storage/
    tasks.go                # read/write tasks as markdown files (YAML frontmatter + body)
  tui/
    app.go                  # root Bubble Tea model, view switching
    tasklist.go             # task list view (grouped by state, ordered)
    noteeditor.go           # note editor overlay
    preview.go              # preview panel (right-hand side)
    pomodoro.go             # pomodoro tick message + timer status formatting
    searchview.go           # search overlay rendering
    keys.go                 # keybindings
    styles.go               # Lip Gloss styles
  kb/
    index.go                # SQLite FTS5 knowledge base: insert, search, remove
  pomodoro/
    timer.go                # pure timer logic (no TUI dependency)

.gorgan/
  tasks/                    # one .md file per task
  kb.db                     # SQLite knowledge base index
```

---

## 4. Tech Stack

| Concern | Library |
|---|---|
| TUI framework | `github.com/charmbracelet/bubbletea` |
| Styling | `github.com/charmbracelet/lipgloss` |
| TUI components | `github.com/charmbracelet/bubbles` (textarea, textinput) |
| Markdown rendering | `github.com/charmbracelet/glamour` |
| Markdown frontmatter | `gopkg.in/yaml.v3` |
| SQLite | `modernc.org/sqlite` (pure Go, no CGo) |

---

## 5. Data Model

### Task (stored as `.gorgan/tasks/<id>.md`)

Task IDs are incrementing integers, zero-padded to 5 digits (`00001.md`).

```markdown
---
id: 1
title: "Buy groceries"
state: todo              # todo | active | inactive | done
order: 3                 # position within state group (lower = higher priority)
time_spent: 5400         # accumulated seconds across all active sessions
active_since: null       # RFC3339 timestamp set when state → active; cleared otherwise
created_at: 2026-04-14T10:00:00Z
updated_at: 2026-04-14T10:00:00Z
done_at: null            # set when state → done; cleared on undo
---

Notes body in plain markdown.
```

### States

| State | Description |
|---|---|
| `todo` | Not yet started |
| `active` | Being worked on; Pomodoro timer running, time accumulating |
| `inactive` | Work interrupted; timer reset, time saved to `time_spent` |
| `done` | Completed; notes indexed in KB |

### State Transitions

```
todo     → active
active   → inactive
active   → done
inactive → active
inactive → done
done     → inactive   ← undo (space); clears done_at, removes from KB
```

### Time Tracking

- `time_spent` stores accumulated seconds from all past active sessions.
- `active_since` is set when a task becomes `active` and cleared when it becomes `inactive` or `done`.
- **Total time displayed** = `time_spent + (now − active_since)` when active; `time_spent` otherwise.
- On `active → inactive` or `active → done`: add `now − active_since` to `time_spent`, clear `active_since`.
- On `done → inactive` (undo): do not modify `time_spent` (keep the time already recorded).
- On app load: if a task is `active` and `active_since` is set, time resumes accumulating from that timestamp.

### Task Ordering

- `order` is an integer within a state group. Lower = higher priority = shown first.
- `done` tasks have no meaningful order; they are always sorted by `done_at` descending.
- When a task transitions to a new state, it is assigned `order = max(existing orders in that group) + 1`.
- A reorder operation (`ctrl+↑` / `ctrl+↓`) swaps `order` between two adjacent tasks, writing exactly 2 files.
- If a group has duplicate `order` values (e.g. tasks created before this field existed), the group is normalised on first reorder (sequential integers assigned, one file write per task in group).

### Knowledge Base (SQLite `kb.db`)

FTS5 virtual table over task titles and notes. Populated on `→ done`, removed on `done → inactive` (undo).

```sql
CREATE VIRTUAL TABLE IF NOT EXISTS kb USING fts5(task_id UNINDEXED, title, body);
```

---

## 6. TUI Design

### Keyboard Reference

| Key | Action |
|---|---|
| `j` / `↓` | Move cursor down |
| `k` / `↑` | Move cursor up |
| `shift+↑` / `shift+↓` | Move selected task up/down within its state group |
| `space` | Cycle state: todo→active, active→inactive, inactive→active, done→inactive (undo) |
| `enter` | Open note editor overlay |
| `d` | Mark selected task done |
| `n` | Add new task (inline title prompt) |
| `esc` | Close note editor / search overlay (auto-saves notes) |
| `p` | Toggle preview panel |
| `tab` | Toggle Today / All view |
| `/` | Open search overlay |
| `q` | Quit |

### Layout

**Without preview panel (`p` to toggle):**
```
┌─────────────────────────────────────────┐
│ status bar (view name, timer, time)     │
├─────────────────────────────────────────┤
│                                         │
│  task list (full width)                 │
│                                         │
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

### Today View

Tasks grouped in order: **Active → Inactive → Todo**.
Done tasks completed today remain visible until midnight, then move to All only.

### All View

Tasks grouped: **Active → Inactive → Todo → Done** (done sorted by `done_at` descending).

### Preview Panel

- Rendered via `glamour` with a dark terminal style.
- Updates immediately as the cursor moves.
- Metadata section shows: title, state (coloured), time spent (`1h 23m` format), pomodoro count, created date.
- Notes section renders the markdown body below a divider.
- Empty notes shows a placeholder: `No notes yet. Press enter to add.`
- When no task is selected (empty list): panel shows placeholder text.

### Status Bar

Always visible at the top. Content when a task is active:
```
gorganized  [Today]  [Pomodoro 2] 18:42 remaining  ·  Active: 1h 23m total  tab: switch  q: quit
```

Time display format: `Xh Ym` (e.g. `1h 23m`, `23m`, `< 1m`). Only hours shown if ≥ 1h.

### Note Editor

Full-screen `bubbles/textarea` overlay, pre-filled with existing notes. `esc` auto-saves.

### Pomodoro Timer

25m work / 5m break. Status bar turns orange when a work interval expires. Timer resets on inactive/done.

---

## 7. Startup Behaviour

- `.gorgan/` absent: prompt `".gorgan/ not found. Create it here? [y/N]"`. Exit if denied.
- `.gorgan/` present: load tasks, open KB, resume timer if any task is `active`.
- Tasks without an `order` field are assigned sequential orders on first load (by `created_at`).

---

## 8. Testing Strategy

- **Unit tests** for `internal/model`: state transitions (including undo), time accumulation logic, order normalisation.
- **Unit tests** for `internal/pomodoro`: tick, interval boundaries, `Remaining`.
- **Integration tests** for `internal/storage`: round-trip including new fields (`order`, `time_spent`, `active_since`).
- **Integration tests** for `internal/kb`: index, search, remove (undo path).
- **TUI behavioural tests** via `Update()` dispatch: reorder swaps, undo transition, preview toggle, time display.
- No tests for TUI rendering output (`View()`), glamour output, or terminal-specific layout.

---

## 9. Boundaries

| Rule | Detail |
|---|---|
| No network access | All I/O is local filesystem + SQLite only |
| Scoped to CWD | All reads and writes go to `.gorgan/` in the directory where `gorgan` is invoked |
| Prompt before creating | Never silently create `.gorgan/`; always ask first |
| Non-destructive done | Marking done updates frontmatter and indexes notes; never deletes the file |
| Undo removes from KB | `done → inactive` removes the task from `kb.db` (it is no longer complete knowledge) |
| No auto-state changes | Pomodoro expiry notifies only; never transitions task state automatically |
| Order on transition | A task appended to a new state group always gets the highest order (lowest priority) in that group |
| Time on undo | Undoing done preserves `time_spent`; the time worked is not erased |
