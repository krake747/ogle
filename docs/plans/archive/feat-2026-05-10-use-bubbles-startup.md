# feat: use bubbles startup

Status: **Ready for implementation.** All design decisions resolved. No open questions.

---

## Context

`ogle` is a TUI application for monitoring Docker Compose projects, built on
`charm.land/bubbletea/v2`. The startup flow presents a file-selection screen
(`internal/ui/views/fileselect/fileselect.go`) when the scanner finds multiple
valid compose files. This view is hand-rolled with keyboard-only navigation
(↑/↓/enter). There is no mouse support anywhere in the application.

The goal is to improve startup flow UX by enabling mouse-driven selection of
compose files. Rather than adding `bubblezone` (which has an explicit compatibility
caveat against `charm.land/bubbletea/v2`), the plan uses `charm.land/bubbles/v2`
— the official component library for bubbletea v2 — whose `list` component
includes built-in mouse click-to-select support.

---

## Decision Log

All decisions were reached through a design interview. Do not re-open them without a specific technical reason.

| # | Decision |
|---|---|
| 1 | Use `charm.land/bubbles/v2` list component instead of `bubblezone` — bubblezone v2 has an explicit compatibility warning against `charm.land/bubbletea/v2`'s compositor; bubbles/list covers the requirement natively. |
| 2 | Do not use `bubbles/filepicker` — it is a filesystem browser (`os.ReadDir`-based) and cannot accept the pre-validated `[]string` list produced by the existing scanner pipeline. |
| 3 | Adopt `bubbles/list` defaults for layout (title top, status bar, help bar at bottom) rather than replicating the current custom layout (title top-left, list bottom-left). Avoids fighting the component's own viewport management. |
| 4 | Enable fuzzy filtering (`/` to filter) in the list component. |
| 5 | Set mouse mode to `tea.MouseModeButton` (click-only, no hover/motion tracking) — sufficient for click-to-select, lower event volume than `MouseModeCellMotion`. |
| 6 | Set `MouseMode` in `dashboard.go` `View()` — the root model that already owns the `tea.View` struct and sets `AltScreen = true`. |
| 7 | Rewrite `fileselect.go` in-place, preserving the existing public API (`New`, `SetFiles`, `SetError`, `SetParsing`, `Init`, `Update`, `View`). Zero changes required in callers. |
| 8 | Surface error and parsing state via `list.NewStatusMessage(...)` in the status bar, replacing the current inline-rendered notices. |

---

## Implementation Steps

Each step must leave the build passing before the next begins.

### Step 1 — Add `charm.land/bubbles/v2` dependency

```sh
go get charm.land/bubbles/v2
go mod tidy
```

Verify: `go build ./...` passes.

### Step 2 — Enable mouse mode at the root

In `internal/ui/flows/dashboard/dashboard.go` `View()` (line 124–129), add:

```go
v.MouseMode = tea.MouseModeButton
```

alongside the existing `v.AltScreen = true`. No other changes to this file.

Verify: `go build ./...` passes. Mouse events now flow through the program (no
component handles them yet — that is fine at this step).

### Step 3 — Rewrite `fileselect.go` using `bubbles/list`

Replace the implementation of `internal/ui/views/fileselect/fileselect.go` with
a wrapper around `charm.land/bubbles/v2/list`. All public symbols must retain
their current signatures.

**Internal `fileItem` type** — implements `list.Item` and `list.DefaultItem`:

```go
type fileItem struct{ path string }

func (f fileItem) Title() string       { return filepath.Base(f.path) }
func (f fileItem) Description() string { return f.path }
func (f fileItem) FilterValue() string { return filepath.Base(f.path) }
```

**`Model` struct** — replaces manual fields with an embedded `list.Model`:

```go
type Model struct {
    list     list.Model
    parseErr error
    errFile  string
    parsing  bool
    // files kept for SetFiles cursor-clamp and error-clear logic
    files []string
}
```

**`New(files []string, width, height int) Model`**:
- Convert `files` to `[]list.Item` via `fileItem`.
- Construct `list.New(items, list.NewDefaultDelegate(), width, height)`.
- Set `l.Title = "ogle"`.
- Enable filtering: `l.SetFilteringEnabled(true)`.

**`SetFiles(files []string) Model`**:
- Convert and call `m.list.SetItems(items)`.
- Clear error state if `errFile` is no longer present (preserve existing logic).

**`SetError(path string, err error) Model`**:
- Store `parseErr` and `errFile`.
- Call `m.list.NewStatusMessage(fmt.Sprintf("notice: %s could not be parsed: %v", base, err))`.

**`SetParsing(v bool) Model`**:
- Store `parsing`.
- If `v == true`: `m.list.NewStatusMessage("Parsing...")`.

**`Init() tea.Cmd`**: delegate to `m.list.Init()`.

**`Update(msg tea.Msg) (Model, tea.Cmd)`**:
- Delegate to `m.list.Update(msg)`, store result.
- After delegation, check `m.list.SelectedItem()` on `tea.MouseReleaseMsg` and
  `tea.KeyPressMsg` with key `"enter"` — if a `fileItem` is selected, emit
  `msgs.FileSelected{Path: item.path}`.
- Return updated model and command.

**`View() string`**: return `m.list.View()`.

Verify: `go build ./...` passes. Manual test: run `ogle` in a directory with
multiple compose files; verify list renders, keyboard nav works, mouse click
selects a file.

---

## Out of Scope

- Mouse support for any view other than `fileselect` (e.g. `watching` view, project dashboard).
- `bubblezone` — dropped entirely; not needed.
- Hover/motion tracking (`MouseModeCellMotion`).
- Preserving the exact current visual layout (title top-left, list bottom-left).
- Any changes to the scanner, parser, watcher, or domain layer.
- Tests (not requested).

---

## Post-Implementation

When all implementation steps are complete and the build passes:

1. Move this file to `docs/plans/archive/`
