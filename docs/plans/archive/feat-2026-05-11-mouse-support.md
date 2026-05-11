# feat: mouse support

Status: **Ready for implementation.** All design decisions resolved. No open questions.

---

## Context

ogle is a terminal UI built with Bubble Tea and the `charm.land/bubbles/v2` list component.
Mouse mode `tea.MouseModeCellMotion` is already enabled at the root. Both `fileselect` and
`servicelist` use `list.Model` internally, but the Bubbles list handles **zero** mouse events —
no hover, no click. All mouse messages are silently discarded.

There is also an existing bug in `fileselect/fileselect.go:119–124`: `MouseReleaseMsg`
unconditionally emits `msgs.FileSelected` for whatever item is currently highlighted, regardless
of whether the mouse was anywhere near the list. This plan fixes that bug as a prerequisite.

---

## Decision Log

All decisions were reached through a design interview. Do not re-open them without a specific
technical reason.

| # | Decision |
|---|---|
| 1 | Hover is a **separate visual highlight** — it does not move the keyboard cursor or emit selection messages. |
| 2 | Terminal position is injected by the parent via a new `SetBounds(x, y, w, h int)` method, replacing `SetSize`. |
| 3 | A single **left-click (`MouseReleaseMsg`, left button)** on a `fileselect` item immediately emits `msgs.FileSelected` and proceeds to parsing. |
| 4 | A single left-click on a `servicelist` item calls `list.Select(index)`, which triggers the existing `lastSelected` comparison and emits `msgs.ServiceSelected` as normal. |
| 5 | Hover visual treatment: **subtle background tint** on the item row(s), distinct from the pink/purple selection indicator. |
| 6 | X-axis bounds are checked — hover clears when the mouse is outside the component's column range. |
| 7 | Hover and click continue to work while the filter input is active. |
| 8 | The broken `MouseReleaseMsg` path in `fileselect.Update()` (lines 119–124) is removed and replaced with correct hit-test-gated logic. |

---

## Implementation Steps

Each step must leave `go build ./...` and `go test ./...` passing before the next begins.

### Step 1 — Fix the fileselect mouse-release bug

In `internal/ui/views/fileselect/fileselect.go`:

Remove the `case tea.MouseReleaseMsg, tea.KeyPressMsg:` block at lines 118–130 entirely. Replace
it with two independent type-switch arms:

```go
case tea.KeyPressMsg:
    if item, ok := m.list.SelectedItem().(fileItem); ok {
        if kp, isKey := msg.(tea.KeyPressMsg); isKey && kp.String() == "enter" {
            emit = func() tea.Msg { return msgs.FileSelected{Path: item.path} }
        }
    }
```

The `MouseReleaseMsg` arm is intentionally absent here; it will be added in Step 5 with correct
hit-test logic. This step leaves mouse releases inert but removes the unconditional emission.

---

### Step 2 — Add `SetBounds` to `servicelist`

In `internal/ui/components/servicelist/servicelist.go`:

1. Add `x, y int` fields to `Model`.
2. Rename `SetSize(w, h int) Model` → `SetBounds(x, y, w, h int) Model`. Update the body to also
   store `m.x = x; m.y = y`.
3. Update the call site in `internal/ui/flows/dashboard/project/states/dashboard.go`:

   In `SetSize(w, h int)`, replace:
   ```go
   d.serviceList = d.serviceList.SetSize(leftContentW, innerH)
   ```
   with:
   ```go
   // svcX=1: left border column; svcY=1: top border row
   d.serviceList = d.serviceList.SetBounds(1, 1, leftContentW, innerH)
   ```

Build must pass after this step.

---

### Step 3 — Add `SetBounds` and `WindowSizeMsg` handling to `fileselect`

In `internal/ui/views/fileselect/fileselect.go`:

1. Add `x, y, w, h int` fields to `Model` (always `0, 0, width, height` for the fullscreen case).
2. Update `New(files []string, width, height int) Model` to initialise `w: width, h: height`.
3. Add `SetBounds(x, y, w, h int) Model` method.
4. In `Update()`, add a `tea.WindowSizeMsg` arm that sets `m.w = msg.Width; m.h = msg.Height`
   (x and y remain 0; fileselect is always fullscreen).

`selecting.go` requires no changes — `fileselect.Update()` handles `WindowSizeMsg` internally.

Build must pass after this step.

---

### Step 4 — Hover delegate for `servicelist`

In `internal/ui/components/servicelist/servicelist.go`:

1. Add `hoverIndex int` field to `Model` (initialise to `-1` in `New()`).
2. Add `delegate hoverDelegate` field to `Model`; initialise in `New()` from the existing
   `DefaultDelegate`.

Define the delegate type:

```go
type hoverDelegate struct {
    list.DefaultDelegate
    hoverIndex int
}

func (d hoverDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
    if index == d.hoverIndex {
        dd := d.DefaultDelegate
        bg := lipgloss.AdaptiveColor{Light: "#E8E8E8", Dark: "#2a2a2a"}
        dd.Styles.NormalTitle = dd.Styles.NormalTitle.Background(bg)
        dd.Styles.NormalDesc  = dd.Styles.NormalDesc.Background(bg)
        dd.Render(w, m, index, item)
        return
    }
    d.DefaultDelegate.Render(w, m, index, item)
}
```

Add private helpers:

```go
// headerHeight returns the number of terminal rows occupied by the list header.
// Status bar and help are disabled in servicelist.
func (m Model) headerHeight() int {
    return lipgloss.Height(m.list.Styles.TitleBar.Render(""))
}

// itemsYStart returns the terminal row at which list items begin.
func (m Model) itemsYStart() int {
    return m.y + m.headerHeight()
}

// hitTest maps absolute terminal coordinates to a visible-item index.
// Returns (index, true) when the cursor is over a valid item row; (0, false) otherwise.
func (m Model) hitTest(mouseX, mouseY int) (int, bool) {
    if mouseX < m.x || mouseX >= m.x+m.w {
        return 0, false
    }
    localY := mouseY - m.itemsYStart()
    if localY < 0 {
        return 0, false
    }
    // servicelist: height=1, spacing=0 → 1 row per item
    localIndex := localY
    globalIndex := m.list.Paginator.Page*m.list.Paginator.PerPage + localIndex
    if globalIndex >= len(m.list.VisibleItems()) {
        return 0, false
    }
    return globalIndex, true
}
```

In `Update()`, **after** delegating to `m.list.Update(msg)`, add:

```go
switch msg := msg.(type) {
case tea.MouseMotionMsg:
    idx, ok := m.hitTest(msg.X, msg.Y)
    newHover := -1
    if ok {
        newHover = idx
    }
    if newHover != m.hoverIndex {
        m.hoverIndex = newHover
        m.delegate.hoverIndex = newHover
        m.list.SetDelegate(m.delegate)
    }
case tea.MouseReleaseMsg:
    if msg.Button == tea.MouseLeft {
        if idx, ok := m.hitTest(msg.X, msg.Y); ok {
            m.list.Select(idx)
            // The existing lastSelected comparison handles ServiceSelected emission.
        }
    }
}
```

Build must pass after this step.

---

### Step 5 — Hover delegate for `fileselect`

In `internal/ui/views/fileselect/fileselect.go`:

Same pattern as Step 4, with two differences:
- Default delegate has `ShowDescription=true`, height=2, spacing=1 → **3 rows per item**.
- fileselect shows a title bar and a status bar → both contribute to `headerHeight`.

Define the delegate type:

```go
type hoverDelegate struct {
    list.DefaultDelegate
    hoverIndex int
}

func (d hoverDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
    if index == d.hoverIndex {
        dd := d.DefaultDelegate
        bg := lipgloss.AdaptiveColor{Light: "#E8E8E8", Dark: "#2a2a2a"}
        dd.Styles.NormalTitle = dd.Styles.NormalTitle.Background(bg)
        dd.Styles.NormalDesc  = dd.Styles.NormalDesc.Background(bg)
        dd.Render(w, m, index, item)
        return
    }
    d.DefaultDelegate.Render(w, m, index, item)
}
```

Add private helpers:

```go
func (m Model) headerHeight() int {
    h := lipgloss.Height(m.list.Styles.TitleBar.Render(""))
    if m.list.ShowStatusBar() {
        h += lipgloss.Height(m.list.Styles.StatusBar.Render(""))
    }
    return h
}

func (m Model) itemsYStart() int {
    return m.y + m.headerHeight()
}

func (m Model) hitTest(mouseX, mouseY int) (int, bool) {
    if mouseX < m.x || mouseX >= m.x+m.w {
        return 0, false
    }
    localY := mouseY - m.itemsYStart()
    if localY < 0 {
        return 0, false
    }
    // fileselect: height=2, spacing=1 → 3 rows per item
    localIndex := localY / 3
    globalIndex := m.list.Paginator.Page*m.list.Paginator.PerPage + localIndex
    if globalIndex >= len(m.list.VisibleItems()) {
        return 0, false
    }
    return globalIndex, true
}
```

In `Update()`, replace the entire existing `switch msg.(type)` block (lines 118–130) with:

```go
switch msg := msg.(type) {
case tea.WindowSizeMsg:
    m.w = msg.Width
    m.h = msg.Height

case tea.MouseMotionMsg:
    idx, ok := m.hitTest(msg.X, msg.Y)
    newHover := -1
    if ok {
        newHover = idx
    }
    if newHover != m.hoverIndex {
        m.hoverIndex = newHover
        m.delegate.hoverIndex = newHover
        m.list.SetDelegate(m.delegate)
    }

case tea.MouseReleaseMsg:
    if msg.Button == tea.MouseLeft {
        if idx, ok := m.hitTest(msg.X, msg.Y); ok {
            if item, ok := m.list.VisibleItems()[idx].(fileItem); ok {
                emit = func() tea.Msg { return msgs.FileSelected{Path: item.path} }
            }
        }
    }

case tea.KeyPressMsg:
    if item, ok := m.list.SelectedItem().(fileItem); ok {
        if msg.String() == "enter" {
            emit = func() tea.Msg { return msgs.FileSelected{Path: item.path} }
        }
    }
}
```

Build must pass after this step.

---

### Step 6 — Smoke-test manually

Run `go build ./...` and `go test ./...`. Then run the app and verify:

- Mousing over the Project Selector shows a background tint on the hovered file row.
- Clicking a file immediately proceeds to parsing.
- On the Dashboard, mousing over the service list shows a background tint.
- Clicking a service moves the selection cursor to that service.
- Moving the mouse to the right pane clears the service list hover.
- Keyboard navigation is unaffected.
- Filter mode (`/`): hover and click still work on filtered results.

---

## Out of Scope

- Full visuals pass (colour palette, borders, typography) — planned separately.
- Hover/click for the right pane (log view) — not yet implemented.
- Double-click semantics.
- Tests for UI components (no existing UI tests; adding them is a separate decision).

---

## Post-Implementation

When all implementation steps are complete and the build passes:

1. Move this file to `docs/plans/archive/`
