# feat: horizontal scroll

Status: **Ready for implementation.** All design decisions resolved. No open questions.

---

## Context

The log pane renders Docker container output using a `charm.land/bubbles/v2/viewport`. The viewport has built-in support for horizontal scrolling (`ScrollLeft`/`ScrollRight`, `SetXOffset`, `Left`/`Right` keybindings) but ogle currently disables it entirely by clearing the KeyMap (`vp.KeyMap = viewport.KeyMap{}` at `internal/ui/components/logpane/logpane.go:31`).

When log wrap is off (the default), long lines exceed the pane width and are silently truncated. Users must toggle wrap on (`w`) to read long lines, but wrapping changes the entire layout. Horizontal scrolling is the standard terminal pattern for viewing wide content without wrapping.

The viewport already makes `SetXOffset` a no-op when `SoftWrap` is true, so horizontal scrolling naturally only works in unwrapped mode — no extra guarding needed.

---

## Decision Log

All decisions were reached through a design interview. Do not re-open them without a specific technical reason.

| # | Decision |
|---|---|
| 1 | Follow the same message-passing pattern as `w` → `ToggleLogWrap`: dashboard catches the key, emits a typed message, logpane handles it by calling viewport scroll methods. |
| 2 | Use a step of 8 cells per keypress, matching common TUI convention. |
| 3 | Keys are `←`/`h` (left) and `→`/`l` (right), matching the viewport's default bindings. These don't conflict with any existing list or dashboard keys. |
| 4 | No changes to `servicepanel.go` or `servicehost.go` — `KeyPressMsg` already reaches all hosts through the existing forwarding loop at `servicepanel.go:80-84`. |

---

## Implementation Steps

### 1. Add `LogScrollH` message type

**File:** `internal/msgs/msgs.go`

Add a new struct carrying a signed scroll amount (positive = right, negative = left):

```go
type LogScrollH struct {
    Amount int
}
```

### 2. Handle scroll in log pane

**File:** `internal/ui/components/logpane/logpane.go`

Add a `case msgs.LogScrollH:` branch to the Update type switch. Call `m.viewport.ScrollRight(msg.Amount)` — a negative Amount scrolls left, positive scrolls right. The viewport's `SetXOffset` (called by `ScrollRight`) is already a no-op when `SoftWrap` is true, so this naturally does nothing in wrapped mode.

### 3. Add dashboard keybindings

**File:** `internal/ui/flows/dashboard/keymap.go`

Add two package-level bindings:

```go
keyScrollLeft  = key.NewBinding(key.WithKeys("left", "h"), key.WithHelp("←/h", "scroll left"))
keyScrollRight = key.NewBinding(key.WithKeys("right", "l"), key.WithHelp("→/l", "scroll right"))
```

No change to `extraBindings` — these go into the `actions` slice, not appended in `ShortHelp`.

### 4. Wire keys to messages

**File:** `internal/ui/flows/dashboard/dashboard.go`

- In `Init()`: add `keyScrollLeft` and `keyScrollRight` to the `actions` slice in the `appKeymap`.
- In `Update()`: add two `case` branches under `tea.KeyPressMsg`, before the default forwarding:

```go
if key.Matches(msg, keyScrollLeft) && !m.showingSettings {
    return m, func() tea.Msg { return msgs.LogScrollH{Amount: -8} }
}
if key.Matches(msg, keyScrollRight) && !m.showingSettings {
    return m, func() tea.Msg { return msgs.LogScrollH{Amount: +8} }
}
```

### 5. Build and verify

```bash
go build ./...
go vet ./...
```

---

## Out of Scope

- Mouse wheel horizontal scrolling (Shift+scroll is supported by the viewport but not wired through `servicepanel` routing — future enhancement).
- Configurable step size (hardcoded to 8 for now).
- Horizontal scroll indicator (e.g. a progress bar showing x-offset — cosmetic, not functional).
- Changes to existing wrap toggle behaviour.

---

## Post-Implementation

When all implementation steps are complete and the build passes:

1. Move this file to `docs/plans/archive/`
