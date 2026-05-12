# refactor: decompose dashboard update

Status: **Ready for implementation.** All design decisions resolved. No open questions.

---

## Context

`internal/ui/flows/dashboard/project/states/dashboard.go` contains a `Dashboard`
state that implements the project flow's `State` interface (established in ADR-0007).
Its `Update()` method has grown to ~102 lines, handling four distinct concerns in a
single flat switch: key dispatch, daemon connectivity lifecycle, domain events
(`ServiceSelected`, `ProjectLoaded`), and child delegation to the Service Inspector
and service list.

The growth is driven by two sources:
1. The `KeyPressMsg` block repeats the `!d.serviceList.IsFiltering()` guard on every
   binding and will grow further as Service Actions are added to the Dashboard.
2. The four daemon lifecycle cases (`DaemonConnected`, `DaemonUnavailable`,
   `gracePeriodExpiredMsg`, `retryTickMsg`) manage a `Connecting → Connected →
   Unavailable → countdown → Connecting` state machine inline, making each transition
   hard to follow and impossible to test in isolation.

This refactor extracts both concerns into named methods, leaving `Update()` as a
thin dispatcher (~30 lines).

---

## Decision Log

All decisions were reached through a design interview. Do not re-open them without
a specific technical reason.

| # | Decision |
|---|---|
| 1 | The `KeyPressMsg` block is replaced with a keybinding table (slice of `{binding, handle}`) — not a Strategy pattern. Strategy requires interchangeable algorithms; these are mutations with no runtime swap. |
| 2 | `Quit` stays outside the keybinding table. It exits the program and skips child delegation — categorically different from state-mutating bindings. The table's contract is: mutate `d`, return nothing, child delegation always follows. |
| 3 | The `!d.serviceList.IsFiltering()` guard is hoisted to the top of `handleKeyPress`, applied once, not repeated per binding. |
| 4 | The daemon lifecycle cases are extracted to private methods (`handleDaemonConnected`, `handleDaemonUnavailable`, `handleGracePeriodExpired`, `handleRetryTick`), each returning a `tea.Cmd`. No new interface or sub-package. |
| 5 | The daemon connectivity state machine (`Connecting/Connected/Unavailable`) is considered stable — no new states anticipated. The full State pattern (as in ADR-0007) is not warranted; method extraction gives the same testability benefit at lower cost. |
| 6 | All new types and methods live in `dashboard.go`. No new files. New files in this package are for distinct types (`layout.go`, `msgs.go`); these are methods on an existing type. |

---

## Implementation Steps

Each step must leave the build passing before the next begins.

### Step 1 — Add `keyBinding` struct

Add to `dashboard.go` (below `dashboardKeyMap`):

```go
type keyBinding struct {
	binding key.Binding
	handle  func()
}
```

Build must pass before continuing.

### Step 2 — Extract `handleZoom` and `handleToggleLabels`

Add two private methods to `*Dashboard`:

```go
func (d *Dashboard) handleZoom() {
	d.layout = d.layout.ToggleMode()
	if d.layout.IsLogFullscreen() {
		d.focus = focusRight
	} else {
		d.focus = focusLeft
	}
	b := d.layout.ServiceListBounds()
	d.serviceList = d.serviceList.SetBounds(b.x, b.y, b.w, b.h)
	lb := d.layout.LogViewBounds()
	d.inspector = d.inspector.SetBounds(lb.w, lb.h)
}

func (d *Dashboard) handleToggleLabels() {
	d.showLabels = !d.showLabels
	d.inspector = d.inspector.SetShowLabels(d.showLabels)
}
```

Build must pass before continuing.

### Step 3 — Add `handleKeyPress` and replace `KeyPressMsg` case

Add the dispatch method:

```go
func (d *Dashboard) handleKeyPress(msg tea.KeyPressMsg) {
	if d.serviceList.IsFiltering() {
		return
	}
	for _, kb := range []keyBinding{
		{d.keys.Zoom, d.handleZoom},
		{d.keys.ToggleLabels, d.handleToggleLabels},
	} {
		if key.Matches(msg, kb.binding) {
			kb.handle()
			return
		}
	}
}
```

Replace the `KeyPressMsg` case in `Update()`:

```go
case tea.KeyPressMsg:
	if key.Matches(msg, d.keys.Quit) && !d.serviceList.IsFiltering() {
		return d, tea.Quit
	}
	d.handleKeyPress(msg)
```

Build and tests must pass before continuing.

### Step 4 — Extract daemon lifecycle methods

Add four private methods:

```go
func (d *Dashboard) handleDaemonConnected() tea.Cmd {
	d.connectState = inspector.ConnectStateConnected
	d.inspector = d.inspector.SetConnectState(inspector.ConnectStateConnected)
	return nil
}

func (d *Dashboard) handleDaemonUnavailable() tea.Cmd {
	if d.connectState != inspector.ConnectStateConnected {
		return nil
	}
	d.connectState = inspector.ConnectStateUnavailable
	d.unavailable = inspector.UnavailableState{SecondsUntilRetry: retryIntervalSeconds}
	d.inspector = d.inspector.SetUnavailable(d.unavailable)
	return startCountdown()
}

func (d *Dashboard) handleGracePeriodExpired() tea.Cmd {
	if d.connectState != inspector.ConnectStateConnecting {
		return nil
	}
	d.connectState = inspector.ConnectStateUnavailable
	d.unavailable = inspector.UnavailableState{SecondsUntilRetry: retryIntervalSeconds}
	d.inspector = d.inspector.SetUnavailable(d.unavailable)
	return startCountdown()
}

func (d *Dashboard) handleRetryTick() tea.Cmd {
	if d.connectState != inspector.ConnectStateUnavailable {
		return nil
	}
	d.unavailable.SecondsUntilRetry--
	d.inspector = d.inspector.SetUnavailable(d.unavailable)
	if d.unavailable.SecondsUntilRetry <= 0 {
		d.connectState = inspector.ConnectStateConnecting
		d.inspector = d.inspector.SetConnectState(inspector.ConnectStateConnecting)
		return svcdocker.Connect(context.Background())
	}
	return startCountdown()
}
```

Replace the four daemon cases in `Update()`:

```go
case msgs.DaemonConnected:
	return d, d.handleDaemonConnected()

case msgs.DaemonUnavailable:
	return d, d.handleDaemonUnavailable()

case gracePeriodExpiredMsg:
	return d, d.handleGracePeriodExpired()

case retryTickMsg:
	return d, d.handleRetryTick()
```

Build and tests must pass.

---

## Out of Scope

- Converting the daemon connectivity state machine to the full State pattern
  (ADR-0007 style). The three states are stable; method extraction is sufficient.
- Adding new key bindings or Service Actions. The table structure supports this,
  but adding bindings is a separate changeset.
- Log Filter, Settings overlay, or any other planned Dashboard features.
- Changing `View()`, `Init()`, `SetSize()`, or any other `State` method.

---

## Post-Implementation

When all implementation steps are complete and the build passes:

1. Move this file to `docs/plans/archive/`
