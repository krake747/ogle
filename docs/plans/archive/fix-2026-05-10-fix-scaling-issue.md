# fix: fix scaling issue

Status: **Ready for implementation.** All design decisions resolved. No open questions.

---

## Context

`ogle` is a Bubbletea v2 TUI for monitoring Docker Compose projects. Two startup views — `fileselect` (the Project Selector) and `watching` (the Watching state) — both implement horizontal and vertical centering using `width` and `height` fields. Both fields are initialised to `0` at construction time.

Bubbletea v2 always calls `p.render(model)` before any message reaches `Update`, including the initial `WindowSizeMsg` it sends asynchronously at startup. This means the first render always uses the `0` fallback, which collapses to the hardcoded `minWidth=80` / `minHeight=24` constants. The views appear in the top-left corner at minimum size regardless of actual terminal dimensions. Subsequent resizes deliver `WindowSizeMsg` and correct the layout — which is why centering works after a resize but not on first paint.

---

## Decision Log

All decisions were reached through a design interview. Do not re-open them without a specific technical reason.

| # | Decision |
|---|---|
| 1 | Query terminal size once in `dashboard.New()` using `github.com/charmbracelet/x/term.GetSize(os.Stdout.Fd())`. Fall back to `0, 0` on error — Bubbletea will correct via `WindowSizeMsg` on first resize. |
| 2 | Pass `width, height int` via constructors down through `startup.New()` → state constructors → `fileHandler` → `fileselect.New()` and `watching.New()`. No `withSize` wrapper, no shared layout struct. |
| 3 | Both `fileselect.New()` and `watching.New()` / `watching.NewDisconnected()` accept `width, height int` and initialise fields directly. All view models with `width`/`height` fields follow this pattern. |
| 4 | `startup.Model` stores `width` and `height`, updates them on `WindowSizeMsg`, and passes current values to `states.NewWatchingWithError` in the `WatcherError` branch — the only place that constructs a new state outside `fileHandler`. |
| 5 | `Watching.Update` intercepts `WindowSizeMsg` to update `w.handler.width`/`w.handler.height` before forwarding to `w.model.Update(msg)`, keeping `fileHandler` dimensions current for any future `Selecting` state it constructs. |
| 6 | `Scanning.Update` does not handle `WindowSizeMsg` — the scan is sub-millisecond and `fileHandler` is already initialised with real dimensions from construction. |
| 7 | Subsequent resizes continue to be handled by `WindowSizeMsg` in each model's `Update` — no change to that path. |

---

## Implementation Steps

Each step must leave the build passing before the next begins.

1. **`internal/ui/views/fileselect/fileselect.go`** — Add `width, height int` to `New()`. Initialise `m.width = width`, `m.height = height`.

2. **`internal/ui/views/watching/watching.go`** — Add `width, height int` to `New()` and `NewDisconnected()`. Initialise `m.width = width`, `m.height = height` in both.

3. **`internal/ui/flows/startup/states/handler.go`** — Add `width, height int` to `fileHandler`. Pass them to `fileselect.New()` in `handle()`, to `watching.New()` in `newWatching()`, and to the `visibleState()` fallback.

4. **`internal/ui/flows/startup/states/watching.go`** — Add `width, height int` to `NewWatching()` and `NewWatchingWithError()`. Pass to `watching.New()` and set on `fileHandler`. Add `WindowSizeMsg` interception in `Watching.Update` to update `w.handler.width`/`w.handler.height` before forwarding to `w.model.Update(msg)`.

5. **`internal/ui/flows/startup/states/scanning.go`** — Add `width, height int` to `NewScanning()`. Set on `fileHandler`. No `WindowSizeMsg` handling.

6. **`internal/ui/flows/startup/startup.go`** — Add `width, height int` to `New()` and to `startup.Model`. Pass dimensions to all three initial-state constructors. Intercept `WindowSizeMsg` in `Update` to keep fields current and pass to `states.NewWatchingWithError` in the `WatcherError` branch.

7. **`internal/ui/flows/dashboard/dashboard.go`** — Import `github.com/charmbracelet/x/term`. In `New()`, call `term.GetSize(os.Stdout.Fd())`, falling back to `0, 0` on error. Pass `width, height` to `startup.New()`.

---

## Out of Scope

- Any changes to the centering math or `maxContentWidth` constants in either view.
- The `watching.NewDisconnected` call sites — `NewDisconnected` has no external callers yet; only its signature and field initialisation are updated.
- The dashboard and project views — they do not have `width`/`height` fields and are unaffected.
- `tea.RequestWindowSize` — already removed from `dashboard.Init()`.

---

## Post-Implementation

When all implementation steps are complete and the build passes:

1. Move this file to `docs/plans/archive/`
