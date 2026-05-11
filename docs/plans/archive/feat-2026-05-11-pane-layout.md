# feat: pane layout

Status: **Ready for implementation.** All design decisions resolved. No open questions.

---

## Context

The Dashboard in `internal/ui/flows/dashboard/project/states/dashboard.go` renders a
two-pane horizontal split (service list left, log view right) using
`lipgloss.JoinHorizontal`. The current implementation has two problems:

1. Layout arithmetic (`leftW`, `rightW`, `paneH`, `innerH`) is duplicated between
   `SetSize` and `View`. Any formula change must be applied in both places.
2. `JoinHorizontal` produces a string and discards coordinate information. Pane bounds
   must be re-derived independently for mouse hit-testing — required by the mouse support
   plan, which calls `SetBounds` on `servicelist`.

The Dashboard also needs a second layout mode — **log-fullscreen** — where the log view
occupies the full terminal width and the service list is hidden.

---

## Decision Log

All decisions were reached through a design interview. Do not re-open them without a
specific technical reason.

| # | Decision |
|---|---|
| 1 | Layout arithmetic and bounds computation are extracted into an unexported `paneLayout` type in the `states` package. |
| 2 | `paneLayout` is **geometry-only**: it computes bounds rects and drives `JoinHorizontal` internally. It does not hold component references or route updates. `Dashboard` remains the single orchestrator of component lifecycle. |
| 3 | Two named layout modes: `modeSplit` (30/70 ratio, capped at 80 cols) and `modeLogFullscreen` (right pane takes 100%, left hidden). |
| 4 | The mode toggle key is `z`. Pressing `z` in either mode toggles to the other. |
| 5 | Focus is coupled to mode: entering `modeLogFullscreen` auto-sets `d.focus = focusRight`; returning to `modeSplit` auto-sets `d.focus = focusLeft`. |
| 6 | Help bar label is dynamic: `z: fullscreen` in split mode, `z: split` in fullscreen mode. |
| 7 | All duplicate layout arithmetic in `View` is eliminated — `View` reads geometry from `paneLayout`. |

---

## Implementation Steps

Each step must leave `go build ./...` and `go test ./...` passing before the next begins.

### Step 1 — Define `paneLayout`

Create `internal/ui/flows/dashboard/project/states/layout.go`.

Move the layout constants from `dashboard.go` into this file:
`servicePaneRatio`, `servicePaneRatioDen`, `servicePaneMaxW`, `borderWidth`,
`borderHeight`, `separatorRows`, `helpBarHeight`.

Define:

```go
type layoutMode int

const (
    modeSplit         layoutMode = iota
    modeLogFullscreen
)

type rect struct{ x, y, w, h int }

type paneLayout struct {
    mode layoutMode
    w, h int
}
```

Implement the following methods:

**`newPaneLayout() paneLayout`** — returns zero-value layout in `modeSplit`.

**`(p paneLayout) SetSize(w, h int) paneLayout`** — stores `w` and `h`, returns updated value.

**`(p paneLayout) ToggleMode() paneLayout`** — flips between `modeSplit` and `modeLogFullscreen`.

**`(p paneLayout) IsLogFullscreen() bool`** — returns `p.mode == modeLogFullscreen`.

**`(p paneLayout) ServiceListBounds() rect`** — in `modeSplit`, returns the content area
of the left pane (accounting for the border: `x=1, y=1`). In `modeLogFullscreen`, returns
`rect{}` — the service list is not visible.

**`(p paneLayout) LogViewBounds() rect`** — in `modeSplit`, returns the content area of
the right pane (x offset = left pane outer width + 1 for right border). In
`modeLogFullscreen`, returns the full terminal content area (`x=1, y=1`, full width minus
border).

**`(p paneLayout) View(serviceListStr, logViewStr string, leftFocused bool) string`** —
renders both panes with `NormalBorder()`, applies `lipgloss.Color("62")` to the focused
pane border and `lipgloss.Color("240")` to the dimmed one, then joins with
`JoinHorizontal`. In `modeLogFullscreen`, renders only the right pane at full terminal
width. Does not render the help bar — that remains in `Dashboard.View`.

Build must pass after this step.

---

### Step 2 — Wire `paneLayout` into `Dashboard`

In `internal/ui/flows/dashboard/project/states/dashboard.go`:

1. Add `layout paneLayout` to `Dashboard`. Initialise with `newPaneLayout()` in
   `NewDashboard`.
2. In `SetSize`: replace the manual arithmetic with:
   ```go
   d.layout = d.layout.SetSize(w, h)
   b := d.layout.ServiceListBounds()
   d.serviceList = d.serviceList.SetSize(b.w, b.h)
   ```
   Note: `SetBounds` does not yet exist on `servicelist` — the mouse support plan renames
   `SetSize` to `SetBounds` and adds `x, y` tracking. At that point this call site
   becomes `d.serviceList.SetBounds(b.x, b.y, b.w, b.h)`.
3. In `View`: remove the duplicate arithmetic and direct `lipgloss` calls. Replace with:
   ```go
   return d.layout.View(d.serviceList.View(), rightContent, d.focus == focusLeft) +
       "\n" + d.help.View(km)
   ```
   where `rightContent` remains the `"logs"` placeholder.
4. Remove the constants that moved to `layout.go`.

Build must pass after this step.

---

### Step 3 — Add `z` keybinding and mode toggle

In `dashboard.go`:

1. Add `Zoom key.Binding` to `dashboardKeyMap`:
   ```go
   Zoom: key.NewBinding(
       key.WithKeys("z"),
       key.WithHelp("z", "fullscreen"),
   )
   ```
2. In `Update`, handle `z`. Guard with `!d.serviceList.IsFiltering()` (same guard as
   `q`) to avoid accidental zoom during filter input:
   ```go
   if key.Matches(keyMsg, d.keys.Zoom) && !d.serviceList.IsFiltering() {
       d.layout = d.layout.ToggleMode()
       if d.layout.IsLogFullscreen() {
           d.focus = focusRight
       } else {
           d.focus = focusLeft
       }
       b := d.layout.ServiceListBounds()
       d.serviceList = d.serviceList.SetSize(b.w, b.h)
   }
   ```

Build must pass after this step.

---

### Step 4 — Update help bar

In `dashboard.go`, in `View`, update the `Zoom` binding label before constructing `km`:

```go
zoomHelp := "fullscreen"
if d.layout.IsLogFullscreen() {
    zoomHelp = "split"
}
d.keys.Zoom = key.NewBinding(key.WithKeys("z"), key.WithHelp("z", zoomHelp))
```

Add `Zoom` to `combinedKeyMap.ShortHelp()`. `key.Binding` is a value type; mutating the
label per render is safe.

Build and all tests must pass after this step.

---

### Step 5 — Smoke-test manually

Run `go build ./...` and `go test ./...`. Then run the app and verify:

- Dashboard renders the two-pane split identically to before.
- Pressing `z` switches to log-fullscreen: right pane fills the terminal, service list
  disappears.
- Pressing `z` again returns to split.
- Help bar shows `z: fullscreen` in split mode, `z: split` in fullscreen mode.
- `q` quits in both modes.
- Service list keyboard navigation and filter work normally in split mode.
- Pressing `z` while the filter is active has no effect.

---

## Out of Scope

- The log view component (viewport, log streaming) — the right pane remains the `"logs"`
  placeholder string.
- Mouse support — `SetBounds` rename and x/y coordinate propagation are handled by the
  mouse support plan. This plan and the mouse support plan are independent and can be
  implemented in either order; when both are done, the `SetSize` call sites in Step 2 and
  Step 3 above become `SetBounds(b.x, b.y, b.w, b.h)`.
- Animated pane transitions.
- A third mode (service-list fullscreen).
- Resize handle or continuously adjustable split ratio.
- Theme injection into `paneLayout` — the two hardcoded `lipgloss.Color` values move from
  `dashboard.go` into `layout.go` but remain hardcoded until the themes plan is
  implemented.

---

## Post-Implementation

When all implementation steps are complete and the build passes:

1. Move this file to `docs/plans/archive/`
