# feat: status bar theming

Status: **Ready for implementation.** All design decisions resolved. No open questions.

---

## Context

The status bar is currently rendered inline in `app.go`'s `View()` (lines 354–371). It uses
`theme.ActionError` for errors but applies no styling to info-level messages (plain terminal
default text). It has no background colour. It is not a component — its state
(`statusMsg`, `statusMsgEnd`, `statusMsgError`) and render logic live directly on the root
`Model`, unlike `topbar` and `helpbar` which are standalone components.

Two `Theme` slots are missing:
- `StatusInfo color.Color` — foreground for non-error status messages
- `StatusBarBackground color.Color` — background tint behind the entire status line

---

## Decision Log

All decisions were reached through a design interview. Do not re-open them without a specific
technical reason.

| # | Decision |
|---|---|
| 1 | Add `StatusInfo color.Color` to `Theme` — info messages get an independent slot rather than reusing `Text` |
| 2 | Add `StatusBarBackground color.Color` to `Theme` — themed background behind the status line |
| 3 | Extract a standalone `statusbar` component mirroring the `topbar`/`helpbar` pattern |
| 4 | Status bar uses `th.ActionError` (error) and `th.StatusInfo` (info); background applied via `th.StatusBarBackground` |

---

## Implementation Steps

Each step must leave the build passing before the next begins.

### Step 1 — Add theme slots

In `internal/ui/theme/theme.go`:
- Add `StatusInfo color.Color` and `StatusBarBackground color.Color` to the `Theme` struct
- Add `StatusInfoColor string` and `StatusBarBackgroundColor string` to `userThemeFile`
- Handle both in `applyOverrides()`

In `internal/ui/theme/builtin.go` (`Default()`):
- `StatusInfo`: `defaultWhite` (`#e4e4e4`) — same as `Text`; neutral but explicitly assigned
- `StatusBarBackground`: `defaultBlack` (`#0B0B0B`) — matches `ServiceListBackground`

In `internal/ui/theme/catppuccino_mocha.go` (`CatppuccinoMocha()`):
- `StatusInfo`: `mochaSubtext1` (`#bac2de`) — slightly dimmer than body text; fitting for transient feedback
- `StatusBarBackground`: `mochaMantle` (`#181825`) — one step darker than base; distinct but subtle

### Step 2 — Create the statusbar component

New file: `internal/ui/components/statusbar/statusbar.go`

```go
// Package statusbar provides a transient one-line status message bar.
// It renders an info or error message for a fixed duration, then clears itself.
package statusbar
```

Model fields:
- `msg string`
- `msgEnd time.Time`
- `isError bool`
- `th *theme.Theme`
- `width int`

Constructor: `New(th *theme.Theme) Model`

`Update()` handles:
- `tea.WindowSizeMsg` → update `width`
- `msgs.ThemeChanged` → update `th`
- `msgs.DisplayError{Err}` → set `msg`, `msgEnd`, `isError = true`; return tick cmd
- `msgs.DisplayStatus{Msg}` → set `msg`, `msgEnd`, `isError = false`; return tick cmd
- `msgs.ClearStatusMsg` → clear `msg` if `time.Now().After(msgEnd)`

`Height() int`: returns `1` when `msg != ""`, otherwise `0`. Used by `app.go` to adjust `avail`.

`View() tea.View`:
- When `msg == ""`, return `tea.NewView("")`
- Build a `lipgloss.Style` with `Background(th.StatusBarBackground)` and width padding to `m.width`
- Apply `Foreground(th.ActionError)` (error) or `Foreground(th.StatusInfo)` (info)
- Return `tea.NewView(style.Render(m.msg))`

### Step 3 — Wire into app.go

- Add `statusbar statusbar.Model` field to `app.Model`
- Remove `statusMsg string`, `statusMsgEnd time.Time`, `statusMsgError bool` fields
- Remove `handleStatusMsg()` method
- `New()`: initialise with `statusbar: statusbar.New(th)`
- `Update()`: forward all messages to `m.statusbar` (the component self-handles `DisplayError`,
  `DisplayStatus`, `ClearStatusMsg`, `ThemeChanged`, `WindowSizeMsg`)
- Remove the three `case msgs.DisplayError`, `case msgs.DisplayStatus`, `case msgs.ClearStatusMsg`
  handlers from app `Update()` — they are now handled by the component
- `View()`:
  - `avail -= m.statusbar.Height()`
  - Slot `m.statusbar.View().Content` between `bodyPadded` and `m.helpbar.View().Content`

### Step 4 — Verify

Run `go build ./...` and `go vet ./...`. Confirm the status bar renders correctly for both
info and error messages, and that the background and foreground colours reflect the active theme.

---

## Out of Scope

- Log Filter feature (unrelated)
- Any changes to `helpbar` or `topbar`
- New built-in themes
- Any changes to the status message duration or dismissal logic

---

## Post-Implementation

When all implementation steps are complete and the build passes:

1. Move this file to `docs/plans/archive/`
