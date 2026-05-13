# refactor: dashboard state refactor

Status: **Ready for implementation.** All design decisions resolved. No open questions.

---

## Context

`internal/ui/flows/dashboard/project/states/dashboard.go` is 941 lines. It contains
five distinct subsystems — key map declarations, Docker connection lifecycle, log
stream handling, mouse drag-to-select, and rendering — all on the `Dashboard` struct,
which carries 23 fields. The `states/` package already extracts cohesive types into
their own files (`logbuffer.go`, `selection.go`, `layout.go`), establishing the
pattern this refactor continues.

Goals: navigability, reduced struct complexity, and unit-testability of the three
largest subsystems in isolation.

---

## Decision Log

All decisions were reached through a design interview. Do not re-open them without
a specific technical reason.

| # | Decision |
|---|---|
| 1 | Extract `connectionMachine` type (connection lifecycle). Mutation + accessor API: methods mutate internal state, Dashboard reads via `ConnectState()` / `Unavailable()` accessors. |
| 2 | Extract `logPane` type (log stream + display). `ComputeDisplayLines` takes `width, height int` and `stderrStyle lipgloss.Style` as parameters — no layout or theme reference stored on the type. |
| 3 | Extract `dragCoordinator` type (mouse drag-to-select). Layout and component views are passed at call time, not stored. Reuse scope: `states` package only; `selectionComponent` remains shared vocabulary. Lives in `selection.go`. |
| 4 | Delete `wrapLine`. Replace with `strings.Split(ansi.Hardwrap(text, width, true), "\n")` using the already-imported `charmbracelet/x/ansi` package. The library handles the full VT grammar; the custom CSI-only parser is inferior. |
| 5 | Move key map declarations (`dashboardKeyMap`, `combinedKeyMap`, `defaultDashboardKeys`, `keyBinding`) verbatim to a new `keys.go`. No logic change. |
| 6 | No further file splits beyond the above. `dashboard.go` at ~400 lines after extraction is acceptable. |

---

## Target Structure

```
internal/ui/flows/dashboard/project/states/
  dashboard.go    ← simplified orchestrator (~400 lines, down from 941)
  connection.go   ← new: connectionMachine + startCountdown
  keys.go         ← new: key map declarations (~100 lines, verbatim move)
  logpane.go      ← new: logPane type
  selection.go    ← extended: add dragCoordinator
  layout.go       ← unchanged
  logbuffer.go    ← unchanged
  msgs.go         ← unchanged
  settings.go     ← unchanged
  state.go        ← unchanged
```

---

## API Contract

### `connectionMachine` (`connection.go`)

```go
type connectionMachine struct {
    state       inspector.ConnectState
    unavailable inspector.UnavailableState
}

func (cm *connectionMachine) HandleConnected()
func (cm *connectionMachine) HandleUnavailable() tea.Cmd        // no-op if not ConnectStateConnected
func (cm *connectionMachine) HandleGracePeriodExpired() tea.Cmd // no-op if not ConnectStateConnecting
func (cm *connectionMachine) HandleRetryTick() tea.Cmd          // fires svcdocker.Connect when countdown hits 0
func (cm *connectionMachine) ConnectState() inspector.ConnectState
func (cm *connectionMachine) Unavailable() inspector.UnavailableState

func startCountdown() tea.Cmd // package-level, moved here from dashboard.go
```

Dashboard pattern after a transition:
```go
case msgs.DaemonConnected:
    d.connection.HandleConnected()
    d.inspector = d.inspector.SetConnectState(d.connection.ConnectState())
    cmd = d.startLogStream(d.selectedService)
    d.syncLogView()
```

### `logPane` (`logpane.go`)

```go
type logPane struct {
    streamer   *logs.LogStreamer
    buffer     logBuffer
    scrollRows int
    paused     bool
    state      inspector.LogAreaState
}

// containerName is pre-computed by the caller via logs.ContainerName.
func (lp *logPane) StartStream(ctx context.Context, containerName string) tea.Cmd
func (lp *logPane) HandleLogLine(msg msgs.LogLine) tea.Cmd
func (lp *logPane) HandleStreamError()
func (lp *logPane) HandleContainerNotFound() tea.Cmd
func (lp *logPane) HandleRetry(ctx context.Context, containerName string) tea.Cmd
func (lp *logPane) ComputeDisplayLines(width, height int, stderrStyle lipgloss.Style) []string
func (lp *logPane) ScrollUp(paneHeight int)   // sets paused=true
func (lp *logPane) ScrollDown(paneHeight int) // clears paused when scrollRows==0
func (lp *logPane) Clear()                    // resets buffer + scroll state
func (lp *logPane) Close()                    // closes streamer
func (lp *logPane) State() inspector.LogAreaState
func (lp *logPane) Paused() bool
```

Dashboard call site for display sync:
```go
lb := d.layout.LogViewBounds()
stderrStyle := lipgloss.NewStyle().Foreground(d.theme.LogStderr)
lines := d.logView.ComputeDisplayLines(lb.w, lb.h, stderrStyle)
d.inspector = d.inspector.SetLogView(lines, d.logView.Paused(), d.logView.State())
```

### `dragCoordinator` (added to `selection.go`)

```go
type dragCoordinator struct {
    drag       dragSelection
    lastPressX int
    lastPressY int
}

func (dc *dragCoordinator) HandleClick(msg tea.MouseClickMsg)
// Returns true when Update must short-circuit (drag active).
func (dc *dragCoordinator) HandleMotion(msg tea.MouseMotionMsg, layout paneLayout) bool
// Returns (selectedText, handled). text is "" if drag produced no selection.
func (dc *dragCoordinator) HandleRelease(
    msg tea.MouseReleaseMsg,
    layout paneLayout,
    listView, inspView, footerView string,
) (string, bool)
// Applies reverse-video highlight to dragged rows in the fully-rendered output.
func (dc *dragCoordinator) ApplyHighlight(rendered string, layout paneLayout) string
func (dc *dragCoordinator) Active() bool
```

`hitTestComponent`, `boundsForComponent`, `extractSelection`, and `applySelectionHighlight`
become methods on `dragCoordinator` (unexported, called from the above).

### Resulting `Dashboard` struct (17 fields, down from 23)

```go
type Dashboard struct {
    ctx             context.Context
    project         *domain.Project
    keys            dashboardKeyMap
    help            help.Model
    serviceList     servicelist.Model
    inspector       inspector.Model
    layout          paneLayout
    focus           int
    selectedService domain.ServiceDef
    showLabels      bool
    theme           *theme.Theme
    themeName       string
    pollInterval    time.Duration
    logBufferCap    int
    connection      connectionMachine // replaces: connectState, unavailable
    logView         logPane           // replaces: logStreamer, logBuffer, logScrollRows, logPaused, logState
    drag            dragCoordinator   // replaces: drag, lastPressX, lastPressY
}
```

---

## Implementation Steps

Each step must leave the build passing before the next begins.

**Step 1 — `keys.go`**

Move `dashboardKeyMap`, `combinedKeyMap`, `shortHelpBaseCount`, `defaultDashboardKeys`,
and `keyBinding` verbatim from `dashboard.go` to a new `keys.go`. No logic change.
Build must pass.

**Step 2 — `connection.go` + tests**

Create `connection.go` with `connectionMachine` and `startCountdown` as specified in
the API contract above. Extract the four handler methods from `dashboard.go` into the
type. Update `dashboard.go` to delegate to `d.connection`. The `connectState` and
`unavailable` fields are removed from `Dashboard`; replaced by `connection connectionMachine`.
Write `connection_test.go` covering:
- `HandleConnected` from `ConnectStateConnecting`
- `HandleUnavailable` guard (no-op when not `ConnectStateConnected`)
- `HandleGracePeriodExpired` guard (no-op when not `ConnectStateConnecting`)
- `HandleRetryTick` countdown to zero fires `svcdocker.Connect` cmd
- `HandleRetryTick` no-op when not `ConnectStateUnavailable`

**Step 3 — `logpane.go` + `ansi.Hardwrap` substitution + tests**

Create `logpane.go` with `logPane` as specified. Extract log-stream handlers from
`dashboard.go`. Delete `wrapLine`; replace all call sites with
`strings.Split(ansi.Hardwrap(text, width, true), "\n")`. Update `dashboard.go`:
remove 5 fields, add `logView logPane`, delegate all log messages.
Write `logpane_test.go` covering `ComputeDisplayLines`:
- Scroll clamping: `scrollRows` > `maxScroll` gets clamped to `maxScroll`
- Pause-row accounting: `availRows` reduced by 1 when paused
- Empty buffer returns nil / empty slice
- `start` and `end` window is correct for given `scrollRows`

**Step 4 — `dragCoordinator` in `selection.go` + tests**

Add `dragCoordinator` to `selection.go` as specified. Move `handleMouseClick`,
`handleMouseMotion`, `handleMouseRelease`, `hitTestComponent`, `boundsForComponent`,
`extractSelection`, and `applySelectionHighlight` from `dashboard.go` into the type.
Update `dashboard.go`: remove 3 fields, add `drag dragCoordinator`, delegate mouse
messages.
Write `selection_test.go` (or extend existing) covering:
- `HandleClick` records press position, clears active drag
- `HandleMotion` below threshold (≤1 cell) returns false, no drag started
- `HandleMotion` above threshold starts drag, returns true
- `HandleRelease` on active drag clears drag state, returns handled=true
- `HandleRelease` on inactive drag returns handled=false

**Step 5 — Final `dashboard.go` cleanup**

Verify Dashboard struct has exactly 17 fields. Confirm `wrapLine` is absent.
Run `go build ./...` and `go test ./...`. Fix any remaining compilation errors.

---

## Out of Scope

- Adding new keybindings or Service Actions.
- Implementing Log Filter (planned feature noted in CONTEXT.md).
- Changing the `paneLayout` type or making it an interface.
- Cross-package reuse of `dragCoordinator`.
- Persistence of Settings changes to the Config File.
- Any changes to `settings.go`, `layout.go`, `logbuffer.go`, `msgs.go`, or `state.go`.

---

## Post-Implementation

When all implementation steps are complete and the build passes:

1. Move this file to `docs/plans/archive/`
