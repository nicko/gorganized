# SPEC.md — gorganized

Features and acceptance criteria. Checked items are implemented.

---

## Feature 1: Project Initialisation

- [x] Running `gorgan` in a directory without `.gorgan/` prompts: `".gorgan/ not found. Create it here? [y/N]"`
- [x] Answering `N` or pressing enter exits without creating anything
- [x] Answering `Y` creates `.gorgan/tasks/` and opens `.gorgan/kb.db`
- [x] Running `gorgan` in a directory with `.gorgan/` launches the TUI immediately

---

## Feature 2: Task Management

- [x] `n` opens an inline title prompt; confirming creates a new `todo` task
- [x] `space` cycles state for the selected task (todo→active, active→inactive, inactive→active, done→inactive)
- [x] `d` marks the selected task done
- [x] Only one task can be `active` at a time (making a second task active implicitly deactivates the first)
- [x] Marking done updates frontmatter; never deletes the file

---

## Feature 3: Task Ordering

- [x] Each task has an `order` field; lower = higher priority = shown first within its group
- [x] On state transition, the task is appended at the lowest priority in the new group
- [x] `shift+↑` moves the selected task up within its state group (swaps `order` with neighbour; writes exactly 2 files)
- [x] `shift+↓` moves the selected task down within its state group
- [x] Groups with duplicate `order` values are normalised on first reorder
- [x] Tasks loaded without an `order` field are normalised on startup (sequential, sorted by `created_at`)

---

## Feature 4: Time Tracking

- [x] `time_spent` (seconds) accumulates across all active sessions
- [x] `active_since` is set when task becomes active; cleared on inactive/done
- [x] Displayed total time = `time_spent + (now − active_since)` while active; `time_spent` otherwise
- [x] On `active → inactive` or `active → done`: elapsed time is flushed into `time_spent`, `active_since` cleared
- [x] On `done → inactive` (undo): `time_spent` is preserved unchanged
- [x] On app load: if a task is `active` and `active_since` is set, the timer resumes from that timestamp

---

## Feature 5: Storage

- [x] Tasks stored as `.gorgan/tasks/<id>.md` (YAML frontmatter + markdown body)
- [x] IDs are incrementing integers, zero-padded to 5 digits (`00001.md`)
- [x] All model fields round-trip correctly through frontmatter serialisation

---

## Feature 6: Task List View

- [x] Today view groups tasks: Active → Inactive → Todo (group headers shown between)
- [x] `j` / `↓` moves cursor down; `k` / `↑` moves cursor up; cursor skips headers
- [x] Done tasks completed today remain visible until midnight; hidden in Today view after that

---

## Feature 7: Today / All View Toggle

- [x] `tab` toggles between Today and All views; header label updates immediately
- [x] All view groups: Active → Inactive → Todo → Done (done sorted by `done_at` descending)

---

## Feature 8: Note Editor

- [x] `enter` opens a full-screen textarea overlay pre-filled with existing notes
- [x] `esc` auto-saves notes and closes the overlay
- [x] Notes stored as the markdown body of the task file

---

## Feature 9: Preview Panel

- [x] `p` toggles the preview panel (default: visible)
- [x] Panel shows: title, state (coloured), time spent, pomodoro count, created date
- [x] Notes rendered via glamour (dark terminal style)
- [x] Panel updates immediately as the cursor moves
- [x] Empty notes shows placeholder: `No notes yet. Press enter to add.`

---

## Feature 10: Pomodoro Timer

- [x] 25-minute work intervals followed by 5-minute break intervals
- [x] Timer starts when a task becomes active; resets on inactive or done
- [x] Status bar shows current phase, time remaining, and total active time for the task
- [x] Status bar style changes when work interval expires
- [x] Completed work interval count tracked on the timer and shown in the preview panel

---

## Feature 11: Knowledge Base

- [x] SQLite FTS5 virtual table at `.gorgan/kb.db` (`task_id UNINDEXED, title, body`)
- [x] Task title + notes indexed when task is marked done
- [x] Entry removed from KB on `done → inactive` (undo)
- [x] Re-indexing on re-done is clean (delete then insert)

---

## Feature 12: Search

- [x] `/` opens the search overlay
- [x] Characters typed build the query; results update live
- [x] Results show title and a snippet
- [x] `backspace` removes the last character from the query
- [x] `esc` closes the overlay and clears the query
- [x] `↑` / `↓` navigate between results
- [x] `enter` navigates the task list to the matching task; switches to All view if needed
- [x] Empty query returns no results
