# refactor: basic dashboard

Status: **Ready for implementation.** All design decisions resolved. No open questions.

---

## Context

The current `dashboard/` package has two structural problems with connection-state
tracking:

1. **Backwards dependency**: `ConnectState` and `UnavailableState` are defined in
   `internal/ui/components/inspector/` — a UI component that should not own
   infrastructure-level types. `dashboard/connection.go` imports them from there.

2. **Manual layer propagation**: Screen intercepts connection messages and
   manually iterates `d.layers` to push state via
   `SetUnavailable`/`SetConnectState`. The layers already handle
   `msgs.DaemonConnected` and `msgs.DaemonUnavailable` in their own `Update` —
   the manual iteration is redundant.

A stub `dashboard2/` package exists and is already imported by `app.go` (line 22).
This plan replaces the stub with a real implementation that uses a standalone
connection package and tea-message-based layer propagation.

---

## Decision Log

All decisions were reached through a design interview. Do not re-open them without a specific technical reason.

| # | Decision |
|---|---|
| 1 | New connection package at `internal/services/docker/connection/` — alongside the existing docker service. |
| 2 | `UnavailableState` uses `RetryAt time.Time` (UTC) instead of `SecondsUntilRetry int`. |
| 3 | Layer propagation happens via the standard tea message pipeline — `msgs.DaemonConnected`/`msgs.DaemonUnavailable` fall through to `routeUnhandled()`. No manual iteration. |
| 4 | Settings stays as an inline overlay within `dashboard2/` — not extracted to a separate flow. |
| 5 | Dashboard states (Connecting/Connected/Unavailable) are thin connection-mode controllers, not full UI instances — `ConnectionMachine` is a pure state machine, Screen owns the shared UI. |
| 6 | `inspector` package stops owning `ConnectState`/`UnavailableState` — imports from the new connection package. |

---

## Implementation Steps

1. **Create `internal/services/docker/connection/connection.go`**
   - Define `ConnectState` (Connecting/Connected/Unavailable)
   - Define `UnavailableState` with `RetryAt time.Time`
   - Define `ConnectionMachine` with methods:
     - `HandleConnected()` — transitions to Connected
     - `HandleUnavailable(now time.Time)` — transitions to Unavailable, sets RetryAt = now + 60s, returns cmd (DaemonUnavailable) or nil if already unavailable
     - `HandleGracePeriodExpired(now time.Time)` — same as HandleUnavailable but only fires from Connecting state
     - `HandleRetryTick(now time.Time)` — if RetryAt <= now, transition to Connecting, return `svcdocker.Connect` cmd; else return nil
     - `ConnectState()` and `Unavailable()` accessors
   - The connection package imports `msgs` (for DaemonUnavailable) and `svcdocker` (for Connect). No UI imports.

2. **Update `internal/ui/components/inspector/inspector.go`**
   - Remove `ConnectState`, `ConnectState*` constants, and `UnavailableState` type definitions
   - Import `"github.com/ma-tf/ogle/internal/services/docker/connection"`
   - Replace all `ConnectState` → `connection.ConnectState`, `ConnectStateConnecting` → `connection.ConnectStateConnecting`, etc.
   - Replace `UnavailableState` → `connection.UnavailableState`
   - Update `SetUnavailable` signature and `renderLogArea` to use `time.Until(m.unavailable.RetryAt)` instead of integer field
   - Update test references in `inspector/` package

3. **Update `internal/ui/components/servicelayer/servicelayer.go`**
   - Import `connection` package instead of inspector types where applicable
   - `SetUnavailable` and `SetConnectState` signatures reference `connection.UnavailableState` / `connection.ConnectState`

4. **Rebuild `dashboard2/dashboard2.go`** — replace stub with:
   - `Model` struct: same fields as current dashboard.Model minus duplicates, plus `*connection.ConnectionMachine`
   - Internal messages: `gracePeriodExpiredMsg`, `retryTickMsg`
   - `New(ctx, project, th, themeName, poll, logBufCap, zm, w, h) Model`
     - Creates `ConnectionMachine` in initial `ConnectStateConnecting`
     - Initializes service list, layers, layout, help
     - Same constructor pattern as current `dashboard/model.go`
   - `Init()` — fires `svcdocker.Connect` + grace period tick + all layer Init commands
   - `Update()` — delegates known msgs to handlers, routes rest via `routeUnhandled`
   - Connection handlers (simplified, no layer iteration):
     - `handleDaemonConnectedMsg()`: `d.conn.HandleConnected()`, return nil (msg falls through to layers)
     - `handleDaemonUnavailable()`: return `d.conn.HandleUnavailable(time.Now().UTC())` (msg falls through)
     - `handleGracePeriodExpired()`: return `d.conn.HandleGracePeriodExpired(time.Now().UTC())`
     - `handleRetryTick()`: return `d.conn.HandleRetryTick(time.Now().UTC())`
   - `View()` — renders via compositor (same pattern as current Screen), uses `d.conn.ConnectState()` and `time.Until(d.conn.Unavailable().RetryAt)` for status displays
   - Key bindings, action dispatch, settings overlay — same as current dashboard but referencing `d.conn` instead of `d.connection`

5. **Update `internal/app/app.go`**
   - Already imports `dashboard2`. Pass new constructor args as needed (follow current `dashboard.New` signature).
   - Remove import of old `dashboard` package if no longer referenced.

6. **Remove old usages and verify build**
   - Check no remaining imports of `inspector.ConnectState` or `inspector.UnavailableState`
   - Remove `dashboard/connection.go` if unused (optional — could leave for reference)
   - Run `go build ./...` and `go test ./...`

---

## Out of Scope

- Extracting Settings to a separate flow or package — stays inline in dashboard2.
- Full dashboard2 parity with every dashboard1 feature — connection extraction only.
- Rewriting `servicelayer` or `logpane` internals — only import path changes.
- Removing or migrating the old `dashboard/` package — left intact.
- Drag-to-select text extraction — stays in dashboard1 only for now.
- Removing `dashboard.State` interface — dashboard2 uses its own internal pattern.

---

## Post-Implementation

When all implementation steps are complete and the build passes:

1. Move this file to `docs/plans/archive/`
