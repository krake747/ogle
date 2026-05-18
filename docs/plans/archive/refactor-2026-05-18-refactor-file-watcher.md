# refactor: refactor-file-watcher

Status: **Ready for implementation.** All design decisions resolved. No open questions.

---

## Context

The watcher is currently created inside `startup.New`. It lives on `startup.Model` and is orphaned when the startup→dashboard transition discards `m.current` — the goroutine leaks, and the watcher cannot continue monitoring for file changes while the dashboard is active.

This plan moves watcher ownership to `app.Model`, keeps it alive across flow transitions, and lets the dashboard react to compose-file changes (Live Reload).

---

## Decision Log

| # | Decision |
|---|---|
| 1 | `app.Model` holds `startup` and `dashboard` as separate `tea.Model` fields (no `current`) — both alive side-by-side. A `phase` enum controls which is active for Update/View. |
| 2 | A third phase, `phaseWatching`, is added for when the project file disappears at runtime. A stub `watching.Model` is used; its return path is deferred. |
| 3 | Message routing: app handles its own cases explicitly (`KeyPressMsg`, `SettingsApplied`, `ProfilesDumped`, `ProjectLoaded`, `FileAvailabilityChanged`, `FileRemoved`). Unhandled messages fall through to a default dispatch by phase to the active model. |
| 4 | `app.Update` intercepts `FileAvailabilityChanged`, dispatches to the active model by phase, then always re-arms via `m.watcher.Next()`. |
| 5 | `FileAvailabilityChanged` is not forwarded to idle models — only the active phase receives it. |
| 6 | Dashboard owns a `parser.Parser`. On `FileAvailabilityChanged` with the project file present, it re-parses and replaces itself (returns a new dashboard model with its own `Init` baked into the command). |
| 7 | On `FileAvailabilityChanged` with the project file missing, dashboard emits `msgs.FileRemoved{File: string}`. App catches it, creates the watching stub, sets `phaseWatching`. |
| 8 | The watching stub ignores all messages (stub for later implementation). The return path (file reappears → back to dashboard) is deferred. |
| 9 | `startup.Model` loses its `watcher.Watcher` field — app owns the watcher, startup never uses it. `startup.New` takes no watcher param, returns `Model` (not `(Model, error)`). |
| 10 | `watcher.New` returns `nil, err` on failure instead of `nullWatcher`. Sentinel `watcher.ErrCreateWatcher` added. `nullWatcher` / `NewNull` are deleted. |
| 11 | `app.New` fails hard on watcher error — returns `(Model, error)`. Caller exits with descriptive error. |
| 12 | `app.Close()` closes the watcher. Wired into `cmd/root.go` after `program.Run()`. |

---

## Implementation Steps

### Step 1 — Add sentinel `watcher.ErrCreateWatcher`, delete `nullWatcher`

**`internal/services/watcher/service.go`:**

- Add `var ErrCreateWatcher = errors.New("create watcher")`
- `New` returns `nil, fmt.Errorf("%w: ...", ErrCreateWatcher, ...)` on both error paths instead of `NewNull()`
- Remove `import "sync"` if it becomes unused

**`internal/services/watcher/null.go`:** Delete entire file.

**`internal/services/watcher/service_test.go`:**

- Delete `TestNewNull` (lines 22–70)
- `"non-existent directory returns error and valid Watcher"`: change `require.NotNil(t, w)` to `require.Nil(t, w)`, remove `require.NoError(t, w.Close())`

### Step 2 — `app.Model`: add phase + watcher + model fields

**`internal/app/app.go`:**

```go
type phase int

const (
    phaseStartup   phase = iota
    phaseDashboard
    phaseWatching
)

type Model struct {
    ctx       context.Context
    cfg       config.Config
    configDir string
    dir       string
    log       *slog.Logger
    theme     *theme.Theme
    zm        *zone.Manager
    watcher   watcher.Watcher
    startup   tea.Model
    dashboard tea.Model
    watching  tea.Model           // for phaseWatching
    phase     phase
    width     int
    height    int
}
```

### Step 3 — `app.New`: create watcher, remove startup error

`app.New` returns `(Model, error)`. Creates watcher, returns `Model{}, fmt.Errorf(...)` on failure. Startup constructed via `startup.New(ctx, log, dir, width, height)` — no error.

### Step 4 — `cmd/root.go`: handle app init error

Update `app.New` call to check error, propagate via `fmt.Errorf("app init: %w", err)`.

### Step 5 — `startup.New`: remove watcher, remove error return

Signature becomes `func New(ctx context.Context, log *slog.Logger, dir string, w, h int) Model`. No watcher field on `startup.Model`. `Init()` returns nil (no Snapshot call).

### Step 6 — `app.Init`: snapshot + startup init

```go
return tea.Batch(m.watcher.Snapshot(), m.startup.Init())
```

### Step 7 — `app.Update`: dispatch by phase

Full rewrite per decision log #3–4, #7. Add `msgs.FileRemoved` handling, `FileAvailabilityChanged` interception with `Next()` re-arm.

### Step 8 — `app.View`: dispatch by phase

```go
switch m.phase { ... }
```

### Step 9 — `app.Close`: close watcher

```go
func (m Model) Close() error { return m.watcher.Close() }
```

Wire into `cmd/root.go` after `program.Run()`.

### Step 10 — Create watching stub

New file `internal/ui/components/watching/watching.go`. Minimal `tea.Model` with `File string` field, no-op Update.

### Step 11 — Add `msgs.FileRemoved`

```go
type FileRemoved struct { File string }
```

### Step 12 — Dashboard handles `FileAvailabilityChanged`

Add `parser.Parser` and `*slog.Logger` to `dashboard.Model`. On `FileAvailabilityChanged`:
- File not in list → emit `FileRemoved`
- File present → re-parse, return new dashboard with its own Init

### Step 13 — Startup forwards `FileAvailabilityChanged`

No code change needed — startup's default Update path forwards unrecognised messages to `m.fileSelect.Update(msg)` as before.

---

## Target Structure

```
internal/
├── app/
│   └── app.go                 (restructured — phase, watcher, two models)
├── msgs/
│   └── msgs.go                (+ FileRemoved)
├── services/
│   └── watcher/
│       ├── service.go          (+ ErrCreateWatcher, nil on error)
│       ├── service_test.go     (- TestNewNull, updated assertion)
│       ├── null.go             (DELETED)
└── ui/
    ├── components/
    │   └── watching/
    │       └── watching.go     (NEW — stub)
    └── flows/
        ├── dashboard/
        │   └── dashboard.go    (+ parser, FACh handler)
        └── startup/
            └── startup.go      (no watcher, no error return)
```

---

## Out of Scope

- Return path from watching to dashboard (deferred to stub implementation)
- Debouncing rapid file change events (watcher's drain-select already handles this)
- Any user-facing feature changes — this is a pure refactor of watcher lifecycle

---

## Post-Implementation

When all implementation steps are complete and the build passes:

1. Move this file to `docs/plans/archive/`
