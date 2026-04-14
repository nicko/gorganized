# Implementation Plan — gorganized

## Dependency Graph

```
[1] scaffold & go.mod
      ↓
[2] model + storage (Task struct, state machine, markdown I/O)
      ↓
[3] basic TUI: Today view (read-only)
      ↓
[4] state transitions + pomodoro logic + timer widget
      ↓
[5] add task inline
      ↓
[6] note editor
      ↓
[7] all view + tab toggle + today-filter
      ↓
[8] knowledge base (SQLite FTS5) + search overlay
```

---

## Task 1 — Project scaffold

**Files:** `go.mod`, `cmd/gorgan/main.go`, stub packages under `internal/`

- `go mod init github.com/nicko/gorganized`
- Dependencies: bubbletea, lipgloss, bubbles, yaml.v3, modernc.org/sqlite
- `main.go`: check for `.gorgan/` in CWD → prompt to create → launch TUI stub

**Acceptance:** `go build ./...` succeeds; prompt works correctly on first run.

---

## Task 2 — Task model + storage

**Files:** `internal/model/task.go`, `internal/storage/markdown.go`, `internal/storage/tasks.go` + tests

- `Task` struct: `ID` (zero-padded 5-digit int, e.g. `00001`), `Title`, `State`, `CreatedAt`, `UpdatedAt`, `DoneAt`, `Notes`
- `State` consts: `StateTodo`, `StateActive`, `StateInactive`, `StateDone`
- `Task.Transition(to State) error` — enforces valid transitions
- `storage.Write`, `storage.Read`, `storage.LoadAll`, `storage.NextID`
- Files named `<id>.md` (e.g. `00001.md`); YAML frontmatter + markdown body for notes

**Acceptance:** `go test ./internal/model/... ./internal/storage/...` green; round-trip write→read produces identical Task.

### CHECKPOINT A
```
go test ./internal/model/... ./internal/storage/...
```

---

## Task 3 — Basic TUI: Today view (read-only)

**Files:** `internal/tui/app.go`, `internal/tui/tasklist.go`, `internal/tui/keys.go`, `internal/tui/styles.go`

- Load tasks on init; group by state: Active → Inactive → Todo
- `j`/`↓`, `k`/`↑` navigation; `q` quits
- No mutations yet

**Acceptance:** renders grouped list from seeded task files; navigation works.

---

## Task 4 — State transitions + pomodoro

**Files:** `internal/pomodoro/timer.go`, `internal/tui/pomodoro.go`, updates to `app.go` + `tasklist.go`

- `pomodoro.Timer`: 25m work / 5m break intervals, tick via `tea.Tick`
- `enter` cycles todo→active→inactive; `d` marks done
- Active: start timer; inactive/done: reset timer
- Status bar: `[Pomodoro N] MM:SS remaining` or `[Break] MM:SS remaining`; visual alert on expiry

**Acceptance:** `go test ./internal/pomodoro/...` green; timer shows/resets with state changes.

---

## Task 5 — Add task inline

**Files:** `internal/tui/app.go`, `internal/tui/tasklist.go`

- `a` → `bubbles/textinput` inline; `enter` creates & saves; `esc` cancels
- New task appears in Todo group immediately

**Acceptance:** task file created on disk; appears in list without restart.

---

## Task 6 — Note editor

**Files:** `internal/tui/noteeditor.go`, `internal/tui/app.go`

- `n` → full-screen `bubbles/textarea` overlay; pre-filled with existing notes
- `esc` auto-saves to disk

**Acceptance:** notes persist across open/close cycles.

---

### CHECKPOINT B
Full workflow usable: create → navigate → cycle states → notes → pomodoro timer.

---

## Task 7 — All view + tab toggle + today-filter

**Files:** `internal/tui/app.go`, `internal/tui/tasklist.go`

- `tab` toggles Today / All; header shows view name
- Done tasks with `done_at` before today's midnight hidden from Today view

**Acceptance:** yesterday's done tasks absent from Today, present in All; today's done tasks in both.

---

## Task 8 — Knowledge base + search

**Files:** `internal/kb/index.go`, `internal/tui/searchview.go`, updates to `app.go`

- `kb.Open` creates `kb.db` with FTS5 virtual table
- `kb.Index(Task)` called on → done; `kb.Search(query)` returns results
- `/` opens search overlay; `enter` navigates to task; `esc` closes

**Acceptance:** `go test ./internal/kb/...` green; done task notes are searchable.

---

### CHECKPOINT C — Full application
```
go build -o gorgan ./cmd/gorgan && ./gorgan
```
