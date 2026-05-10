# feat: startup ux improvements

Status: **Ready for implementation.** All design decisions resolved. No open questions.

---

## Context

The startup flow drives the user from app launch to a loaded compose project through four states: `Scanning`, `Watching`, `Selecting`, and `Parsing`. The two visible views — `watching` and `fileselect` — render plain, unconstrained strings with no terminal dimension awareness, no fullscreen mode, and no layout structure. The goal is to improve the visual quality of the startup flow without changing its behaviour.

The dashboard (`project/states/idle.go`) is a stub and is explicitly out of scope.

---

## Decision Log

All decisions were reached through a design interview. Do not re-open them without a specific technical reason.

| # | Decision |
|---|---|
| 1 | Alt-screen (`tea.WithAltScreen()`) is enabled from program start — applied at the `tea.NewProgram` call site, not toggled mid-session. |
| 2 | Minimum supported terminal size is 80×24. If the actual terminal is smaller, render as if it were 80×24 (clamp, no warning, no blocking). |
| 3 | Both `watching.Model` and `fileselect.Model` store `width` and `height`, updated via `tea.WindowSizeMsg`. Clamping (`max(w, 80)` / `max(h, 24)`) is applied at the point of use in `View()`. |
| 4 | `tea.WindowSizeMsg` propagates automatically through the existing fallthrough paths in `dashboard.Update` and `startup.Update` — no explicit handling needed at those layers. |
| 5 | Content block is fluid up to 120 columns wide, then centred horizontally beyond that. |
| 6 | Layout: main content is centred vertically in the space above the footer; keybinding hints are pinned to the last row of the terminal. |
| 7 | Long lines (e.g. deep directory paths in `watching`) wrap within the content block width rather than truncate. |
| 8 | The "ogle" header remains plain text — no borders, ASCII art, or external styling library. No lipgloss dependency is introduced. |
| 9 | The `fileselect` list remains a plain cursor list (`> ` prefix) — no borders or dividers. |
| 10 | The `Parsing` state surfaces activity by calling `SetParsing(bool)` on the prior view (`watching.Model` or `fileselect.Model`), which renders a "Parsing..." notice inline. This is consistent with the existing `SetNotice` / `SetError` mutator pattern. |

---

## Implementation Steps

Each step must leave the build passing before the next begins.

### Step 1 — Enable alt-screen

Find the `tea.NewProgram` call site and add `tea.WithAltScreen()`.

### Step 2 — Add `WindowSizeMsg` handling to `watching.Model`

- Add `width int` and `height int` fields to `watching.Model`.
- Handle `tea.WindowSizeMsg` in `watching.Model.Update`, storing clamped dimensions (`max(w, 80)` / `max(h, 24)`).
- No layout changes yet — `View()` is unchanged in this step.

### Step 3 — Add `WindowSizeMsg` handling to `fileselect.Model`

Same as Step 2 for `fileselect.Model`.

### Step 4 — Add `SetParsing(bool)` to both views

- Add a `parsing bool` field to `watching.Model` and `fileselect.Model`.
- Implement `SetParsing(bool) Model` on both, consistent with `SetNotice` / `SetError`.
- Update `View()` on both to render a `"Parsing..."` notice line when `parsing` is true.
- Update `Parsing` state (`states/parsing.go`) to call `SetParsing(true)` on the display model when entering, and `SetParsing(false)` on exit/error.

### Step 5 — Implement layout in `watching.View()`

Using the stored `width` and `height`:
- Clamp effective width to `min(width, 120)`.
- Build the content block: `"ogle"` header + blank line + body (wrapped to content width).
- Compute vertical offset: `(height - contentLines - 1) / 2` to centre content above the pinned footer.
- Render leading blank lines, then content, then fill remaining lines, then the keybinding footer on the last row.
- Wrap long body lines (e.g. directory paths) at content width using a simple word/character wrap.

### Step 6 — Implement layout in `fileselect.View()`

Same layout approach as Step 5:
- Clamp effective width to `min(width, 120)`.
- Content block: `"ogle"` header + blank line + prompt + blank line + file list.
- Centre content vertically above pinned footer.
- Keybinding footer pinned to last row.

### Step 7 — Verify build and manual smoke test

- `go build ./...` must pass.
- Run the program in a terminal at 80×24, 120×40, and a size smaller than 80×24 to confirm clamping behaviour.

---

## Out of Scope

- The project dashboard (`internal/ui/flows/dashboard/project/`).
- Any new dependencies (no lipgloss, no bubbles).
- Changes to startup flow behaviour or state transitions.
- Spinner or animated loading indicators.
- Terminal size warnings or blocking resize prompts.
- The `Scanning` state (it renders a blank screen and is not user-visible).

---

## Post-Implementation

When all implementation steps are complete and the build passes:

1. Move this file to `docs/plans/archive/`
