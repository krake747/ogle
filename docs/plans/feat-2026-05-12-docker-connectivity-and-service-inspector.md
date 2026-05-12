# feat: Docker Connectivity and Service Inspector

Status: **Ready for implementation.** All design decisions resolved. No open questions.

---

## Context

ogle currently has no Docker daemon interaction. The Dashboard opens after a Project is loaded from the Compose File, but the right pane renders the static string `"logs"` and the service list shows names with no Service State, Service Health, or any runtime data.

This plan introduces:

- A Docker connectivity layer that the Dashboard starts on entry and retries on failure.
- The **Service Inspector** — the right pane, replacing the `"logs"` placeholder — showing a stacked detail header (service metadata) above the Log Stream.
- Degraded states: connecting placeholder, Docker Unavailable with live countdown, frozen states at last-known values on runtime loss.
- The **Label Toggle** for surfacing `ogle.*` Docker labels per service.

The Log Stream itself (live Docker log tailing) and State Polling (periodic container state queries) are out of scope here. This plan builds the connectivity foundation and the Service Inspector shell that those features will plug into.

---

## Codebase Reference

An implementing agent must know the following before touching any file.

**Module path:** `github.com/ma-tf/ogle`

**Import paths** — this project uses `charm.land/` paths, not `github.com/charmbracelet/`:
```
charm.land/bubbletea/v2
charm.land/bubbles/v2/{help,key,list,...}
charm.land/lipgloss/v2
```

**Key existing types:**
```go
// internal/domain/domain.go
type Project struct {
    Name     string
    File     string
    Services []ServiceDef
}
type ServiceDef struct {
    Name          string
    Image         string
    ContainerName string
}

// internal/msgs/msgs.go — all cross-boundary messages live here
type FileAvailabilityChanged struct{ Files []string }
type ProjectLoaded struct{ Project *domain.Project }
type ServiceSelected struct{ Service domain.ServiceDef }
// ... others; see file for full list
```

**`Dashboard` state (the file being modified most):** `internal/ui/flows/dashboard/project/states/dashboard.go`
- `Update` returns `(State, tea.Cmd)` — a custom interface, not `(tea.Model, tea.Cmd)`. This is only for flow-level states (the things in `states/`). Component models (inspector, servicelist) use standard `(tea.Model, tea.Cmd)` or their own value-type pattern.
- `Init()` currently returns `nil`. The watcher subscription loop is owned entirely by the root orchestrator (`internal/ui/flows/dashboard/dashboard.go`) — do not touch watcher subscription from the project state.

**Value-receiver pattern:** `servicelist.Model` and `paneLayout` are value types. All mutating methods return a new copy — callers must assign back:
```go
m.serviceList, cmd = m.serviceList.Update(msg)
```
The new `inspector.Model` must follow the same pattern.

**`tea.Cmd` emission:** inline functions returning `tea.Msg` directly, combined with `tea.Batch`:
```go
emit := func() tea.Msg { return msgs.SomeMsg{} }
return m, tea.Batch(cmd, emit)
```

**Timer functions available in `charm.land/bubbletea/v2`:**
- `tea.Every(d, fn)` — repeating tick
- `tea.Tick(d, fn)` — one-shot timer
- `tea.After` does **not exist** in this version

**Clipboard:** `github.com/atotto/clipboard` is already in `go.sum` (indirect). Promote it to a direct dependency when implementing label drag-to-copy.

---

## Decision Log

All decisions were reached through a design interview. Do not re-open them without a specific technical reason.

| # | Decision |
|---|---|
| 1 | Dashboard opens immediately on `ProjectLoaded`. Docker connects concurrently — there is no startup gate. |
| 2 | The right pane is called the **Service Inspector**. It is a stacked layout: compact detail header above a scrolling Log Stream area. |
| 3 | The Log Stream area shows `"Connecting to Docker…"` during the initial grace period (~5s), then `"Docker unavailable — retrying in Xs…"` with a live per-second countdown. |
| 4 | Service States in the service list show `—` for all services until the first State Poll completes. |
| 5 | The detail header renders partially on entry: Compose File fields (service name, image, ports) appear immediately; Docker fields (container hash, Service State, Service Health, State Age) show `—` until `DaemonConnected` is received. |
| 6 | On runtime Docker Unavailable: the Log Stream area reverts to the unavailable placeholder. Docker fields in the detail header freeze at their last-known values. |
| 7 | Service Actions are disabled (keybindings suppressed, not shown in the help bar) while Docker is unavailable. |
| 8 | Docker connection management lives in `internal/services/docker`. It exposes a `Connect()` cmd that returns `msgs.DaemonConnected` or `msgs.DaemonUnavailable`. |
| 9 | State Polling and Log Stream gate on `msgs.DaemonConnected` — neither starts until that message is received. |
| 10 | Retry interval is 60 seconds. The countdown ticks live via `tea.Every(time.Second, …)` in the Dashboard's `Update`. |
| 11 | The `ogle.*` label section is hidden by default. The **Label Toggle** (a keybind) shows/hides it globally across all services. The section is fixed-size, scrollable, and focusable as a sub-focus within the Service Inspector (Tab to enter, Escape to return). |
| 12 | Label interaction: drag (mouse down + move) copies the label value to clipboard; Ctrl+click opens the value as a URL (underline shown on Ctrl+hover) if the value is http/https. |
| 13 | **State Age** covers all Service States — it is the time elapsed since the Service entered its current state. Not "uptime" (running-only) or "downtime" (non-running-only). |
| 14 | **Service Health** is the Docker health check result, distinct from Service State. |
| 15 | Docker connectivity starts immediately on Dashboard `Init()`, in parallel with the existing watcher subscription. |

---

## Target Structure

```
internal/
├── domain/
│   └── domain.go               ← add ServiceHealth type; ServiceRuntimeData struct
├── msgs/
│   └── msgs.go                 ← add DaemonConnected, DaemonUnavailable
├── services/
│   └── docker/
│       └── service.go          ← Connect() cmd; ping via Docker SDK
└── ui/
    ├── components/
    │   ├── servicelist/
    │   │   └── servicelist.go  ← Description() returns Service State or "—"
    │   └── inspector/          ← new package
    │       ├── inspector.go    ← Service Inspector model (header + log area)
    │       ├── header.go       ← detail header: name, hash, state, age, health, image, ports
    │       └── labels.go       ← ogle.* label section (scrollable, focusable)
    └── flows/
        └── dashboard/
            └── project/
                └── states/
                    └── dashboard.go  ← wire Inspector; handle DaemonConnected/Unavailable; retry loop
```

---

## Implementation Steps

Each step must leave the build passing before the next begins.

### Step 1 — Domain types

In `internal/domain/domain.go`:

- Add `ServiceHealth` type with values: `Healthy`, `Unhealthy`, `Starting`, `NoHealthcheck`, `Unknown`.
- Add `ServiceRuntimeData` struct:
  ```go
  type ServiceRuntimeData struct {
      ContainerID string
      State       ServiceState   // existing type
      Health      ServiceHealth
      StateAge    time.Duration
  }
  ```
- Keep `ServiceDef` unchanged — runtime data is separate from the Compose File declaration.

### Step 2 — New message types

In `internal/msgs/msgs.go`, add:

```go
type DaemonConnected struct{}
type DaemonUnavailable struct{ Err error }
```

Update the message table in `docs/flows.md`.

### Step 3 — Docker service package

Create `internal/services/docker/service.go`:

- `Connect(ctx context.Context) tea.Cmd` — attempts to ping the Docker daemon (via Docker SDK or direct socket ping). Returns `msgs.DaemonConnected` on success, `msgs.DaemonUnavailable{Err}` on failure.
- No long-running goroutine. The retry loop is driven by `tea.Every` in the Dashboard. This package is a pure cmd factory.
- Add the Docker SDK dependency to `go.mod` (`github.com/docker/docker/client`).

### Step 4 — Service Inspector component (shell)

Create `internal/ui/components/inspector/inspector.go`:

- Accepts a `domain.ServiceDef` (always available) and optional `*domain.ServiceRuntimeData` (nil until `DaemonConnected`).
- Detail header: renders service name, image from `ServiceDef` immediately. Renders container hash, Service State, Service Health, State Age from `ServiceRuntimeData` — shows `—` when nil.
- Log Stream area: placeholder only at this stage. Accepts a `ConnectState` value.
- Apply multi-column layout with truncation to the detail header. Use the existing `LogViewBounds()` rect for sizing.

Define `ConnectState` in this package (`internal/ui/components/inspector/`):

```go
type ConnectState int

const (
    ConnectStateConnecting  ConnectState = iota
    ConnectStateConnected
    ConnectStateUnavailable
)

type UnavailableState struct {
    SecondsUntilRetry int
}
```

Log Stream area render logic:
- `ConnectStateConnecting` — renders `"Connecting to Docker…"`
- `ConnectStateUnavailable` + `UnavailableState` — renders `"Docker unavailable — retrying in Xs…"`
- `ConnectStateConnected` — renders empty placeholder (Log Stream implementation is future work)

Follow the value-receiver pattern: `inspector.Model` is a value type; all mutating methods return a new copy.

### Step 5 — Label section

**First, update the domain and parser to carry labels:**

In `internal/domain/domain.go`, add `Labels map[string]string` to `ServiceDef`:
```go
type ServiceDef struct {
    Name          string
    Image         string
    ContainerName string
    Labels        map[string]string
}
```

In `internal/services/parser/service.go`, the internal `composeFile.Services` struct needs a matching field. Note: the existing `ContainerName` field has a pre-existing bug — it uses `yaml:"containerName"` but the correct Compose key is `container_name`. Fix this while you're here:
```go
// internal struct in parser/service.go
struct {
    Image         string            `yaml:"image"`
    ContainerName string            `yaml:"container_name"` // was yaml:"containerName" — bug fix
    Build         any               `yaml:"build"`
    Labels        map[string]string `yaml:"labels"`
}
```
Map `Labels` into `ServiceDef` in the same loop that maps `Image` and `ContainerName`.

**Then, create `internal/ui/components/inspector/labels.go`:**

- Renders only `ogle.*` prefixed key-value pairs from `domain.ServiceDef.Labels`.
- Fixed height, scrollable with arrow keys when focused.
- Sub-focus within the Service Inspector: Tab enters the label section, Escape returns focus to the Service Inspector.
- Ctrl+hover: detect http/https values, render with underline. Ctrl+click opens in system browser (`exec.Command("xdg-open", url)`).
- Mouse drag (mouse down + move): copy the label value to clipboard using `github.com/atotto/clipboard` (already in `go.sum`; promote to a direct dependency in `go.mod`).
- Hidden by default; visibility controlled by a bool passed in from the Dashboard model.

### Step 6 — Label Toggle keybinding

In the Dashboard model (`states/dashboard.go`):

- Add `showLabels bool` field.
- Add a keybinding (e.g. `l`) that toggles `showLabels` and passes it into the Service Inspector on each render.
- Add the binding to the help model; show it unconditionally (it is always relevant on the Dashboard).

### Step 7 — Wire Service Inspector into the Dashboard

In `states/dashboard.go`:

- Replace the `"logs"` string in `View()` with the Service Inspector component's `View()`.
- Add `inspector inspector.Model` field to the `Dashboard` struct.
- Add `connectState inspector.ConnectState` and `unavailable inspector.UnavailableState` fields.
- `Init()` currently returns `nil`. Change it to return `services/docker.Connect()`. Do **not** touch `watcher.Next()` — watcher subscription is owned entirely by the root orchestrator (`internal/ui/flows/dashboard/dashboard.go`) and must not be re-subscribed from here.
- On `msgs.DaemonConnected`: set `connectState = inspector.ConnectStateConnected`. Start the State Poll and Log Stream cmds (stubs for now — return a no-op cmd; the real implementations come in future plans).
- On `msgs.DaemonUnavailable`: set `connectState = inspector.ConnectStateUnavailable`, `unavailable = inspector.UnavailableState{SecondsUntilRetry: 60}`. Start the per-second countdown tick (see below).
- **Grace period timer:** On `Init()`, also return a `tea.Tick(5*time.Second, func(t time.Time) tea.Msg { return gracePeriodExpiredMsg{} })`. Define `gracePeriodExpiredMsg` as an unexported local type. On receipt, if `connectState` is still `ConnectStateConnecting`, transition to `ConnectStateUnavailable{SecondsUntilRetry: 60}` and start the countdown tick. (`tea.After` does not exist in this version — use `tea.Tick` for one-shot timers.)
- **Countdown tick:** `tea.Every(time.Second, func(t time.Time) tea.Msg { return retryTickMsg{} })`. On each `retryTickMsg`, decrement `unavailable.SecondsUntilRetry`. When it reaches 0, fire `services/docker.Connect()` and reset to `ConnectStateConnecting`.

### Step 8 — Handle ServiceSelected in the Dashboard

The Dashboard currently does not intercept `msgs.ServiceSelected` — it falls through to `serviceList.Update`. The Service Inspector must display the correct service when the cursor moves.

In `states/dashboard.go`:

- Add `selectedService domain.ServiceDef` field to the `Dashboard` struct. Initialise it to the first service in `project.Services` in `NewDashboard`.
- In `Update`, handle `msgs.ServiceSelected` before the fall-through to `serviceList.Update`:
  ```go
  case msgs.ServiceSelected:
      m.selectedService = msg.Service
      m.inspector = m.inspector.SetService(msg.Service)
      return m, nil
  ```
- Add `SetService(def domain.ServiceDef) Model` to the inspector model.
- Also handle `msgs.ProjectLoaded` to reset `selectedService` to the first service of the new project (Live Reload may remove the currently selected service).

### Step 9 — Service list degraded state

In `internal/ui/components/servicelist/servicelist.go`:

- `Description()` on `serviceItem` currently returns `""`. Change it to return the Service State string if `ServiceRuntimeData` is available, or `"—"` if not.
- The service list needs to accept an optional `map[string]*domain.ServiceRuntimeData` (keyed by service name) so the Dashboard can push runtime data in after polling starts. For now this map will always be nil/empty — the `"—"` placeholder is the correct display.

### Step 10 — Suppress Service Actions when Docker Unavailable

In `states/dashboard.go`:

- When `connectState != ConnectStateConnected`, remove Service Action keybindings from the active keymap passed to the help bar.
- Ensure `Update()` ignores Service Action key presses in this state.

---

## Out of Scope

- **Log Stream implementation** — Log tailing from Docker is a separate plan. The Log Stream area is a placeholder here.
- **State Polling implementation** — Periodic container state queries are a separate plan. Step 7 fires a stub cmd on `DaemonConnected`; real polling is future work.
- **Service Actions implementation** — stop/start/restart/rebuild are future work. This plan only suppresses the bindings when Docker is unavailable.
- **Settings integration** — poll interval, retry interval, and log buffer cap are not configurable in this plan.
- **Orphan display** — Orphans in the service list are out of scope here.
- **Log Filter** — planned feature, not addressed here.

---

## Post-Implementation

When all implementation steps are complete and the build passes:

1. Move this file to `docs/plans/archive/`
