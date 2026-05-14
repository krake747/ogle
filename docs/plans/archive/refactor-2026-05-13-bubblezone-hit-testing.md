# refactor: bubblezone Hit-Testing (Part 2 of 2)

Status: **Ready for implementation.** All design decisions resolved. No open questions.

Requires: `refactor-2026-05-13-bubblezone-wiring.md` complete and build passing.

---

## Context

This plan completes the replacement of manual mouse hit-testing with
`bubblezone/v2`. Part 1 (wiring) added the dependency, threaded
`*zone.Manager` through all constructors, marked zones in rendering, and wired
`zone.Scan` at the root. By the start of this plan, zones are being emitted but
nothing consumes them yet.

This plan replaces all three hit-test sites, fixes a drag-origin bug in
`fileselect`, and deletes the now-unused `hoverlist.Layout` struct.

Zone names in use (established in Part 1): `"item-{i}"` per list item;
`"pane-left"` and `"pane-right"` for the two dashboard panes.

---

## Decision Log

| # | Decision |
|---|---|
| 4 | Zone names: `"item-{i}"` in `hoverlist.delegate.Render`; `"pane-left"` and `"pane-right"` in `PaneLayout.View`. (Established in Part 1; reproduced here for reference.) |
| 5 | `hoverlist.Layout` struct and its `HitTest` method are deleted after all call sites are replaced (step 7). `ServiceListBounds` and `LogViewBounds` on `PaneLayout` are retained — they serve layout sizing, not hit-testing. |
| 6 | `fileselect.Model` gains a `pressedIdx int` field (default `-1`). Set on `tea.MouseClickMsg`; `msgs.FileSelected` is only emitted on `tea.MouseReleaseMsg` when the release zone matches `pressedIdx`. This fixes the drag-origin bug. |
| 7 | `handleMouseWheel` coordinate check in `dashboard.go` is intentionally NOT replaced — it is not one of the three targeted hit-test sites and is not buggy. |
| 8 | In fullscreen/zoom mode `"pane-left"` zone will not exist; `zm.Get("pane-left").InBounds(x, y)` returns false naturally — no special case needed. |
| 12 | `DragCoordinator.hitTestComponent` (in `selection.go`) replaces the `ServiceListBounds`/`LogViewBounds` checks with zone lookups for `"pane-left"` and `"pane-right"`. The `SelectionFooter` y-coordinate check (`y == paneH`) is preserved — the footer help bar is not a rendered zone. |

---

## Implementation Findings

| # | Finding |
|---|---|
| F1 | `selection_test.go` calls `states.NewPaneLayout(nil)` (no zm arg yet) and constructs `var dc states.DragCoordinator` as a zero value. After step 6, `hitTestComponent` calls `dc.zm.Get(...)` — nil `zm` panics. Fix: add an exported `SetZoneManager(*zone.Manager)` method to `DragCoordinator` (analogous to `SetLastPress`/`SetActiveDrag`) and rewrite `TestDragCoordinator_HandleMotion_AboveThreshold_StartsDrag` to set up a fake zone via `zm.Mark`/`zm.Scan` before exercising `HandleMotion`. Do this as part of step 6. |
| F2 | `project.Model` does **not** need to store `zm`. `project.New` passes it through to `states.NewDashboard` at construction time; `project.Model.Update` never calls `NewDashboard` directly. |
| F6 | Removing `layout hoverlist.Layout` from `servicelist.Model` also requires removing the `m.layout.Width = w` line in `SetBounds` and the `m.layout.*` assignments. In `fileselect`, the `WindowSizeMsg` handler also sets `m.layout.Width = sz.Width` — that line must be removed in step 5. |

---

## Implementation Steps

Each step must leave the build passing before the next begins.

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
`SetBounds` assignments to `m.layout.OriginX` / `m.layout.OriginY` (F6).

Run `go build ./...`.

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
   to `m.layout.OriginX` / `m.layout.OriginY`, and the `m.layout.Width = sz.Width`
   line in the `WindowSizeMsg` handler (F6).

Run `go build ./...`.

### Step 6 — Replace `DragCoordinator.hitTestComponent` with zone lookup

In `internal/ui/flows/dashboard/project/states/selection.go`, replace the
`hitTestComponent` method body. The zone lookups replace the two bounding-rect
checks; the `SelectionFooter` y-coordinate check is **preserved**:

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

Per F1: add `SetZoneManager(zm *zone.Manager)` to `DragCoordinator` and update
`TestDragCoordinator_HandleMotion_AboveThreshold_StartsDrag` in
`selection_test.go` to call `dc.SetZoneManager(zm)` and set up a fake zone via
`zm.Mark`/`zm.Scan` before exercising `HandleMotion`.

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
        project/
          states/
            dashboard.go                      (pass zm to newDragCoordinator)
            selection.go                      (DragCoordinator gains zm; replace hitTestComponent; SetZoneManager)
            selection_test.go                 (update test to use SetZoneManager + fake zone)
    components/
      servicelist/
        servicelist.go                        (replace hitTest; remove layout field)
    views/
      fileselect/
        fileselect.go                         (replace hitTest; fix drag-origin bug; remove layout field)
    hoverlist/
      hoverlist.go                            (delete Layout struct and HitTest method)
```

---

## Out of Scope

- `handleMouseWheel` coordinate check in `dashboard.go` — not a hit-test site; not buggy.
- `ServiceListBounds` / `LogViewBounds` on `PaneLayout` — serve layout sizing, not hit-testing; retained as-is.
- Adding new unit tests for zone-based hit-testing — follow-on work.
- Log streamer injection (covered by `refactor-2026-05-13-log-streamer-injection.md`).

---

## Post-Implementation

When all steps are complete and the build passes:

1. Move this file to `docs/plans/archive/`.
2. Move `refactor-2026-05-13-bubblezone-wiring.md` to `docs/plans/archive/` if not already done.
