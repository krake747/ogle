# feat: mouse click support

Status: **Ready for implementation.** All design decisions resolved. No open questions.

---

## Context

ogle is a Bubble Tea v2 TUI for Docker Compose project monitoring. It has three phases: startup (file picker), dashboard (service list + inspector + logs), and watching (waiting for compose file).

Currently, all navigation and actions are keyboard-only. Mouse mode is enabled (`MouseModeCellMotion`) and BubbleZone is used for hit-testing, but no component handles `MouseClickMsg`. The servicelist already wraps items in BubbleZone zones via `hoverlist.Delegate` (used only for rendering — the hover index is never updated). The fileselect uses a vanilla `list.DefaultDelegate` with no zone support at all.

The plan adds mouse click interaction for list selection in the startup file picker and for service actions (toggle start/stop, rebuild) in the dashboard service list. The help bar stays read-only.

---

## Decision Log

All decisions were reached through a design interview. Do not re-open them without a specific technical reason.

| # | Decision |
|---|---|
| 1 | Fileselect uses `hoverlist.Delegate` for zone registration and hover (consistent with servicelist) |
| 2 | Single click on a file in startup immediately confirms the selection (emits `FileSelected`) |
| 3 | Mouse events for servicelist are handled inside `servicelist.Model` (not at dashboard level) |
| 4 | Double-click detection: immediately move selection on first click, track timestamp. Second click on same item within 350ms triggers start/stop toggle. No delay for single-click feedback |
| 5 | Shift+click moves selection to clicked item and triggers rebuild |
| 6 | Mouse motion updates hover index in both lists (visual highlight via existing `hoverlist.Delegate`) |
| 7 | Help bar is not made clickable (Bubbles `help.Model` is a rendering black box; custom implementation not worth the complexity) |
| 8 | When the settings overlay is showing, mouse events are swallowed rather than forwarded to the service list |

---

## Implementation Steps

Each step leaves the build passing before the next begins.

### Step 1: Wire zone manager and theme through the startup flow

The fileselect needs access to `*zone.Manager` and `*theme.Theme` to register zones and style hover highlights.

Files to edit:
- **`internal/ui/components/fileselect/fileselect.go`**: Add `zm *zone.Manager` and `th *theme.Theme` to the `Model` struct. Update `New()` signature to accept them. No behavioral change yet — just store the references.
- **`internal/ui/flows/startup/startup.go`**: Add `zm *zone.Manager` and `th *theme.Theme` to the `Model` struct. Update `New()` signature to accept and pass them to `fileselect.New()`.
- **`internal/app/app.go`**: Pass `m.zm` and `m.theme` to `startup.New()`.

### Step 2: Add mouse support to fileselect

- **`internal/ui/components/fileselect/fileselect.go`**:
  - Replace `list.NewDefaultDelegate()` with `hoverlist.NewDelegate(base, th, zm)`
  - Add `hitTest(mouseX, mouseY int) (int, bool)` method that iterates `m.list.VisibleItems()` and checks `zm.Get("item-N").InBounds()`
  - In `Update`, add case for `tea.MouseMotionMsg`: hit-test, call `m.delegate.SetHover(idx)` (or `-1` if no hit)
  - In `Update`, add case for `tea.MouseClickMsg` on left button: hit-test, call `m.list.Select(idx)`, emit `msgs.FileSelected` for the clicked item
  - Ensure the `msgs.FileSelected` emission matches the existing Enter-key path

### Step 3: Add mouse support to servicelist

- **`internal/ui/components/servicelist/servicelist.go`**:
  - Store `zm *zone.Manager` on the `Model` struct (already passed to `New()` but wasn't stored)
  - Add `lastClickTime time.Time` and `lastClickIdx int` fields for double-click tracking
  - Add `hitTest(mouseX, mouseY int) (int, bool)` method (same pattern as fileselect)
  - In `Update`, add case for `tea.MouseMotionMsg`: hit-test, call `m.delegate.SetHover(idx)` (or `-1` if no hit)
  - In `Update`, add case for `tea.MouseClickMsg` on left button:
    1. Hit-test to find clicked item
    2. Call `m.list.Select(idx)` to move selection
    3. If selection changed, emit `msgs.ServiceSelected`
    4. If `msg.Mod.Contains(tea.ModShift)` → emit `msgs.ServiceRebuild` (same payload as keyboard 'b')
    5. Else if same item and within 350ms of `lastClickTime` → emit `msgs.ServiceStart` or `msgs.ServiceStop` depending on current state (same logic as keyboard 's')
    6. Else → record `lastClickTime = now`, `lastClickIdx = idx`
  - Handle the `msgs.ServiceSelected` emission for mouse-driven selection changes (the current code at the bottom of `Update` already handles this for keyboard; ensure it also fires when `m.list.Select(idx)` is called from the mouse handler)

### Step 4: Block mouse events behind the settings overlay

- **`internal/ui/flows/dashboard/dashboard.go`**:
  - In `Update`, when `m.showingSettings` is true, skip forwarding `tea.MouseClickMsg`, `tea.MouseMotionMsg`, `tea.MouseReleaseMsg`, and `tea.MouseWheelMsg` to sub-components. Mouse events should still reach `m.settings2` so the overlay could potentially handle them in future, but for now they are simply dropped.

---

## Out of Scope

- Mouse wheel scrolling for the service list (wheel events flow to the log pane via the service panel; changing this would require coordinate-based routing)
- Help bar click-to-trigger (decided against)
- Click handling for the daemon status header or service inspector/log pane
- Right-click or middle-click support
- Any changes to keyboard shortcuts
- Updating the help bar documentation (explicitly excluded by the user)

---

## Post-Implementation

When all implementation steps are complete and the build passes:

1. Move this file to `docs/plans/archive/`
