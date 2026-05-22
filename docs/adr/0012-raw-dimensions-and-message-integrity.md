# ADR-0012: Components receive raw terminal dimensions; no parent pre-calculation or message reconstruction

**Status:** Proposed

## Context

The ogle component tree uses Bubble Tea's `tea.Model` convention: every component implements `Init()`, `Update(tea.Msg)`, and `View()`. Terminal dimensions arrive as `tea.WindowSizeMsg{Width, Height}` from the runtime and propagate through the tree. In practice, parent components have been pre-calculating usable dimensions (subtracting chrome height, splitting percentages) and either (a) passing pre-chewed sizes to child constructors, or (b) reconstructing a synthetic `tea.WindowSizeMsg` with modified `Height` before forwarding. Two related smells appear:

1. **Helper methods exposing internals** — `servicehost.ServiceName()` and `statusbar.Height()` expose internal state so parents can route messages and adjust layout.
2. **Message reconstruction in the chain** — `startup.go` builds `tea.WindowSizeMsg{Width: msg.Width, Height: msg.Height - frameHeight}` instead of passing the original message through. `dashboard.go` and `watching.go` each duplicate `frameHeight` and subtract it on receipt.

The result is a shallow seam: every child must know what its parent subtracted, and parents must query children for height or name to compose the frame. The interface of each module is nearly as complex as its implementation.

Options considered:

1. **Status quo** — parents pre-calculate dimensions, pass them to constructors, and reconstruct messages. Rejected: leaks layout policy across seams; duplication of `frameHeight`, `listRatio`, and `listMinTermWidth`; children cannot be tested without reverse-engineering parent arithmetic.
2. **Raw dimensions, internal calculation** — `tea.WindowSizeMsg` propagates unchanged. Every component stores raw `w` and `h` and derives its own content size internally. Parents never query children for height or routing state; `ServiceSelected` messages drive selection, not `ServiceName()`. Accepted.
3. **Shared layout value object** — a `layout.Dashboard` struct centralises split ratios and chrome constants. Rejected: adds a new dependency; the duplication of `listRatio` is small and local; the deeper problem is message reconstruction, not constant duplication.

## Decision

- `tea.WindowSizeMsg` is never reconstructed, wrapped, or modified in the middle of the message chain. It propagates everywhere with its original `Width` and `Height`.
- No component receives pre-calculated dimensions in its constructor. `w, h` parameters are removed from `New()` where they exist solely to seed initial size.
- Each component owns its dimension calculations internally, using only the raw terminal `w` and `h` that arrive via `Update`.
- Parent components do not query children for height or internal state to adjust layout. `statusbar` renders zero-height content when inactive; `app.View()` appends it unconditionally. `servicehost` decides internally whether to consume input and whether to render, driven by `ServiceSelected` messages.

## Consequences

- `frameHeight` lives only in `app.go`; it is removed from `dashboard`, `startup`, and `watching`.
- `listRatio` and `listMinTermWidth` may still be duplicated between `carousel` and `logpane` because both independently derive their allocation from raw `w`. This is accepted as the lesser evil versus parent pre-calculation.
- `servicehost.ServiceName()` and `statusbar.Height()` are removed. The interface of a component is `Init/Update/View` only.
- Tests improve: any model can be exercised with `tea.WindowSizeMsg{Width: 80, Height: 24}` and asserted to store exactly 80×24, with no need to reverse-engineer what a parent subtracted.
- View tests assert on rendered content (empty string vs active message) rather than querying `Height()`.
