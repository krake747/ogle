# Charm Ecosystem Compatibility

ogle runs on `charm.land/bubbletea/v2` — the v2 API, which has breaking changes from
`github.com/charmbracelet/bubbletea` v1 (different message types, `tea.View`, `tea.Cmd`
signatures, etc.). Libraries that import the old v1 module path are link-incompatible.

This document records the compatibility verdict for every charm-adjacent library so future
contributors don't re-investigate.

## Compatibility table

| Library | Import path | v2 compatible | Notes |
|---|---|---|---|
| bubbletea | `charm.land/bubbletea/v2` | ✅ **in use** | runtime |
| bubbles | `charm.land/bubbles/v2` | ✅ **in use** | list, help, key, paginator |
| lipgloss | `charm.land/lipgloss/v2` | ✅ **in use** | all terminal styling |
| huh | `charm.land/huh/v2` | ✅ compatible | interactive forms; prefer over hand-rolling (see below) |
| glamour | `charm.land/glamour/v2` | ✅ compatible | markdown rendering; no current use case |
| bubblezone | `github.com/lrstanley/bubblezone/v2` | ✅ compatible | mouse zone tracking; candidate to replace manual hit-testing |
| harmonica | `github.com/charmbracelet/harmonica` | ✅ compatible | pure math, zero deps; spring-physics animations |
| ntcharts | `github.com/NimbleMarkets/ntcharts/v2` | ✅ compatible | terminal charts; no current use case (see below) |
| glow | application, not a library | — | not embeddable; glamour is the underlying library |

> **ntcharts import path:** the `main` branch is v1 (`github.com/charmbracelet/bubbletea`).
> The **default** branch is `v2` — use `github.com/NimbleMarkets/ntcharts/v2`.
>
> **bubblezone import path:** v2 is a separate module — use
> `github.com/lrstanley/bubblezone/v2`, not `github.com/lrstanley/bubblezone`.

## Per-library notes

### huh (`charm.land/huh/v2`)

A form library: field focus cycling, Tab/Shift-Tab navigation, Enter/Esc confirm/cancel,
rendering. **Prefer it over hand-rolling any new interactive overlay that resembles a form.**

Current fit in ogle:

- The **Settings** overlay (`states/settings.go`) is a 273-line hand-rolled state machine
  covering the same concerns. `huh.Select` maps cleanly to the Theme field; the
  step-based arrow-key UX for Poll Interval and Log Buffer Cap does not have a native huh
  type, and the live-theme-preview side-effect is bespoke. Retrofitting Settings is a
  moderate refactor with only partial gain — call that a separate decision.
- Any **new** form-like interaction (Log Filter text input, Explicit File path prompt, etc.)
  should use huh rather than a custom implementation.

### glamour (`charm.land/glamour/v2`)

Renders markdown to styled terminal output. No current use case — ogle does not render
markdown anywhere. Reach for it if description fields, help text, or label values ever
contain markdown.

### bubblezone (`github.com/lrstanley/bubblezone/v2`)

Declarative mouse-zone tracking: mark zones at render time, hit-test by name on mouse
events. Would replace:

- `hoverlist.Layout.HitTest` (`internal/ui/hoverlist/hoverlist.go:71`) — manual coordinate math
- `dashboard.hitTestComponent` (`internal/ui/flows/dashboard/project/states/dashboard.go:864`) — manual bounds checking
- `fileselect.hitTest` (`internal/ui/views/fileselect/fileselect.go:144`) — wrapper around hoverlist

Also resolves the known mouse-release bug in `fileselect` (emits `msgs.FileSelected` on any
`tea.MouseReleaseMsg` regardless of drag origin) as a side-effect, because zones carry
enter/exit semantics.

Tradeoff: each component's `View()` must inject zone markers, and `zone.Scan()` must be
called on the root output. Touches the render pipeline.

### harmonica (`github.com/charmbracelet/harmonica`)

Spring-physics math library — zero dependencies, no bubbletea import. Compatible with any
runtime by construction. You drive it with `tea.Every` ticks and call `spring.Update()` each
frame to animate a value toward a target.

Candidate use case: smooth log-pane scroll (currently `logScrollRows` snaps instantly).
Adds a constant tick overhead during active scroll animation. Low priority.

### ntcharts (`github.com/NimbleMarkets/ntcharts/v2`)

Terminal charts (sparklines, bar charts, line charts). No current use case — State Polling
is not yet implemented so there is no runtime time-series data. When State Polling lands,
ntcharts is the natural reach for visualising Service State history or restart frequency in
the Service Inspector.

Also depends on `github.com/lrstanley/bubblezone/v2` internally, so both arrive together.
