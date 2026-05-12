# feat: docker actions

Status: **Ready for implementation.** All design decisions resolved. No open questions.

---

## Context

`ogle` is a terminal UI (BubbleTea v2 + Lipgloss v2) for observing Docker Compose projects. The domain is defined in `docs/CONTEXT.md`. The Dashboard shows a two-pane layout: service list (left) and Service Inspector (right). Service list items are already single-line (`ShowDescription = false`; the description field is hidden). The docker package (`internal/services/docker/service.go`) currently only pings the daemon via `Connect()`. Service Actions (start, stop, restart, rebuild) are defined in `CONTEXT.md` but not yet implemented.

**Known build issue:** `internal/ui/flows/dashboard/project/states/dashboard.go` currently has orphaned statements outside function bodies causing compile failures. Fix these before beginning Step 1.

---

## Decision Log

All decisions were reached through a design interview. Do not re-open them without a specific technical reason.

| # | Decision |
|---|---|
| 1 | All four Service Actions are implemented: start, stop, restart, rebuild |
| 2 | Rebuild is `docker compose up --build -d <service>` — single command; compose handles the stop/recreate lifecycle |
| 3 | Start is `docker compose up -d <service>` — handles both `exited` and `not created` |
| 4 | Stop is `docker compose stop <service>` |
| 5 | Restart is `docker compose restart <service>` |
| 6 | All commands run as `docker compose -f <file> -p <name> <subcmd> <service>` using `exec.CommandContext` |
| 7 | Keybindings: `s` (start/stop, context-sensitive), `r` (restart, `running` only), `b` (rebuild, any state when daemon connected) |
| 8 | Help bar is context-sensitive — only shows action keys valid for the selected service's current state; all action keys suppressed when Docker Unavailable |
| 9 | Per-service in-flight lock — action keys suppressed for a service while it has an in-flight action; other services unaffected |
| 10 | No confirmation prompts — actions trigger immediately on keypress |
| 11 | In-flight state shown in the service list title: `◌ service-name  stopping…` — static hollow circle, orange; transient label reflects the action (see String Literals table) |
| 12 | Error state: revert to last-known state icon + red error suffix appended (e.g. `● api  stop failed`); persists until the next action is triggered on that service |
| 13 | Optimistic state update on success: stop→`exited`, start/restart/rebuild→`running` |
| 14 | Action keybindings are focus-independent — work regardless of which pane is focused; suppressed during filter input (existing `IsFiltering()` guard in `handleKeyPress` covers this) |
| 15 | Implementation shells out to the `docker compose` CLI (not the Docker Engine API) |
| 16 | Status icon prefixed to service name in the title line; icon and colour mapping defined in the String Literals table below |

---

## String Literals Reference

All user-visible strings for this feature. Do not invent alternatives.

### State icons

| Condition | Icon | Theme colour field |
|---|---|---|
| `rt == nil` (no runtime data yet) | `●` | `StateMuted` |
| `ServiceStateRunning` | `●` | `StateRunning` |
| `ServiceStateExited` or `ServiceStateDead` | `●` | `StateExited` |
| `ServiceStateNotCreated` | `○` | `StateMuted` |
| `ServiceStatePaused` | `●` | `StatePaused` |
| `ServiceStateRestarting` | `●` | `StateTransient` |
| `ServiceStateUnknown` | `●` | `StateMuted` |
| Action in-flight | `◌` | `StateTransient` |

### Transient action labels (appended after service name during in-flight)

| Action | Label |
|---|---|
| stop | `stopping…` |
| start | `starting…` |
| restart | `restarting…` |
| rebuild | `rebuilding…` |

### Error suffixes (appended after service name on failure, styled with `ActionError` colour)

| Action | Suffix |
|---|---|
| stop | `stop failed` |
| start | `start failed` |
| restart | `restart failed` |
| rebuild | `rebuild failed` |

---

## Target Structure

```
internal/
├── domain/
│   └── domain.go                          ← add ServiceAction type (modified)
├── msgs/
│   └── msgs.go                            ← add ServiceActionCompleted (modified)
├── services/docker/
│   ├── service.go                         ← unchanged
│   └── actions.go                         ← new file
└── ui/
    ├── theme/
    │   ├── theme.go                       ← add 6 colour fields to Theme struct (modified)
    │   └── builtin.go                     ← set colour values in Default() and CatppuccinoMocha() (modified)
    ├── components/servicelist/
    │   └── servicelist.go                 ← add action state, icon rendering, new Model methods (modified)
    └── flows/dashboard/project/states/
        └── dashboard.go                   ← fix broken build, add keybindings, wire action dispatch (modified)
```

---

## Implementation Steps

Each step must leave the build passing before the next begins.

### Step 0 — Fix the broken build in `dashboard.go`

`internal/ui/flows/dashboard/project/states/dashboard.go` has orphaned statements outside function bodies. Identify and remove or relocate them. Confirm `go build ./...` passes before proceeding.

### Step 1 — Add `ServiceAction` type to `internal/domain/domain.go`

```go
type ServiceAction string

const (
    ServiceActionStop    ServiceAction = "stop"
    ServiceActionStart   ServiceAction = "start"
    ServiceActionRestart ServiceAction = "restart"
    ServiceActionRebuild ServiceAction = "rebuild"
)
```

### Step 2 — Add `ServiceActionCompleted` to `internal/msgs/msgs.go`

```go
// ServiceActionCompleted is emitted by a docker action cmd when the
// docker compose subprocess exits, whether successfully or not.
type ServiceActionCompleted struct {
    ServiceName string
    Action      domain.ServiceAction
    Err         error
}
```

No existing message types change.

### Step 3 — Add action cmd factories to `internal/services/docker/actions.go`

New file. Four pure cmd factory functions consistent with `Connect()`. Each shells out using `exec.CommandContext(ctx, "docker", "compose", "-f", file, "-p", projectName, ...)` and returns `msgs.ServiceActionCompleted`.

```go
func Stop(ctx context.Context, file, projectName, serviceName string) tea.Cmd
func Start(ctx context.Context, file, projectName, serviceName string) tea.Cmd    // subcmd: "up", "-d"
func Restart(ctx context.Context, file, projectName, serviceName string) tea.Cmd
func Rebuild(ctx context.Context, file, projectName, serviceName string) tea.Cmd  // subcmd: "up", "--build", "-d"
```

Wrap any non-nil `cmd.Run()` error with `fmt.Errorf`. A non-zero exit is a failure; `Err` must be non-nil in that case.

### Step 4 — Add state-icon colours to the theme

**`internal/ui/theme/theme.go`** — add six fields to the `Theme` struct. Type is `lipgloss.Color` (not `lipgloss.Style`), consistent with `HoverBackground`:

```go
StateRunning   lipgloss.Color  // running
StateExited    lipgloss.Color  // exited / dead
StatePaused    lipgloss.Color  // paused
StateTransient lipgloss.Color  // restarting, action in-flight
StateMuted     lipgloss.Color  // not created, unknown, nil runtime
ActionError    lipgloss.Color  // error suffix text
```

Also add corresponding fields to `userThemeFile` (YAML tags: `stateRunningColor`, `stateExitedColor`, `statePausedColor`, `stateTransientColor`, `stateMutedColor`, `actionErrorColor`) and handle them in `applyOverrides`, following the existing pattern for `BorderFocusedColor`.

**`internal/ui/theme/builtin.go`** — set concrete values in both `Default()` and `CatppuccinoMocha()`:

For `Default()` (ANSI-256):
- `StateRunning`: `"2"` (green)
- `StateExited`: `"1"` (red)
- `StatePaused`: `"3"` (yellow)
- `StateTransient`: `"214"` (orange)
- `StateMuted`: `"240"` (dim grey, matches existing `blurred`)
- `ActionError`: `"1"` (red)

For `CatppuccinoMocha()` (hex):
- `StateRunning`: `"#a6e3a1"` (green)
- `StateExited`: `"#f38ba8"` (red)
- `StatePaused`: `"#f9e2af"` (yellow)
- `StateTransient`: `"#fab387"` (peach/orange)
- `StateMuted`: `"#6c7086"` (overlay0)
- `ActionError`: `"#f38ba8"` (red)

### Step 5 — Add icon rendering and action state to the service list

In `internal/ui/components/servicelist/servicelist.go`:

**`serviceItem` changes:**

Add fields and change `Title()` to return a precomputed string:

```go
type serviceItem struct {
    def            domain.ServiceDef
    runtime        *domain.ServiceRuntimeData
    actionInFlight bool
    actionLabel    string  // e.g. "stopping…"
    actionError    string  // e.g. "stop failed"
    displayTitle   string  // precomputed ANSI-styled string; returned by Title()
}

func (s serviceItem) Title() string { return s.displayTitle }
```

`serviceItem` has no direct theme access — the theme lives on `servicelist.Model`. All methods that create or update a `serviceItem` must call `buildTitle` (which takes the theme) and store the result in `displayTitle`.

**`buildTitle` helper** (package-level, unexported):

```go
func buildTitle(name string, rt *domain.ServiceRuntimeData, inFlight bool, actionLabel, actionError string, th *theme.Theme) string
```

Logic:
- If `inFlight`: icon = `◌`, colour = `th.StateTransient`, append `  actionLabel` after the name
- Else: pick icon and colour from the State Icons table using `rt` (nil → grey `●`; `ServiceStateNotCreated` → grey `○`)
- If `actionError != ""`: append `  actionError` styled with `th.ActionError`
- Compose: `lipgloss.NewStyle().Foreground(colour).Render(icon) + " " + name + suffix`

**New `Model` methods** (all follow the existing value-receiver pattern; iterate `m.list.Items()`, find by `serviceItem.def.Name`, mutate, call `m.list.SetItems`):

```go
// SetActionInFlight marks the named service as in-flight, rebuilds its displayTitle.
func (m Model) SetActionInFlight(name, label string) Model

// SetActionSuccess clears action state, applies optimistic ServiceState.
// Creates a minimal ServiceRuntimeData{State: optimisticState} if runtime is nil.
func (m Model) SetActionSuccess(name string, optimisticState domain.ServiceState) Model

// SetActionError clears in-flight state, sets error suffix on the named service.
func (m Model) SetActionError(name, errMsg string) Model

// SelectedEffectiveState returns state info for the selected item.
// hasState is false when runtime is nil and no optimistic state has been applied.
func (m Model) SelectedEffectiveState() (state domain.ServiceState, hasState bool, inFlight bool)
```

**`SetRuntimes` update:** When iterating items to apply new runtime data, preserve `actionInFlight`, `actionLabel`, `actionError` from the existing item. Rebuild `displayTitle` with the new runtime but the preserved action state.

**`toItems`:** Pass `th *theme.Theme` as a parameter. Call `buildTitle` for each item with `rt = runtimes[svc.Name]` (which may be nil).

### Step 6 — Wire keybindings and action dispatch in the Dashboard

In `internal/ui/flows/dashboard/project/states/dashboard.go`:

**Add action bindings to `dashboardKeyMap`:**

```go
ActionStop    key.Binding  // key "s", help "s stop"
ActionStart   key.Binding  // key "s", help "s start"
ActionRestart key.Binding  // key "r", help "r restart"
ActionRebuild key.Binding  // key "b", help "b rebuild"
```

`ActionStop` and `ActionStart` share the key `"s"` — only one appears in the help bar at a time.

**Refactor `handleKeyPress` to return `tea.Cmd`:**

The existing signature is `func (d *Dashboard) handleKeyPress(msg tea.KeyPressMsg)`. Change it to `func (d *Dashboard) handleKeyPress(msg tea.KeyPressMsg) tea.Cmd` and thread the return value through the `tea.KeyPressMsg` branch in `Update`. The existing `IsFiltering()` guard at the top of `handleKeyPress` already suppresses all keys during filter input — no change needed there.

**Add action handling inside `handleKeyPress`** (after the existing `Zoom`/`ToggleLabels` loop):

```go
if d.connectState != ConnectStateConnected {
    return nil
}
state, hasState, inFlight := d.serviceList.SelectedEffectiveState()
if inFlight {
    return nil
}
name := d.selectedService.Name
file := d.project.File
proj := d.project.Name

switch {
case key.Matches(msg, d.keys.ActionStop) && hasState && state == domain.ServiceStateRunning:
    d.serviceList = d.serviceList.SetActionInFlight(name, "stopping…")
    return svcdocker.Stop(d.ctx, file, proj, name)

case key.Matches(msg, d.keys.ActionStart) &&
    (!hasState || state == domain.ServiceStateExited || state == domain.ServiceStateNotCreated):
    d.serviceList = d.serviceList.SetActionInFlight(name, "starting…")
    return svcdocker.Start(d.ctx, file, proj, name)

case key.Matches(msg, d.keys.ActionRestart) && hasState && state == domain.ServiceStateRunning:
    d.serviceList = d.serviceList.SetActionInFlight(name, "restarting…")
    return svcdocker.Restart(d.ctx, file, proj, name)

case key.Matches(msg, d.keys.ActionRebuild):
    d.serviceList = d.serviceList.SetActionInFlight(name, "rebuilding…")
    return svcdocker.Rebuild(d.ctx, file, proj, name)
}
return nil
```

**Handle `msgs.ServiceActionCompleted` in `Update`:**

```go
case msgs.ServiceActionCompleted:
    optimistic := domain.ServiceStateRunning
    if msg.Action == domain.ServiceActionStop {
        optimistic = domain.ServiceStateExited
    }
    if msg.Err != nil {
        d.serviceList = d.serviceList.SetActionError(msg.ServiceName, string(msg.Action)+" failed")
    } else {
        d.serviceList = d.serviceList.SetActionSuccess(msg.ServiceName, optimistic)
    }
    return d, nil
```

**Context-sensitive help bar:**

Add a helper method to `Dashboard`:

```go
func (d *Dashboard) actionBindings() []key.Binding {
    if d.connectState != ConnectStateConnected {
        return nil
    }
    state, hasState, inFlight := d.serviceList.SelectedEffectiveState()
    if inFlight {
        return nil
    }
    var bindings []key.Binding
    switch {
    case hasState && state == domain.ServiceStateRunning:
        bindings = append(bindings, d.keys.ActionStop, d.keys.ActionRestart)
    default: // exited, not created, or no runtime data yet
        bindings = append(bindings, d.keys.ActionStart)
    }
    return append(bindings, d.keys.ActionRebuild)
}
```

Add an `actionBindings []key.Binding` field to `combinedKeyMap` and include them at the end of `ShortHelp()`. Populate it in `footerView()`:

```go
func (d *Dashboard) footerView() string {
    km := combinedKeyMap{
        dashboard:      d.keys,
        list:           d.serviceList.KeyMap(),
        actionBindings: d.actionBindings(),
    }
    return d.help.View(km)
}
```

---

## Out of Scope

- State Polling — not yet implemented; optimistic updates fill the gap for now
- Log streaming
- Orphan Toggle
- Settings overlay
- `docker compose down` — destructive (removes volumes); outside this feature boundary
- Mouse-click triggering of actions — keyboard only
- Service Actions on Orphans

---

## Post-Implementation

When all implementation steps are complete and the build passes:

1. Move this file to `docs/plans/archive/`
