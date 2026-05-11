# refactor: mouse support refactor

Status: **Ready for implementation.** All design decisions resolved. No open questions.

---

## Context

Mouse support was added to `fileselect` and `servicelist` in a previous iteration. Both modules now
contain near-identical hover-rendering and hit-test logic (~50 lines each):

- A `hoverDelegate` struct embedding `list.DefaultDelegate` with a verbatim `Render` method.
- Three helper methods: `headerHeight()`, `itemsYStart()`, `hitTest()`.
- A `hoverIndex int` field on `Model`, kept in sync with `delegate.hoverIndex` by two manual
  assignments on every hover change — a hidden invariant with no enforcement.

The only real differences between the two implementations are the row stride (3 for `fileselect`,
1 for `servicelist`) and the number of header rows (2 vs 1) — both of which are parameters, not
structural differences.

Injection was considered and rejected: it would push delegate configuration knowledge to callers
(`dashboard.go`, the startup flow), which is a worse trade than the duplication it solves.

The solution is **composition via a shared package**. Both modules import `internal/ui/hoverlist/`
and use its concrete types internally. No caller changes required.

---

## Decision Log

All decisions were reached through a design interview. Do not re-open them without a specific
technical reason.

| # | Decision |
|---|---|
| 1 | Extract to `internal/ui/hoverlist/` — a peer of `components/`, `views/`, and `flows/`, not a subdirectory of `components/`. `hoverlist` has no Bubble Tea model (no `Init`/`Update`/`View`); placing it under `components/` would misrepresent its role as infrastructure rather than a sub-model. |
| 2 | `hoverlist` exports exactly two things: a `Delegate` type and a `HitTest` pure function. No interfaces, no injection. |
| 3 | `Delegate.SetHover(index int) bool` uses a pointer receiver and returns `true` when the index changed. This eliminates the `Model.hoverIndex` field from both `fileselect` and `servicelist` — `delegate.hoverIndex` becomes the single source of truth. `list.SetDelegate` is called only when `SetHover` returns `true`. |
| 4 | `HitTest` is a pure function taking all values as explicit parameters. It has no dependency on `list.Model` or any bubbles type and is trivially table-driven testable. |
| 5 | `headerRows` is a named constant in each consuming module, not a runtime measurement. `fileselect` always shows a title bar and a status bar (`const headerRows = 2`); `servicelist` shows a title bar only (`const headerRows = 1`). This eliminates the `headerHeight()`, `itemsYStart()` methods and the `lipgloss.Height` runtime measurement from both files. |
| 6 | Constructor injection rejected. `New()` in each module continues to own delegate configuration (`ShowDescription`, `SetSpacing`, etc.). The caller (Dashboard state, startup flow) requires no changes. |

---

## Implementation Steps

Each step must leave `go build ./...` and `go test ./...` passing before the next begins.

### Step 1 — Create `internal/ui/hoverlist/hoverlist.go`

Create the new package. It must export:

**`Delegate`** — wraps `list.DefaultDelegate` with hover rendering:

```go
type Delegate struct {
    list.DefaultDelegate
    hoverIndex int
}

func NewDelegate(base list.DefaultDelegate) Delegate {
    return Delegate{DefaultDelegate: base, hoverIndex: -1}
}

// SetHover updates the hovered VisibleItems index (-1 = none).
// Reports whether the index changed; callers should call list.SetDelegate
// only when true.
func (d *Delegate) SetHover(index int) bool {
    if d.hoverIndex == index {
        return false
    }
    d.hoverIndex = index
    return true
}

func (d Delegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
    if index == d.hoverIndex {
        dd := d.DefaultDelegate
        bg := lipgloss.Color("237")
        dd.Styles.NormalTitle = dd.Styles.NormalTitle.Background(bg)
        dd.Styles.NormalDesc = dd.Styles.NormalDesc.Background(bg)
        dd.Render(w, m, index, item)
        return
    }
    d.DefaultDelegate.Render(w, m, index, item)
}
```

**`HitTest`** — pure coordinate geometry:

```go
// HitTest maps absolute terminal coordinates to an index into list.VisibleItems().
// originX, originY: top-left of the list content area in terminal space.
// width:            content width in columns.
// headerRows:       rows above the first item (title bar, status bar, etc.).
// rowStride:        rows per item (delegate.Height() + delegate.Spacing()).
// pageOffset:       Paginator.Page * Paginator.PerPage.
// itemCount:        len(list.VisibleItems()).
// Returns (index, true) on a hit; (0, false) otherwise.
func HitTest(mouseX, mouseY, originX, originY, width, headerRows, rowStride, pageOffset, itemCount int) (int, bool) {
    if mouseX < originX || mouseX >= originX+width {
        return 0, false
    }
    localY := mouseY - (originY + headerRows)
    if localY < 0 {
        return 0, false
    }
    localIndex := localY / rowStride
    globalIndex := pageOffset + localIndex
    if globalIndex >= itemCount {
        return 0, false
    }
    return globalIndex, true
}
```

Build must pass after this step.

---

### Step 2 — Refactor `servicelist`

In `internal/ui/components/servicelist/servicelist.go`:

1. Add `const headerRows = 1` — title bar only; status bar and help are disabled.
2. Replace the `hoverDelegate` type definition and its `Render` method with `hoverlist.Delegate`.
3. In `Model`, replace the `delegate hoverDelegate` and `hoverIndex int` fields with a single
   `delegate hoverlist.Delegate` field. Remove `hoverIndex int`.
4. Update `New()`: replace `hoverDelegate{DefaultDelegate: base, hoverIndex: -1}` with
   `hoverlist.NewDelegate(base)`.
5. Delete `headerHeight()`, `itemsYStart()`, and the existing `hitTest()` method.
6. Add a thin `hitTest` wrapper:

```go
func (m Model) hitTest(mouseX, mouseY int) (int, bool) {
    return hoverlist.HitTest(
        mouseX, mouseY, m.x, m.y,
        m.list.Width(), headerRows, 1,
        m.list.Paginator.Page*m.list.Paginator.PerPage,
        len(m.list.VisibleItems()),
    )
}
```

7. In `Update()`, replace the two-field hover sync:

```go
// Before:
if newHover != m.hoverIndex {
    m.hoverIndex = newHover
    m.delegate.hoverIndex = newHover
    m.list.SetDelegate(m.delegate)
}

// After:
if m.delegate.SetHover(newHover) {
    m.list.SetDelegate(m.delegate)
}
```

8. Remove the `"charm.land/lipgloss/v2"` import if it is now unused (it will be — `lipgloss` is
   only referenced by the deleted `hoverDelegate` and `headerHeight`). Keep `"io"` only if still
   needed; it will also be unused once `hoverDelegate.Render` is deleted.

Build must pass after this step.

---

### Step 3 — Refactor `fileselect`

In `internal/ui/views/fileselect/fileselect.go`:

1. Add `const headerRows = 2` — title bar (1 row) + status bar (1 row); both are always shown.
2. Replace the `hoverDelegate` type and its `Render` method with `hoverlist.Delegate`.
3. In `Model`, replace `delegate hoverDelegate` and `hoverIndex int` with `delegate hoverlist.Delegate`.
   Remove `hoverIndex int`.
4. Update `New()`: replace `hoverDelegate{DefaultDelegate: list.NewDefaultDelegate(), hoverIndex: -1}`
   with `hoverlist.NewDelegate(list.NewDefaultDelegate())`.
5. Delete `headerHeight()`, `itemsYStart()`, and the existing `hitTest()` method.
6. Add a thin `hitTest` wrapper:

```go
func (m Model) hitTest(mouseX, mouseY int) (int, bool) {
    return hoverlist.HitTest(
        mouseX, mouseY, m.x, m.y,
        m.w, headerRows, 3,
        m.list.Paginator.Page*m.list.Paginator.PerPage,
        len(m.list.VisibleItems()),
    )
}
```

7. In `Update()`, replace the two-field hover sync with `m.delegate.SetHover(newHover)` as in Step 2.
8. Remove `"charm.land/lipgloss/v2"` and `"io"` imports if now unused.

Build must pass after this step.

---

### Step 4 — Build and test

Run `go build ./...` and `go test ./...`. Both must pass clean.

Verify that `charm.land/lipgloss/v2` and `"io"` are not orphaned anywhere (they will now be
imported only by `hoverlist`).

---

## Target Structure

```
internal/ui/
  hoverlist/
    hoverlist.go       ← new: Delegate type + HitTest pure function
  components/
    servicelist/
      servicelist.go   ← refactored: uses hoverlist, ~50 lines shorter
  views/
    fileselect/
      fileselect.go    ← refactored: uses hoverlist, ~50 lines shorter
```

---

## Out of Scope

- Tests for `hoverlist.HitTest` — the pure function is now testable; writing those tests is a
  separate decision tracked under the UI model test conventions (ADR-0011).
- Mouse support for the right pane (log view) — planned separately.
- Visual changes to the hover tint colour.
- Any changes to callers (`dashboard.go`, startup flow states).

---

## Post-Implementation

When all implementation steps are complete and the build passes:

1. Move this file to `docs/plans/archive/`
