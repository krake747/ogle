# feat: service list

Status: **Ready for implementation.** All design decisions resolved. No open questions.

---

## Context

`Dashboard` is the main screen after a Project is loaded. It renders a two-pane layout: service list on the left, log stream on the right. Both panes currently render placeholder strings (`"services"`, `"logs"`). `domain.Project.Services` holds a `[]ServiceDef` (name, image, container name) but nothing renders it yet. No runtime Service State exists yet — this plan covers static display only.

---

## Decision Log

All decisions were reached through a design interview. Do not re-open them without a specific technical reason.

| # | Decision |
|---|---|
| 1 | Scope is static display only: service names, cursor navigation, Service Filter. No Service State, Service Actions, or Orphan Toggle. |
| 2 | Component lives in `internal/ui/components/servicelist/` — a reusable UI primitive, not a full-screen view. |
| 3 | Wraps `bubbles/list.Model` — same pattern as `fileselect`. |
| 4 | Single-line items (service name only) using a custom delegate with `ShowDescription = false`. Expandable to two lines when Service State is added. |
| 5 | Component emits `msgs.ServiceSelected{Service domain.ServiceDef}` when cursor moves to a new service. Not emitted when list is empty or filter matches nothing. |
| 6 | `msgs.ServiceSelected` carries `domain.ServiceDef` by value (small struct, three strings; absence of selection is handled by not emitting). |
| 7 | Use `list.DefaultKeyMap()` — `/` for filter already matches the spec. Disable `Quit` and `ForceQuit` bindings so `q` and `ctrl+c` are not consumed by the component. |
| 8 | `SetSize(w, h int)` method on the component — terminal can resize at any time. |
| 9 | `New` accepts `[]domain.ServiceDef`, not `*domain.Project` — narrow dependency. |
| 10 | `SetServices([]domain.ServiceDef) Model` handles Live Reload updates — consistent with `fileselect.SetFiles()`. |
| 11 | Service list exposes its key map; `Dashboard` merges it into the single help bar at the bottom. |

---

## Implementation Steps

Each step must leave the build passing before the next begins.

### Step 1 — Add `msgs.ServiceSelected`

In `internal/msgs/msgs.go`, add:

```go
// ServiceSelected is emitted by the service list component when the cursor
// moves to a new service.
type ServiceSelected struct {
    Service domain.ServiceDef
}
```

Run `go build ./...` — must pass.

---

### Step 2 — Implement `internal/ui/components/servicelist/servicelist.go`

Create the file. Requirements:

- `serviceItem` implements `list.Item`:
  - `Title() string` → `s.def.Name`
  - `Description() string` → `""`
  - `FilterValue() string` → `s.def.Name`
- Delegate: `list.NewDefaultDelegate()` with `ShowDescription = false`.
- Key map: `list.DefaultKeyMap()` with `Quit` and `ForceQuit` disabled:
  ```go
  km := list.DefaultKeyMap()
  km.Quit.SetEnabled(false)
  km.ForceQuit.SetEnabled(false)
  ```
- `Model` is a value type wrapping `list.Model`. Tracks `lastSelected string` (service name) to detect cursor changes and avoid duplicate emissions.
- `New(services []domain.ServiceDef, w, h int) Model`
- `SetSize(w, h int) Model` — propagates to inner list
- `SetServices(services []domain.ServiceDef) Model` — replaces items
- `Init() tea.Cmd`
- `Update(msg tea.Msg) (Model, tea.Cmd)` — delegates to inner list; after update, if `SelectedItem()` differs from `lastSelected`, emit `msgs.ServiceSelected` and update `lastSelected`
- `View() string`
- `KeyMap() list.KeyMap` — returns the component's key map for help bar merging

Run `go build ./...` — must pass.

---

### Step 3 — Fix `fileselect` — disable `Quit`/`ForceQuit`

In `internal/ui/views/fileselect/fileselect.go`, after constructing the inner `list.Model` in `New`, disable the same two bindings:

```go
km := list.DefaultKeyMap()
km.Quit.SetEnabled(false)
km.ForceQuit.SetEnabled(false)
l.KeyMap = km
```

Run `go build ./...` — must pass.

---

### Step 4 — Wire `servicelist` into `Dashboard`

In `internal/ui/flows/dashboard/project/states/dashboard.go`:

1. Add `serviceList servicelist.Model` field to `Dashboard`.
2. In `NewDashboard`, construct it: `servicelist.New(project.Services, 0, 0)` (dimensions set by the first `SetSize` call).
3. In `SetSize`: compute `leftContentW` and `innerH` as currently calculated, then call `d.serviceList.SetSize(leftContentW, innerH)`.
4. In `Update`:
   - Forward `msg` to `d.serviceList.Update(msg)` and batch the returned cmd.
   - Handle `msgs.ProjectLoaded`: call `d.serviceList.SetServices(project.Services)`.
5. In `View`: replace the `"services"` placeholder render with `d.serviceList.View()` as the content of `leftInner`.
6. Update the help bar: extend `dashboardKeyMap` to include the service list bindings, or pass a combined key map to `d.help.View()`.

Run `go build ./...` — must pass.

---

## Target Structure

```
internal/
  msgs/
    msgs.go                          ← add ServiceSelected
  ui/
    components/
      servicelist/
        servicelist.go               ← new
    views/
      fileselect/
        fileselect.go                ← disable Quit/ForceQuit
    flows/
      dashboard/
        project/
          states/
            dashboard.go             ← wire servicelist
```

---

## Out of Scope

- Service State (`running`, `exited`, etc.) — requires State Polling, not yet implemented
- Service Actions (stop, start, restart, rebuild)
- Orphan Toggle
- Log Stream / right pane
- Tab/focus switching between panes
- Two-line item rendering (reserved for when Service State is added)

---

## Post-Implementation

When all implementation steps are complete and the build passes:

1. Move this file to `docs/plans/archive/`

---

## Follow-up fixes (post-archive)

Identified during code review after the original implementation. All design decisions are resolved. Implement in order; each step must leave the build passing.

### Fix 1 — `lastSelected` not cleared on `SetProject` (`servicelist.go`)

In `SetProject`, add `m.lastSelected = ""` before returning.

Reason: after Live Reload, if the cursor lands on a service whose name matches the stale `lastSelected`, `ServiceSelected` is never emitted even though the underlying `ServiceDef` may have changed.

---

### Fix 2 — `q` quits while Service Filter is active (`servicelist.go` + `dashboard.go`)

Add `IsFiltering() bool` to `servicelist.Model`:

```go
func (m Model) IsFiltering() bool {
    return m.list.FilterState() == list.Filtering
}
```

In `Dashboard.Update`, guard the quit handler:

```go
if key.Matches(keyMsg, d.keys.Quit) && !d.serviceList.IsFiltering() {
    return d, tea.Quit
}
```

---

### Fix 3 — KeyMap replacement bypasses `updateKeybindings()` (`servicelist.go` + `fileselect.go`)

**Root cause**: `list.New()` calls `updateKeybindings()` which correctly sets initial enabled state (e.g. `CancelWhileFiltering.SetEnabled(false)`, `ClearFilter.SetEnabled(false)`). Both files then assign a freshly-constructed `KeyMap` via `l.KeyMap = ...`, which re-enables all bindings — undoing `updateKeybindings()`. This causes two `esc` bindings to appear in the help bar simultaneously.

**Fix for `servicelist.go`**: Replace `newKeyMap()` helper and `l.KeyMap = newKeyMap()` with in-place modification. Delete the `newKeyMap()` function.

```go
l.KeyMap.Quit.SetEnabled(false)
l.KeyMap.ForceQuit.SetEnabled(false)
```

**Fix for `fileselect.go`**: Same pattern — only `ForceQuit` is disabled (see Fix 4):

```go
l.KeyMap.ForceQuit.SetEnabled(false)
```

---

### Fix 4 — `q` does not quit from Project Selector (`fileselect.go`)

Decision: only disable `ForceQuit` on the fileselect list, not `Quit`. The current code disables both.

Reason: `q` must quit the app from the Project Selector when the list is not filtering. `ctrl+c` (`ForceQuit`) is disabled on the inner list because it is handled at the root orchestrator level.

Combined with Fix 3, the only binding disabled on fileselect is `ForceQuit`.

---

### Fix 5 — Mouse release anywhere emits `FileSelected` (`fileselect.go`)

**Root cause**: The `Update` switch handles `tea.MouseReleaseMsg` and emits `FileSelected` for any mouse release regardless of coordinates. The comment in the code says the opposite of what the code does.

**Fix**: Emit `FileSelected` on `MouseReleaseMsg` only when the release Y coordinate falls within the list item area. The title occupies row 0; the help bar occupies the last row. Items occupy rows `1` through `m.h - 2` (inclusive).

This requires `fileselect.Model` to track `h int`. Add the field; set it in `New`; update it in `SetSize` (which is also added for Fix 6).

```go
case tea.MouseReleaseMsg:
    mouseMsg := msg.(tea.MouseReleaseMsg)
    if mouseMsg.Y >= 1 && mouseMsg.Y <= m.h-2 {
        if item, ok := m.list.SelectedItem().(fileItem); ok {
            emit = func() tea.Msg { return msgs.FileSelected{Path: item.path} }
        }
    }
case tea.KeyPressMsg:
    if kp := msg.(tea.KeyPressMsg); kp.String() == "enter" {
        if item, ok := m.list.SelectedItem().(fileItem); ok {
            emit = func() tea.Msg { return msgs.FileSelected{Path: item.path} }
        }
    }
```

---

### Fix 6 — Service Filter bindings clutter the Project Selector help bar (`fileselect.go`)

**Why**: `bubbles/list` `ShortHelp()` hardcodes `Filter` (`/`), `ClearFilter` (`esc`), `AcceptWhileFiltering` (`enter`), and `CancelWhileFiltering` (`esc`) into the short help — there is no API to suppress them from short help only.

**Approach**: disable the list's internal help renderer; render a custom help bar manually.

Steps:

1. Call `l.SetShowHelp(false)` in `New`.
2. Pass `height - 1` to `list.New` (reserve 1 row for the custom help bar).
3. Add `help help.Model` field to `fileselect.Model`.
4. `fileselect.Model` implements `help.KeyMap`:
   - `ShortHelp() []key.Binding` → `[CursorUp, CursorDown, selectBinding, Quit, ShowFullHelp]`
   - `FullHelp() [][]key.Binding` → navigation column + Service Filter column (`/`, `esc cancel`, `enter apply`) + quit column
5. Handle `?` in `Update` to toggle `m.help.ShowAll`.
6. `View()` → `m.list.View() + "\n" + m.help.View(m)`
7. Add `SetSize(w, h int) Model` method; pass `h - 1` to `m.list.SetSize` (not the full height). Update `m.h = h`.

Key bindings to show:
- Short help: `↑/↓ navigate`, `enter select`, `q quit`, `? more`
- Full help col 1 (navigation): `↑/↓`, `enter select`, `q quit`
- Full help col 2 (Service Filter): `/ filter`, `esc cancel`, `enter apply`, `? close`

`enter` appears in both short help (select item) and full help filter column (apply filter) — use two separate `key.Binding` values with distinct help text.

---

### Deferred

- **"Go back" from Dashboard to Project Selector on `q`**: requires architectural changes (storing the startup model, a `msgs.GoBack` message, watcher subscription handling). Needs its own design interview and plan.
