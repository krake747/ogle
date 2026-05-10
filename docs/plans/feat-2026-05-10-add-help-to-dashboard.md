# feat: add help to dashboard

Status: **Ready for implementation.** All design decisions resolved. No open questions.

---

## Context

The dashboard project flow currently has a single `Idle` state that renders basic project info (name, file, service count) and handles `q` to quit via a raw string comparison. There is no help bar. The `State` interface has no sizing contract, so states have no way to know terminal dimensions.

This plan adds a bubbles `help.Model` to the `Idle` state, wires up proper keybinding definitions using `charm.land/bubbles/v2/key`, and extends the `State` interface with `SetSize(w, h int)` so `project.Model` can propagate `tea.WindowSizeMsg` down to active states.

---

## Decision Log

All decisions were reached through a design interview. Do not re-open them without a specific technical reason.

| # | Decision |
|---|---|
| 1 | Only active keybindings are defined (`q` quit). Disabled placeholder bindings for unimplemented features are not added. |
| 2 | `help.Model` is owned by each state individually, not by `project.Model`. States are self-contained; the help bar is part of that contract. |
| 3 | `SetSize(w, h int)` is added to the `State` interface. `project.Model` intercepts `tea.WindowSizeMsg` and calls `current.SetSize`. |
| 4 | Both `w` and `h` are passed in `SetSize` — height will be needed by future states (log viewport, service list) and avoids a future interface break. |
| 5 | Key handling migrates from raw string comparison to `key.Binding` + `key.Matches`. The same binding definition drives both input handling and help text. |
| 6 | `Idle.View()` composes the full layout using stored `w`/`h`. Help bar rendered on the last row. No shared layout helper extracted yet — premature before a second state exists. |

---

## Implementation Steps

Each step must leave the build passing before the next begins.

1. **Add `charm.land/bubbles/v2` dependency** — run `go get charm.land/bubbles/v2` and tidy.

2. **Extend `State` interface** — add `SetSize(w, h int)` to `internal/ui/flows/dashboard/project/states/state.go`.

3. **Wire `SetSize` in `project.Model`** — handle `tea.WindowSizeMsg` in `project.go`; call `m.current.SetSize(msg.Width, msg.Height)` and return an updated model.

4. **Rewrite `Idle` state** — in `internal/ui/flows/dashboard/project/states/idle.go`:
   - Define a `keyMap` struct with a `Quit key.Binding` (`key.WithKeys("q")`, `key.WithHelp("q", "quit")`).
   - Add `w`, `h int` and `help help.Model` fields to `Idle`.
   - Implement `SetSize(w, h int)` — store dimensions, call `s.help.SetWidth(w)`.
   - Replace raw `msg.String() == "q"` with `key.Matches(msg, s.keys.Quit)`.
   - Rewrite `View()` to render project info and position help bar on the last row using `h`.

5. **Verify** — `go build ./...` passes.

---

## Target Structure

No new files or packages. All changes are within existing files.

```
internal/ui/flows/dashboard/project/
├── project.go                  ← handles tea.WindowSizeMsg; calls SetSize
└── states/
    ├── state.go                ← SetSize(w, h int) added to interface
    └── idle.go                 ← help.Model, keyMap, SetSize, rewritten View
```

---

## Out of Scope

- Keybindings for any feature not yet implemented (service navigation, actions, filter, orphan toggle, log scrolling).
- Full/short help toggle (`?` key and `help.ShowAll`).
- Styling customisation beyond bubbles defaults.
- Propagating size to startup flow states.

---

## Post-Implementation

When all implementation steps are complete and the build passes:

1. Move this file to `docs/plans/archive/`
