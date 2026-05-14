# refactor: bubblezone Wiring (Part 1 of 2)

Status: **Ready for implementation.** All design decisions resolved. No open questions.

Prerequisite for: `refactor-2026-05-13-bubblezone-hit-testing.md`

---

## Context

Three locations in the UI perform manual mouse hit-testing by comparing raw
pixel coordinates against stored bounding rectangles. `bubblezone/v2` (import
path `github.com/lrstanley/bubblezone/v2`) replaces all three sites with named
zones.

This plan covers the infrastructure half: adding the dependency, threading
`*zone.Manager` through all constructors, marking zones in rendering, and
wiring `zone.Scan` at the root. After this plan the build passes and zones are
being marked and scanned — but nothing consumes them yet. Hit-test replacement
happens in Part 2.

---

## Decision Log

| # | Decision |
|---|---|
| 1 | `bubblezone/v2` is not yet a dependency — step 1 adds it via `go get`. |
| 2 | A single `*zone.Manager` is created in the root `dashboard.Model` constructor (`dashboard.go`) and threaded down through all sub-model constructors. It is not a global. |
| 3 | `zone.Scan()` is called on `v.Content` in the root `dashboard.Model.View()`. `View()` returns `tea.View`; the correct form is `v.Content = zm.Scan(v.Content)` followed by `return v`. One frame delay is acceptable and expected. |
| 4 | Zone names: `"item-{i}"` in `hoverlist.delegate.Render`; `"pane-left"` and `"pane-right"` in `PaneLayout.View`. No prefix collisions because `fileselect` and `servicelist` are never rendered simultaneously. |
| 9 | `startup.Model` and all startup states (`scanning`, `watching`, `selecting`, `handler`) that construct or thread `fileselect.Model` receive the `zm *zone.Manager` parameter. `startup.Model.Update`'s `msgs.WatcherError` case (which calls `states.NewWatchingWithError`) must also pass `m.zm`. |
| 10 | `hoverlist.NewDelegate` gains a `zm *zone.Manager` parameter stored on the unexported `delegate` struct. `servicelist.New` and `fileselect.New` — which are already receiving `zm` by step 2 — pass it through to `NewDelegate`. |
| 11 | `hoverlist.delegate.Render` captures `DefaultDelegate.Render` output into a `strings.Builder` (via `io.Writer`), then writes `zm.Mark(name, buf.String())` to `w`. This is the only correct way to inject zone marks into a `Render(w io.Writer, ...)` method. |

---

## Implementation Findings

| # | Finding |
|---|---|
| F3 | `Selecting` gaining a `zm` field (step 2) is redundant: `Selecting.handler` (a `fileHandler`) already carries `zm`, and `Selecting` never calls `fileselect.New` directly. Adding the field requires `zm: s.zm` / `zm: fh.zm` at all four inline struct-literal sites (`withError`, `withParsing`, two `Update` cases). Retain the field to match the plan; annotate inline literals accordingly. |
| F4 | `PaneLayout.View` has two branches: `modeSplit` (both `leftInner` + `rightInner`) and `modeLogFullscreen` (only `rightInner`). Mark `pane-right` in **both** branches so drag detection works in fullscreen. `pane-left` is only marked in `modeSplit`; decision 8 in Part 2 (returns false naturally) handles the fullscreen case. |
| F5 | `hoverlist.go` currently imports only `"io"`. Step 3 needs `"fmt"` and `"strings"` added. |

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

Per F4: mark `pane-right` in both `modeSplit` and `modeLogFullscreen` branches.
Mark `pane-left` only in `modeSplit`.

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

Add `"strings"` and `"fmt"` to the `hoverlist` imports (F5).

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
            dashboard.go                      (store zm; pass to servicelist, newPaneLayout)
            settings.go                       (pass zm through both NewDashboard calls)
            layout.go                         (receive zm; mark pane-left/pane-right in View)
      startup/
        startup.go                            (receive and store zm; thread to states)
        states/
          scanning.go                         (receive zm)
          watching.go                         (receive zm; WatcherError case passes zm)
          handler.go                          (fileHandler gains zm; passes to fileselect.New)
          selecting.go                        (Selecting gains zm)
    components/
      servicelist/
        servicelist.go                        (receive zm; pass to NewDelegate)
    views/
      fileselect/
        fileselect.go                         (receive zm; store zm)
    hoverlist/
      hoverlist.go                            (NewDelegate gains zm; Render uses strings.Builder + zm.Mark)
```

---

## Post-Implementation

When all steps are complete and the build passes, continue with
`refactor-2026-05-13-bubblezone-hit-testing.md`.

Move this file to `docs/plans/archive/` when Part 2 is also complete.
