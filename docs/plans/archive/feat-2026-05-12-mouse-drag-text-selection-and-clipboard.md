# feat: Mouse Drag Text Selection and Clipboard

Status: **Ready for implementation.** All design decisions resolved. No open questions.

---

## Context

ogle currently uses `github.com/atotto/clipboard` (an indirect dependency) to copy
label values to the clipboard in the Service Inspector's Label section. This approach
requires platform-specific clipboard APIs (X11, Wayland, macOS) and has no
cross-terminal support.

Users want to be able to select and copy any visible text in the TUI — service names,
detail header fields, log output, label values, the keybind footer — by clicking and
dragging with the mouse, without holding Shift. Native terminal Shift+drag was
considered but ruled out: it cannot be constrained to component boundaries and always
spans the full terminal width.

This plan:
- Replaces `atotto/clipboard` with `tea.SetClipboard` (OSC52 — built into bubbletea v2,
  no third-party dependency)
- Implements a Dashboard-owned drag selection layer that detects left-button drag
  gestures, extracts plain text from the rows spanned within the originating component,
  and copies to clipboard via `tea.SetClipboard`

---

## Codebase Reference

**Mouse events (bubbletea v2):**
```
tea.MouseClickMsg   — left button pressed
tea.MouseMotionMsg  — mouse moved; .Button != 0 when a button is held (drag)
tea.MouseReleaseMsg — left button released
```

**Clipboard (bubbletea v2 — no external dep):**
```go
tea.SetClipboard(s string) tea.Cmd  // writes via OSC52
```

**ANSI stripping (already a transitive dep via bubbletea):**
```go
import "github.com/charmbracelet/x/ansi"
ansi.Strip(s string) string  // strips escape sequences, leaves plain text
```

**Component bounds (existing):**
```go
// states/layout.go
d.layout.ServiceListBounds() rect  // {x, y, w, h} of service list content area
d.layout.LogViewBounds()    rect   // {x, y, w, h} of right pane content area
```

**Footer row in rendered output:**
```go
paneH := d.layout.h - separatorRows - helpBarHeight  // = d.layout.h - 2
// footer is at string-line index paneH in the renderFull() output
```

---

## Decision Log

All decisions were reached through a design interview. Do not re-open them without a
specific technical reason.

| # | Decision |
|---|---|
| 1 | Replace `atotto/clipboard` with `tea.SetClipboard` (OSC52). Remove `atotto/clipboard` as a direct dependency; it may remain as an indirect dep via other packages but must not be imported directly. |
| 2 | Copy is triggered by left-button click-drag (no Shift key, no visual selection cursor or markers). The gesture is entirely mouse-driven. |
| 3 | Scope covers all visible text regions: service list, detail header (inspector), Log Stream area, Label section, and the keybind footer. |
| 4 | Selection granularity is **row-level**: all complete rows spanned by the drag are copied as plain text (one row per line). Character-within-row precision is not required. |
| 5 | Click vs drag is distinguished by a **1-cell movement threshold**. Any `MouseMotionMsg` with `.Button == tea.MouseLeft` that moves more than 1 cell from the press origin activates drag selection. A release with ≤1 cell movement is treated as a click and forwarded to the normal TUI interaction. |
| 6 | The Dashboard state owns all drag selection logic. It intercepts mouse messages before forwarding to child components. On drag release it calls `tea.SetClipboard`; on click release it forwards to components unchanged. |
| 7 | Selection is **constrained to the component where the drag started** (determined by hit-testing the press origin against component bounds). The end cell is clamped to that component's row range. A drag that starts on a pane border or outside any content area is ignored. |
| 8 | **Visual feedback**: the rows under active selection are rendered with full-row inversion during drag. In split mode this inverts both panes across the selected rows; this is an accepted visual imprecision for a transient gesture. Character-column-precise highlighting is out of scope. |
| 9 | The existing label drag-to-copy in `inspector/labels.go` is replaced by the Dashboard-level drag mechanism. Clipboard writes go through `tea.SetClipboard` at the Dashboard level only. |
| 10 | `dragIdx` in `labelsModel` is renamed to `pressIdx` (not deleted). It is needed to confirm Ctrl+click released on the same row as the press for URL-open. `dragX` and the drag-distance copy block are removed. |
| 11 | A `renderFull()` private method is added to `Dashboard`. It assembles the complete rendered string (panes + `"\n"` + help bar). Both `View()` and `extractSelection()` call `renderFull()`, ensuring both operate on the identical string. `applySelectionHighlight` receives the full `renderFull()` output. |
| 12 | The footer (`selectionFooter`) is in scope. Its row index in the `renderFull()` output is `paneH = d.layout.h - separatorRows - helpBarHeight`, not `d.layout.h - 1`. Both `hitTestComponent` and `boundsForComponent` use the `paneH` formula. This must be verified empirically during implementation since it depends on lipgloss not emitting a trailing newline on the pane render. |
| 13 | The `d.keys.Zoom` help-text mutation is moved from `View()` to `Update()`, inside the zoom key press handler, where it is set immediately after `d.layout.ToggleMode()`. `renderFull()` reads `d.keys` without mutating. |
| 14 | Text extraction uses the **component's own `View()` output** (not the full `renderFull()` string). `extractSelection()` switches on `d.drag.component` to get the source string, then adjusts drag row indices to component-local coordinates (`localRow = terminalRow - b.y`). This avoids x-range slicing and the UTF-8 column-vs-byte-offset problem caused by border characters. |
| 15 | Visual highlighting (`applySelectionHighlight`) uses full-row inversion on the `renderFull()` string. No x-range slicing. Accepted imprecision: in split mode, both panes are inverted for selected rows. |

---

## Target Structure

No new packages are introduced. Changes are localised to:

```
internal/
├── ui/
│   ├── components/
│   │   └── inspector/
│   │       └── labels.go          ← rename dragIdx→pressIdx; remove dragX, drag-copy
│   └── flows/
│       └── dashboard/
│           └── project/
│               └── states/
│                   ├── dashboard.go   ← add drag state; renderFull(); footerView();
│                   │                    move keys.Zoom mutation; intercept mouse msgs
│                   └── selection.go   ← new: dragSelection, selectionComponent,
│                                          extractText, abs, clamp
go.mod                                 ← remove atotto/clipboard direct dependency
```

---

## Implementation Steps

Each step must leave the build passing before the next begins.

### Step 1 — Update labels.go

In `internal/ui/components/inspector/labels.go`:

- Remove the `github.com/atotto/clipboard` import.
- Remove `copyToClipboardCmd`.
- Rename `dragIdx int` → `pressIdx int` (init `-1`). Remove `dragX int`.
- In `MouseClickMsg`: set `m.pressIdx = idx` (was `m.dragIdx`). Remove `m.dragX = msg.X`.
- In `MouseReleaseMsg`: check `m.pressIdx >= 0`; compare `pressIdx == releaseIdx` for
  Ctrl+click URL-open. Remove the `abs(msg.X - m.dragX) > 0` drag-distance copy block
  entirely. Reset `m.pressIdx = -1`.
- Keep `openURLCmd` — Ctrl+click URL-open is unchanged.

In `go.mod`: remove `github.com/atotto/clipboard` from the direct `require` block.
Run `go mod tidy`.

### Step 2 — Create selection.go

Create `internal/ui/flows/dashboard/project/states/selection.go`:

```go
package states

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// dragSelection tracks an in-progress mouse drag selection.
type dragSelection struct {
	active    bool
	startX    int
	startY    int
	endY      int
	component selectionComponent
}

// selectionComponent identifies which component owns the active drag.
type selectionComponent int

const (
	selectionNone        selectionComponent = iota
	selectionServiceList
	selectionInspector
	selectionFooter
)

// rows returns the [min, max] row range covered by the selection (inclusive).
func (s dragSelection) rows() (int, int) {
	if s.startY <= s.endY {
		return s.startY, s.endY
	}
	return s.endY, s.startY
}

// extractText takes component-local lines, a local row range, and component
// bounds and returns plain text for the clipboard. Lines are ANSI-stripped and
// trimmed of trailing whitespace.
func extractText(lines []string, minRow, maxRow int, bounds rect) string {
	var sb strings.Builder

	for row := minRow; row <= maxRow; row++ {
		if row < bounds.y || row >= bounds.y+bounds.h {
			continue
		}
		if row >= len(lines) {
			break
		}
		plain := strings.TrimRight(ansi.Strip(lines[row]), " ")
		sb.WriteString(plain)
		if row < maxRow {
			sb.WriteByte('\n')
		}
	}

	return sb.String()
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
```

### Step 3 — Add drag state and helpers to Dashboard

In `states/dashboard.go`, add three fields to `Dashboard`:

```go
drag       dragSelection
lastPressX int
lastPressY int
```

Add `renderFull()` and `footerView()` helpers (no state mutation):

```go
func (d *Dashboard) renderFull() string {
	return d.layout.View(d.serviceList.View(), d.inspector.View(), d.focus == focusLeft) +
		"\n" + d.footerView()
}

func (d *Dashboard) footerView() string {
	km := combinedKeyMap{dashboard: d.keys, list: d.serviceList.KeyMap()}
	return d.help.View(km)
}
```

Replace `View()` with a non-mutating version:

```go
func (d *Dashboard) View() string {
	if d.layout.w == 0 || d.layout.h == 0 {
		return ""
	}
	full := d.renderFull()
	if d.drag.active {
		full = d.applySelectionHighlight(full)
	}
	return full
}
```

### Step 4 — Move keys.Zoom mutation to Update()

In the zoom key press handler in `Update()`, after `d.layout.ToggleMode()`:

```go
if d.layout.IsLogFullscreen() {
	d.keys.Zoom = key.NewBinding(key.WithKeys("z"), key.WithHelp("z", "split"))
	d.focus = focusRight
} else {
	d.keys.Zoom = key.NewBinding(key.WithKeys("z"), key.WithHelp("z", "fullscreen"))
	d.focus = focusLeft
}
```

Verify `defaultDashboardKeys.Zoom` is initialised with `"fullscreen"` help text (default
mode is split — already correct in the existing code). Remove any `keys.Zoom` mutation
from the old `View()`.

### Step 5 — Intercept mouse messages in Dashboard.Update

Add cases **before** the fall-through to inspector and service list. Also add helper
methods `hitTestComponent`, `boundsForComponent`, `extractSelection`, and
`applySelectionHighlight`.

```go
case tea.MouseClickMsg:
	if msg.Button == tea.MouseLeft {
		d.lastPressX = msg.X
		d.lastPressY = msg.Y
		d.drag = dragSelection{}
	}

case tea.MouseMotionMsg:
	if msg.Button == tea.MouseLeft {
		dx := msg.X - d.lastPressX
		dy := msg.Y - d.lastPressY
		if !d.drag.active && (abs(dx) > 1 || abs(dy) > 1) {
			comp := d.hitTestComponent(d.lastPressX, d.lastPressY)
			if comp != selectionNone {
				d.drag = dragSelection{
					active:    true,
					startX:    d.lastPressX,
					startY:    d.lastPressY,
					endY:      msg.Y,
					component: comp,
				}
			}
		} else if d.drag.active {
			b := d.boundsForComponent(d.drag.component)
			d.drag.endY = clamp(msg.Y, b.y, b.y+b.h-1)
		}
		if d.drag.active {
			return d, nil
		}
	}

case tea.MouseReleaseMsg:
	if msg.Button == tea.MouseLeft && d.drag.active {
		text := d.extractSelection()
		d.drag = dragSelection{}
		if text != "" {
			return d, tea.SetClipboard(text)
		}
		return d, nil
	}
	d.drag = dragSelection{}
```

Helper methods:

```go
func (d *Dashboard) hitTestComponent(x, y int) selectionComponent {
	lb := d.layout.ServiceListBounds()
	if x >= lb.x && x < lb.x+lb.w && y >= lb.y && y < lb.y+lb.h {
		return selectionServiceList
	}
	rb := d.layout.LogViewBounds()
	if x >= rb.x && x < rb.x+rb.w && y >= rb.y && y < rb.y+rb.h {
		return selectionInspector
	}
	paneH := d.layout.h - separatorRows - helpBarHeight
	if y == paneH {
		return selectionFooter
	}
	return selectionNone
}

func (d *Dashboard) boundsForComponent(c selectionComponent) rect {
	switch c {
	case selectionServiceList:
		return d.layout.ServiceListBounds()
	case selectionInspector:
		return d.layout.LogViewBounds()
	case selectionFooter:
		paneH := d.layout.h - separatorRows - helpBarHeight
		return rect{x: 0, y: paneH, w: d.layout.w, h: 1}
	default:
		return rect{}
	}
}

// extractSelection uses each component's own View() output to avoid x-range
// slicing across split-pane terminal rows (Decision 14).
func (d *Dashboard) extractSelection() string {
	b := d.boundsForComponent(d.drag.component)
	minRow, maxRow := d.drag.rows()
	localMin := minRow - b.y
	localMax := maxRow - b.y

	var source string
	switch d.drag.component {
	case selectionServiceList:
		source = d.serviceList.View()
	case selectionInspector:
		source = d.inspector.View()
	case selectionFooter:
		source = d.footerView()
	default:
		return ""
	}

	lines := strings.Split(source, "\n")
	return extractText(lines, localMin, localMax, rect{x: 0, y: 0, w: b.w, h: b.h})
}
```

Import `"strings"` and `"github.com/charmbracelet/x/ansi"` in `dashboard.go`.

### Step 6 — Visual highlight overlay

```go
func (d *Dashboard) applySelectionHighlight(rendered string) string {
	lines := strings.Split(rendered, "\n")
	minRow, maxRow := d.drag.rows()
	b := d.boundsForComponent(d.drag.component)
	highlight := lipgloss.NewStyle().Reverse(true)

	for row := minRow; row <= maxRow; row++ {
		if row < b.y || row >= b.y+b.h || row >= len(lines) {
			continue
		}
		lines[row] = highlight.Render(ansi.Strip(lines[row]))
	}

	return strings.Join(lines, "\n")
}
```

Full-row inversion — no x-range slicing (Decision 15).

### Step 7 — Confirm labels.go is clean

Verify `labelsModel` has no `dragIdx`, `dragX`, or `atotto/clipboard` references.
Confirm the inspector package builds cleanly after Step 1.

### Step 8 — go mod tidy

```bash
go mod tidy
```

Confirm `atotto/clipboard` is no longer a direct dependency. It may remain as
`// indirect` if another transitive dep pulls it in; that is acceptable.

---

## Out of Scope

- **Character-within-row precision** — selection granularity is row-level only.
- **Column-precise visual highlight** — in split mode the full terminal row inverts.
  Character-column-precise highlighting requires unicode-aware column slicing not
  available in current transitive deps.
- **Keyboard-driven selection** (vim-style visual mode).
- **Multi-component spanning** — a drag is constrained to its origin component.
- **Primary clipboard (X11)** — `tea.SetClipboard` writes to the system clipboard only.
- **Log Stream live drag selection** — the Log Stream is a placeholder; rows are treated
  as static text. Scroll-aware log line selection is deferred to the Log Stream plan.

---

## Post-Implementation

When all implementation steps are complete and the build passes:

1. Move this file to `docs/plans/archive/`
