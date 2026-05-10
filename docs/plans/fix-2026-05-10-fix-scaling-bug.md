# fix: fix scaling bug

Status: **Ready for implementation.** All design decisions resolved. No open questions.

---

## Context

`ogle` is a Bubbletea v2 TUI for monitoring Docker Compose projects. The startup flow includes a `fileselect` view shown when multiple compose files are found. This view computes a horizontally- and vertically-centered layout using `m.width` and `m.height`.

The program sets `v.AltScreen = true` on the `View` return value (`dashboard.go:111`) but does **not** pass `tea.WithAltScreen()` to `tea.NewProgram`. In Bubbletea v2, only the latter causes an initial `WindowSizeMsg` to be emitted at startup. Without it, `m.width` and `m.height` remain `0` on first render, causing the view to fall back to `minWidth=80` / `minHeight=24` and render in the top-left corner regardless of actual terminal size. A `WindowSizeMsg` is only received on manual resize, at which point centering works correctly.

---

## Decision Log

All decisions were reached through a design interview. Do not re-open them without a specific technical reason.

| # | Decision |
|---|---|
| 1 | Add `tea.WithAltScreen()` to `tea.NewProgram` in `cmd/root.go`. This is the canonical Bubbletea fix for missing initial `WindowSizeMsg`. |
| 2 | Do not remove `v.AltScreen = true` from `dashboard.View()` — its role (switching the renderer to alt screen buffer) is separate and correct. |
| 3 | No changes to `fileselect.go` layout logic — the centering math is correct once `width`/`height` are populated. |

---

## Implementation Steps

1. In `cmd/root.go`, add `tea.WithAltScreen()` to the `tea.NewProgram(...)` call.
2. Run `go build ./...` — must pass before proceeding.
3. Manually verify: launch `ogle` in a terminal wider than 120 columns and confirm the file select view renders centered on first paint, not just after resize.

---

## Out of Scope

- Any changes to `fileselect.go` layout logic or `maxContentWidth`.
- Fixing the blank-line padding omission (`l == ""` path skips `pad`) — cosmetically harmless and not the reported bug.
- Any other views or flows beyond the startup `fileselect`.

---

## Post-Implementation

When all implementation steps are complete and the build passes:

1. Move this file to `docs/plans/archive/`
