# refactor: Replace Manual Hit-Testing with bubblezone

Status: **Ready for implementation.** All design decisions resolved. No open questions.

---

## Context

Three locations in the UI perform manual mouse hit-testing by comparing raw
pixel coordinates against stored bounding rectangles:

1. `hoverlist.Layout.HitTest` — `internal/ui/hoverlist/hoverlist.go:71`
2. `DragCoordinator.hitTestComponent` — `internal/ui/flows/dashboard/project/states/selection.go:258`
3. `fileselect.hitTest` — `internal/ui/views/fileselect/fileselect.go:144`

This pattern is fragile: bounding boxes must be kept in sync with layout;
off-by-one errors are common; and the coordinate arithmetic is invisible to
tests.

`bubblezone/v2` (import path `github.com/lrstanley/bubblezone/v2`) replaces
all three sites with named zones. The rendering layer marks zones on output
strings; the event layer resolves coordinates to zone names. This eliminates
coordinate arithmetic entirely and makes hit-testing a string comparison.

A secondary bug is fixed in this plan: `fileselect.Model` emits
`msgs.FileSelected` on any `MouseReleaseMsg` regardless of whether the release
falls on the same item that received the press (drag-origin bug).

---

## Decision Log

All decisions were reached through a design interview. Do not re-open them without a specific technical reason.

| # | Decision |
|---|---|
| 1 | `bubblezone/v2` is not yet a dependency — step 1 adds it via `go get`. |
| 2 | A single `*zone.Manager` is created in the root `dashboard.Model` constructor (`dashboard.go`) and threaded down through all sub-model constructors. It is not a global. |
| 3 | `zone.Scan()` is called on `v.Content` in the root `dashboard.Model.View()`. `View()` returns `tea.View`; the correct form is `v.Content = zm.Scan(v.Content)` followed by `return v`. One frame delay is acceptable and expected. |
| 4 | Zone names: `"item-{i}"` in `hoverlist.delegate.Render`; `"pane-left"` and `"pane-right"` in `PaneLayout.View`. No prefix collisions because `fileselect` and `servicelist` are never rendered simultaneously. |
| 5 | `hoverlist.Layout` struct and its `HitTest` method are deleted after all call sites are replaced (step 7). `ServiceListBounds` and `LogViewBounds` on `PaneLayout` are retained — they serve layout sizing, not hit-testing. |
| 6 | `fileselect.Model` gains a `pressedIdx int` field (default `-1`). Set on `tea.MouseClickMsg`; `msgs.FileSelected` is only emitted on `tea.MouseReleaseMsg` when the release zone matches `pressedIdx`. This fixes the drag-origin bug. |
| 7 | `handleMouseWheel` coordinate check in `dashboard.go` is intentionally NOT replaced — it is not one of the three targeted hit-test sites and is not buggy. |
| 8 | In fullscreen/zoom mode `"pane-left"` zone will not exist; `zm.Get("pane-left").InBounds(x, y)` returns false naturally — no special case needed. |
| 9 | `startup.Model` and all startup states (`scanning`, `watching`, `selecting`, `handler`) that construct or thread `fileselect.Model` receive the `zm *zone.Manager` parameter. `startup.Model.Update`'s `msgs.WatcherError` case (which calls `states.NewWatchingWithError`) must also pass `m.zm`. |
| 10 | `hoverlist.NewDelegate` gains a `zm *zone.Manager` parameter stored on the unexported `delegate` struct. `servicelist.New` and `fileselect.New` — which are already receiving `zm` by step 2 — pass it through to `NewDelegate`. |
| 11 | `hoverlist.delegate.Render` captures `DefaultDelegate.Render` output into a `strings.Builder` (via `io.Writer`), then writes `zm.Mark(name, buf.String())` to `w`. This is the only correct way to inject zone marks into a `Render(w io.Writer, ...)` method. |
| 12 | `DragCoordinator.hitTestComponent` (in `selection.go`) replaces the `ServiceListBounds`/`LogViewBounds` checks with zone lookups for `SelectionServiceList` and `SelectionInspector`. The `SelectionFooter` y-coordinate check (`y == paneH`) is preserved — the footer help bar is not a rendered zone. |

---

## Implementation Findings

Findings from codebase ingestion. These clarify or correct details in the steps
below; they do not require re-opening closed decisions.

| # | Finding |
|---|---|
| F1 | `selection_test.go` calls `states.NewPaneLayout(nil)` (no zm arg yet) and constructs `var dc states.DragCoordinator` as a zero value. After step 2 `NewPaneLayout` gains a second `nil` zm arg. After step 6 `hitTestComponent` calls `dc.zm.Get(...)` — nil `zm` panics. Fix: add an exported `SetZoneManager(*zone.Manager)` method to `DragCoordinator` (analogous to `SetLastPress`/`SetActiveDrag`) and rewrite `TestDragCoordinator_HandleMotion_AboveThreshold_StartsDrag` to set up a fake zone via `zm.Mark`/`zm.Scan` before exercising `HandleMotion`. Do this as part of step 6/7. |
| F2 | `project.Model` does **not** need to store `zm`. `project.New` passes it through to `states.NewDashboard` at construction time; `project.Model.Update` never calls `NewDashboard` directly (Settings handles that via its own stored `zm`). The "thread zm" annotation in the target structure is accurate — no field storage needed in `project.go`. |
| F3 | `Selecting` gaining a `zm` field (plan step 2) is redundant: `Selecting.handler` (a `fileHandler`) already carries `zm`, and `Selecting` never calls `fileselect.New` directly. Adding the field requires `zm: s.zm` / `zm: fh.zm` at all four inline struct-literal sites (`withError`, `withParsing`, two `Update` cases). Retain the field to match the plan; annotate inline literals accordingly. |
| F4 | `PaneLayout.View` has two branches: `modeSplit` (both `leftInner` + `rightInner`) and `modeLogFullscreen` (only `rightInner`). Mark `pane-right` in **both** branches so drag detection works in fullscreen. `pane-left` is only marked in `modeSplit`; decision 8 (returns false naturally) handles the fullscreen case. |
| F5 | `hoverlist.go` currently imports only `"io"`. Step 3 needs `"fmt"` and `"strings"` added. |
| F6 | Removing `layout hoverlist.Layout` from `servicelist.Model` also requires removing the `m.layout.Width = w` line in `SetBounds` and the `m.layout.*` assignments. In `fileselect`, the `WindowSizeMsg` handler also sets `m.layout.Width = sz.Width` — that line must be removed in step 5. |

---

## Implementation Steps

Each step must leave the build passing before the next begins.

### Step 1 — Add `bubblezone/v2` dependency

```bash
go get github.com/lrstanley/bubblezone/v2
go mod tidy
```

Run `go build ./...`.

### Step 2 — Thread `*zone.Manager` through all constructors

Create a single `zone.Manager` in the root `dashboard.Model` constructor
(`dashboard.go:New`):

```go
zm := zone.New()
```

Store it as a `zm *zone.Manager` field on `dashboard.Model` and add
`zm *zone.Manager` parameter to every constructor in the call chain:

- `internal/ui/flows/dashboard/dashboard.go` — `New`; creates `zone.New()`; stores `zm`; passes to `startup.New`
- `internal/ui/flows/startup/startup.go` — `New`; stores `zm`; passes to all state constructors
- `internal/ui/flows/startup/states/scanning.go` — `NewScanning`
- `internal/ui/flows/startup/states/watching.go` — `NewWatching`, `NewWatchingWithError`
- `internal/ui/flows/startup/states/handler.go` — `fileHandler` struct gains `zm` field; passes to `fileselect.New`
- `internal/ui/flows/startup/states/selecting.go` — `Selecting` struct gains `zm` field
- `internal/ui/views/fileselect/fileselect.go` — `New`; stores `zm`
- `internal/ui/hoverlist/hoverlist.go` — `NewDelegate` gains `zm *zone.Manager`; stored on `delegate`
- `internal/ui/flows/dashboard/project/project.go` — `New`; passes to `states.NewDashboard`
- `internal/ui/flows/dashboard/project/states/dashboard.go` — `NewDashboard`; stores `zm`; passes to `servicelist.New` and `newPaneLayout`
- `internal/ui/flows/dashboard/project/states/settings.go` — passes `zm` through both `NewDashboard` calls
- `internal/ui/components/servicelist/servicelist.go` — `New`; stores `zm`; passes to `hoverlist.NewDelegate`
- `internal/ui/flows/dashboard/project/states/layout.go` — `NewPaneLayout`; stores `zm`

Also update `startup.Model.Update`'s `msgs.WatcherError` case to pass `m.zm`
to `states.NewWatchingWithError`.

Run `go build ./...`.

### Step 3 — Mark zones in rendering + wire `zone.Scan` at root

**`PaneLayout.View`** (`layout.go`): wrap `leftInner` and `rightInner` with
zone marks before passing to the lipgloss border styles:

```go
leftInner  = zm.Mark("pane-left",  leftInner)
rightInner = zm.Mark("pane-right", rightInner)
```

**`hoverlist.delegate.Render`** (`hoverlist.go`): capture the default
delegate's output, then write the zone-marked result to `w`:

```go
func (d *delegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	var buf strings.Builder
	if index == d.hoverIndex {
		dd := d.DefaultDelegate
		bg := d.theme.HoverBackground
		dd.Styles.NormalTitle = dd.Styles.NormalTitle.Background(bg)
		dd.Styles.NormalDesc = dd.Styles.NormalDesc.Background(bg)
		dd.Render(&buf, m, index, item)
	} else {
		d.DefaultDelegate.Render(&buf, m, index, item)
	}
	_, _ = io.WriteString(w, d.zm.Mark(fmt.Sprintf("item-%d", index), buf.String()))
}
```

Add `"strings"` and `"fmt"` to the `hoverlist` imports if not already present.

**Root `dashboard.Model.View()`** (`dashboard.go`): apply `zone.Scan` to the
final content before returning:

```go
func (m Model) View() tea.View {
	v := m.current.View()
	v.Content = m.zm.Scan(v.Content)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}
```

Run `go build ./...` and do a manual smoke-test (no visual regression expected;
zone markers are non-printable bytes that lipgloss passes through).

### Step 4 — Wire `servicelist` hit-testing via zones

In `internal/ui/components/servicelist/servicelist.go`, replace the `hitTest`
method that uses `hoverlist.Layout.HitTest` with zone lookups:

```go
func (m Model) hitTest(x, y int) (int, bool) {
	for i := range m.list.VisibleItems() {
		if m.zm.Get(fmt.Sprintf("item-%d", i)).InBounds(x, y) {
			return i, true
		}
	}
	return 0, false
}
```

Remove the `layout hoverlist.Layout` field from `servicelist.Model` and all
`SetBounds` assignments to `m.layout.OriginX` / `m.layout.OriginY`. Run
`go build ./...`.

### Step 5 — Wire `fileselect` hit-testing + fix drag-origin bug

In `internal/ui/views/fileselect/fileselect.go`:

1. Replace `hitTest` with zone lookups (same pattern as step 4).
2. Add `pressedIdx int` field, initialised to `-1` in `New`.
3. In `tea.MouseClickMsg` handler: set `m.pressedIdx = hitResult` (where
   `hitResult` is the index from `m.hitTest(msg.X, msg.Y)`, or `-1` on miss).
4. In `tea.MouseReleaseMsg` handler: only emit `msgs.FileSelected` if
   `releaseIdx == m.pressedIdx && m.pressedIdx >= 0`. Reset `m.pressedIdx = -1`
   in all cases after handling the release.
5. Remove the `layout hoverlist.Layout` field and all `SetBounds` assignments
   to `m.layout.OriginX` / `m.layout.OriginY`.

Run `go build ./...`.

### Step 6 — Replace `DragCoordinator.hitTestComponent` with zone lookup

In `internal/ui/flows/dashboard/project/states/selection.go`, replace the
`hitTestComponent` method body. The zone lookups replace the two bounding-rect
checks; the `SelectionFooter` y-coordinate check is **preserved** because the
footer help bar is not a rendered zone:

```go
func (dc *DragCoordinator) hitTestComponent(x, y int, layout PaneLayout) SelectionComponent {
	if dc.zm.Get("pane-left").InBounds(x, y) {
		return SelectionServiceList
	}

	if dc.zm.Get("pane-right").InBounds(x, y) {
		return SelectionInspector
	}

	paneH := layout.h - separatorRows - helpBarHeight
	if y == paneH {
		return SelectionFooter
	}

	return SelectionNone
}
```

`DragCoordinator` gains a `zm *zone.Manager` field; `newDragCoordinator` gains
a `zm *zone.Manager` parameter and stores it. Update `NewDashboard` to pass
`zm` to `newDragCoordinator`.

Run `go build ./...`.

### Step 7 — Delete `hoverlist.Layout`

In `internal/ui/hoverlist/hoverlist.go`, delete the `Layout` struct and its
`HitTest` method. Confirm no remaining references:

```bash
grep -r "hoverlist\.Layout\|\.HitTest" --include="*.go" .
```

Run `go build ./...` and `go test ./...`.

---

## Target Structure

Files modified:

```
internal/
  ui/
    flows/
      dashboard/
        dashboard.go                          (create zm; zone.Scan in View)
        project/
          project.go                          (thread zm)
          states/
            dashboard.go                      (store zm; pass to servicelist, newPaneLayout, newDragCoordinator)
            settings.go                       (pass zm through both NewDashboard calls)
            layout.go                         (receive zm; mark pane-left/pane-right in View)
            selection.go                      (DragCoordinator gains zm; replace hitTestComponent)
      startup/
        startup.go                            (receive and store zm; thread to states)
        states/
          scanning.go                         (receive zm)
          watching.go                         (receive zm; WatcherError case passes zm)
          handler.go                          (fileHandler gains zm; passes to fileselect.New)
          selecting.go                        (Selecting gains zm)
    components/
      servicelist/
        servicelist.go                        (receive zm; pass to NewDelegate; replace hitTest; remove layout field)
    views/
      fileselect/
        fileselect.go                         (receive zm; replace hitTest; fix drag-origin bug; remove layout field)
    hoverlist/
      hoverlist.go                            (NewDelegate gains zm; Render uses strings.Builder + zm.Mark; delete Layout struct)
```

---

## Out of Scope

- `handleMouseWheel` coordinate check in `dashboard.go` — not a hit-test site; not buggy.
- `ServiceListBounds` / `LogViewBounds` on `PaneLayout` — serve layout sizing, not hit-testing; retained as-is.
- Adding new unit tests for zone-based hit-testing — follow-on work.
- Log streamer injection (covered by `refactor-2026-05-13-log-streamer-injection.md`).

---

## Post-Implementation

When all implementation steps are complete and the build passes:

1. Move this file to `docs/plans/archive/`
