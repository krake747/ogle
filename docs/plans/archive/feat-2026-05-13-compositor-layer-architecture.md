# feat: compositor-layer-architecture

Status: **Ready for implementation.** All design decisions resolved through design interview. Implementation resolutions documented below.

---

## Context

ogle is a Bubble Tea TUI for observing Docker Compose projects. The current
Dashboard state (`internal/ui/flows/dashboard/project/states/dashboard.go`) owns
a single `inspector.Model` for the right pane. When the Selected Service changes,
the log buffer is discarded, the log stream is closed, and a new stream is started
— scroll position and log history are lost on every selection change.

Settings is a full-terminal state swap: `project.Model` discards the entire
Dashboard and constructs a new one on dismiss, losing all accumulated log history.

Layout is managed by a hand-rolled `paneLayout` struct (`states/layout.go`). The
lipgloss v2 `Compositor`/`Layer` API (already in the dependency tree at
`charm.land/lipgloss/v2`) provides a native replacement.

This plan replaces the current approach with a full-terminal `lipgloss.Compositor`
containing per-service `ServiceLayer` sub-models, each with a persistent Log Stream
and Log Buffer. All streams run as peers from project load; no stream is ever
considered "background".

---

## Decision Log

All decisions were reached through a design interview. Do not re-open them
without a specific technical reason.

| # | Decision |
|---|---|
| 1 | "Layer" is the lipgloss compositor primitive — valid for service panes and the Settings overlay |
| 2 | Each Service Layer owns a persistent Log Stream; all streams are peers — no foreground/background distinction |
| 3 | Focus follows the top layer; keyboard log interactions target the top Service Layer. Multi-pane focus deferred. |
| 4 | Eager creation — all Service Layers and streams are created when a Project loads |
| 5 | Settings is a full-terminal compositor layer; created on open, removed on dismiss. Dashboard stays live underneath. |
| 6 | Live Reload: service added → new layer + stream; service removed → layer + stream destroyed; top layer's service removed → fall back to first service in updated list |
| 7 | Single full-terminal `lipgloss.Compositor`; `paneLayout` retired entirely |
| 8 | Each Service Layer is a full Bubble Tea sub-model with `Init()`, `Update()`, `SetSize(w, h)`, `View()` |
| 9 | The service list is also a compositor layer at a fixed position and Z |
| 10 | Log fullscreen is a layer bounds operation; the `modeSplit`/`modeLogFullscreen` enum is retired |
| 11 | Log buffer cap is per Service Layer (not a shared total) |
| 12 | Orphans receive Service Layers on the same terms as declared Services. Orphan Toggle controls visibility, not layer existence. |
| 13 | Canonical term: **Service Layer** |

---

## Implementation Resolutions

Resolved during design interview to clarify blockers and design gaps:

1. **Log message routing:** `msgs.LogLine`, `msgs.LogStreamError`, and `msgs.LogStreamContainerNotFound` did not carry service identity. **Resolution:** Add `ServiceName string` field to all three message types. Tagging happens in `servicelayer.Init()` by wrapping the streamer's `Next()` cmd before dispatch.

2. **State polling:** `msgs.ServiceStateChanged` was referenced but did not exist; state polling is unimplemented. **Resolution:** Remove all `msgs.ServiceStateChanged` references from the plan. State polling remains a future concern; `servicelayer` does not handle runtime state updates in this plan.

3. **Layer lifecycle and compositor mutability:** `lipgloss.Layer` is immutable with no `SetContent` method; `Compositor` has no `RemoveLayers`. **Resolution:** Rebuild the compositor fresh in `View()` each frame. Layer Z-positions and bounds are stored in model state; `*Layer` objects are ephemeral view artefacts. This is idiomatic Bubble Tea. Fullscreen toggle and live reload work by omitting/including layer content when rebuilding.

4. **Drag selection:** `DragCoordinator.ApplyHighlight` relies on `PaneLayout` coordinates; Step 5 removes `PaneLayout`. **Resolution:** Descope `DragCoordinator` from this plan. Step 5 will break drag highlight; a follow-up plan adapts `DragCoordinator` to compositor bounds or `Hit()`.

5. **LogPane overlap:** `LogPane` already encapsulates all the state and methods the plan assigns to `servicelayer.Model`. **Resolution:** `servicelayer.Model` embeds `logpane.LogPane` rather than rewriting it. `logpane.go` is extracted to `internal/ui/components/logpane/` in Step 2 before `servicelayer` is created; Step 3 (servicelayer) imports it from there. This avoids the circular dependency that would result from `servicelayer` (in `components/`) importing `states` for `LogPane` while `states/dashboard.go` imports `servicelayer`. The extraction is a pure package restructuring — no observable behaviour change.

6. **Message naming:** Plan referenced `msgs.DockerUnavailable/Available`; actual types are `msgs.DaemonUnavailable/DaemonConnected`. **Resolution:** Updated plan text to use correct names.

7. **Settings message:** Plan introduced `msgs.SettingsChanged`; `msgs.SettingsApplied` already exists with the required payload. **Resolution:** Plan uses existing `msgs.SettingsApplied`.

8. **Sub-model View signature:** Plan specified `View(w, h int) string`; standard Bubble Tea uses `SetSize(w, h)` + `View()`. **Resolution:** `servicelayer.Model` implements standard `SetSize()` + `View()`. Parent calls `SetSize()` on each frame before `View()`.

---

## Implementation Steps

Each step must leave the build passing before the next begins.

### Step 1 — Update CONTEXT.md

Add to `docs/CONTEXT.md`:

**Service Layer**: A persistent compositor layer that is the unit of observation
for a single Service (or Orphan). Owns a Service Inspector, Log Stream, Log
Buffer, and scroll state. Created eagerly when a Project loads; destroyed when
a Service is removed via Live Reload. All Service Layers are peers — none is
foreground or background.
_Avoid_: log pane, inspector pane

Update existing entries:

- **Service Inspector**: revise from "The right pane of the Dashboard" to "The
  view component within a Service Layer: a compact detail header above the Log
  Stream area."
- **Settings**: add that it is rendered as a full-terminal compositor layer over
  the Dashboard, which remains live underneath.
- **Selected Service**: update to "The Service whose Service Layer is currently
  on top of the compositor stack."

### Step 2 — Extract `LogPane` to `internal/ui/components/logpane/`

`servicelayer` (Step 3) lives in `internal/ui/components/`; `states/dashboard.go` (Step 4) imports `servicelayer`. If `servicelayer` imported `internal/ui/flows/dashboard/project/states` for `LogPane`, the result is a circular dependency the Go compiler rejects as a hard error:

```
states → servicelayer → states
```

`LogPane` must live in a shared package that neither leg of this pair imports from the other.

Move `internal/ui/flows/dashboard/project/states/logpane.go` and `logbuffer.go` to `internal/ui/components/logpane/` (`package logpane`).

**Changes in the new package:**

- `LogPane` type and all exported methods (`StartStream`, `HandleLogLine`, `HandleStreamError`, `HandleContainerNotFound`, `HandleRetry`, `ComputeDisplayLines`, `ScrollUp`, `ScrollDown`, `Clear`, `Close`, `State`, `Paused`, etc.) are unchanged in signature. Import path changes from `states` to `logpane`.
- `logStreamRetryMsg` moves into this package as an unexported type. Add `LogPane.Update(msg tea.Msg) (*LogPane, tea.Cmd)` that handles `logStreamRetryMsg` internally — neither `dashboard.go` nor `servicelayer` needs to type-switch on it.
- Add `LogPane.SetConnecting()` — sets internal state to `inspector.LogAreaConnecting`. Add `LogPane.MarkUnavailable()` — transitions `LogAreaNotFound → LogAreaUnavailable`. These replace three direct unexported-field writes in `dashboard.go` (lines 270 and 299 set `LogAreaConnecting`; line 335 sets `LogAreaUnavailable`) that cannot cross a package boundary.

**Changes in `states/`:**

- Delete `logpane.go` and `logbuffer.go`; add import of `internal/ui/components/logpane`.
- Remove `logStreamRetryMsg` from `msgs.go`.
- Replace the struct-literal `LogPane{streamer: ..., buffer: ..., ...}` initialiser in `NewDashboard` with `logpane.NewLogPane(streamer, bufCap)`.
- Replace the three direct `d.logView.state = ...` field writes with `d.logView.SetConnecting()` / `d.logView.MarkUnavailable()`.
- Route unrecognised messages through `d.logView.Update(msg)` for internal retry handling.

This step is a pure package restructuring — no observable behaviour change. Build passes.

### Step 3 — Introduce `servicelayer.Model` as a Bubble Tea sub-model

Create `internal/ui/components/servicelayer/servicelayer.go`.

The model owns:
- `inspector.Model` (the existing Service Inspector component)
- Embedded `logpane.LogPane` (from `internal/ui/components/logpane/`) — manages log stream, buffer, scroll state, and display line computation
- Service definition (`domain.ServiceDef`)
- `focused bool` — set by the parent to gate scroll key handling
- Z index (`int`) — used during compositor rebuild to stack layers

`Init()` starts the log stream via `logPane.StartStream(ctx, containerName)` (or sets `LogAreaNotFound` / `LogAreaConnecting` as appropriate).

`Update()` handles:
- `msgs.LogLine` (with `ServiceName` field added) — forward to `logPane`; only consumed if `msg.ServiceName == s.service.Name`
- `msgs.LogStreamError` (with `ServiceName` field added) — forward to `logPane` if service matches
- `msgs.LogStreamContainerNotFound` (with `ServiceName` field added) — forward to `logPane` if service matches
- `msgs.DaemonUnavailable` / `msgs.DaemonConnected` — update `logPane` state
- Scroll key messages — only consumed when `focused == true`; delegated to `logPane`
- `tea.WindowSizeMsg` — forward to `inspector.Model` via `SetSize(w, h)`; recompute display lines via `logPane.ComputeDisplayLines`

`SetSize(w, h int)` stores dimensions for `View()`.

`View() string` delegates to `inspector.Model.View()`.

**Message type additions:** Add `ServiceName string` field to `msgs.LogLine`, `msgs.LogStreamError`, and `msgs.LogStreamContainerNotFound`. Tagging happens in `servicelayer.Init()` by wrapping the streamer's `Next()` cmd to inject the service name before dispatching messages.

This step is purely additive. The existing Dashboard is unchanged. Build passes.

### Step 4 — Migrate Dashboard to `map[string]*servicelayer.Model`

In `internal/ui/flows/dashboard/project/states/dashboard.go`:

Replace:
```
inspector       inspector.Model
logBuffer       []string
logStreamer     *docker.LogStreamer
scrollOffset    int
displayLines    []string
selectedService domain.ServiceDef
logState        inspector.LogAreaState
```
With:
```
layers    map[string]*servicelayer.Model  // keyed by service name
topLayer  string                           // service name of the top layer
nextZ     int                              // monotonically increasing z-index for compositor
```

Changes:
- `Init()`: create a `servicelayer.Model` for every Service and Orphan in the
  project; collect and return all their `Init()` commands.
- `handleServiceSelected`: update `d.topLayer` only. Do not close streams or
  discard buffers. Increment `d.nextZ` and set the selected layer's Z.
- Message routing:
  - `msgs.LogLine` → route to `d.layers[msg.ServiceName]` if present
  - `msgs.LogStreamError` → route to `d.layers[msg.ServiceName]` if present
  - `msgs.LogStreamContainerNotFound` → route to `d.layers[msg.ServiceName]` if present
  - `tea.WindowSizeMsg` → fan out `SetSize(w, h)` to all layers
  - Scroll/pause key messages → call `SetFocused(true)` on `d.layers[d.topLayer]`, `SetFocused(false)` on all others, then route
- `renderFull()`: for the right pane, call `d.layers[d.topLayer].View()` (after size is set); keep `paneLayout` for geometry for now.

Log history and scroll position now survive service selection changes. Build passes.

### Step 5 — Replace `paneLayout` with `lipgloss.Compositor`

Retire `internal/ui/flows/dashboard/project/states/layout.go`.

**Compositor rebuild pattern:** `lipgloss.Layer` is immutable and `Compositor` exposes no `RemoveLayers` method. The correct pattern for Bubble Tea is to **rebuild the compositor fresh in each `View()` call**. Layer Z-positions and bounds are stored in the model state (e.g., `servicelayer.Model.z`); these are passed into `NewLayer(...).X(...).Y(...).Z(...)` each frame. The `*Layer` objects themselves are ephemeral — the persistence is in the model state, not the layer objects.

In `Dashboard.View()`:

Build a `lipgloss.Compositor` (call this each frame) containing:

- **Service list layer** — `lipgloss.NewLayer(content).X(0).Y(0).Z(1).ID("service-list")`;
  width = `min(termW * 30/100, 80)`.
- **Service Layer layers** — one for each service in `d.layers`. For each:
  - Set `servicelayer` to focused true/false
  - Call `servicelayer.SetSize(rightW, paneH)` then `View()` to get content
  - Create `lipgloss.NewLayer(content).X(leftW).Y(0).Z(servicelayer.z).ID(serviceName)`
  - Add to compositor
- **Settings layer** — if settings is open:
  - Render the settings form, wrap in `lipgloss.NewLayer(...).X(0).Y(0).Z(100).ID("settings")`
  - Add to compositor

Assign `d.nextZ++` to service layers on selection (stored in `servicelayer.z`).

Log fullscreen toggle: omit the service list layer when building the compositor; expand the top service layer to full terminal width. The `modeSplit`/`modeLogFullscreen` enum is removed.

Return `compositor.Render()` from `View()`. Build passes.

**Compositor sizing:** `lipgloss.NewCompositor` takes no width/height parameter. `Render()` auto-sizes its canvas to the union bounding rectangle of all layer contents — it does not pad to terminal dimensions. Each layer's content string must therefore be pre-sized to the desired pixel dimensions before being passed to `lipgloss.NewLayer`. Use `lipgloss.NewStyle().Width(w).Height(h).Render(content)` (or equivalent) on each content string. The service list layer is sized to `(leftW, paneH)`, each Service Layer content to `(rightW, paneH)`, and the Settings overlay to `(termW, termH)`. Together they fill the terminal exactly.

**Help bar:** The help bar row is appended after `compositor.Render()`, exactly as in the current `renderFull()` pattern — it is not a compositor layer.

**Known breakage:** `DragCoordinator.ApplyHighlight` currently post-processes the rendered string using `PaneLayout` coordinates. Step 5 removes `PaneLayout`, breaking drag selection highlighting until a follow-up plan adapts `DragCoordinator` to use `Compositor.Hit()` or similar. This is deferred — flag the user to expect this regression.

### Step 6 — Migrate Settings to a compositor layer

Currently `project.Model` holds a `State` interface (`*Dashboard` or `*Settings`).

Changes:
- Opening Settings: Dashboard adds the Settings layer to the compositor at Z=100.
  `project.Model` no longer swaps state.
- Dismissing Settings: omit the Settings layer on the next `View()` call (it is reconstructed per-frame, so simply don't build it). Dashboard continues with all streams and scroll positions intact.
- Settings communicates accepted changes to Dashboard via the existing
  `msgs.SettingsApplied` message rather than by reconstructing the Dashboard.
- Evaluate whether the `State` interface and `project/states/state.go` remain
  useful with only one state remaining; remove if not.

Build passes.

### Step 7 — Live Reload layer lifecycle

In the Live Reload handler in `dashboard.go`:

- **Service added**: create new `servicelayer.Model`, call `Init()`, add to
  `d.layers` and to the compositor.
- **Service removed**: stop the layer's stream, delete from `d.layers`, remove
  from the compositor.
- **Top layer's service removed**: set `d.topLayer` to the first service in the
  updated project's service list; bring that layer to the highest Z.
- **No services remain**: show an empty state in the inspector region.

Build passes.

### Step 8 — Orphan layer lifecycle

Orphans are created as Service Layers in Step 4. Confirm and wire up the
dynamic path:

- `msgs.OrphanDiscovered`: create Service Layer + stream, add to compositor.
- `msgs.OrphanGone`: destroy layer + stream, remove from compositor.
- Orphan Toggle: update service list item visibility only. The orphan's Service
  Layer and Log Stream are unaffected.

Build passes.

---

## Target Structure

```
internal/ui/
├── components/
│   ├── inspector/           (unchanged)
│   ├── logpane/             (NEW — extracted from states/ in Step 2)
│   │   └── logpane.go
│   ├── servicelayer/        (NEW — Step 3)
│   │   └── servicelayer.go
│   └── servicelist/         (unchanged)
└── flows/
    └── dashboard/
        └── project/
            ├── project.go   (simplified — Settings no longer a State branch)
            └── states/
                ├── dashboard.go  (migrated)
                ├── layout.go     (retired in Step 5)
                ├── settings.go   (refactored — compositor layer, not a State)
                └── state.go      (removed if no longer needed after Step 6)
```

---

## Out of Scope

- **Multi-pane layout** — showing two or more Service Layers simultaneously.
  The compositor bounds model supports it; deferred.
- **Focus keybinding for multi-pane** — `Tab` to cycle focus between visible
  layers. Deferred with multi-pane.
- **Log Filter** — already planned in `CONTEXT.md`; unaffected by this plan.
- **Persisting Settings changes to the Config File** — separate concern; unchanged.
- **Mouse hit-testing on layers** — `lipgloss.Compositor.Hit()` enables click
  targeting per layer; not wired in this plan.

---

## Post-Implementation

When all implementation steps are complete and the build passes:

1. Move this file to `docs/plans/archive/`
