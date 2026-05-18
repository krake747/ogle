# feat: watching-states

Status: **Ready for implementation.** All design decisions resolved. No open questions.

---

## Context

The root state machine in `internal/app/app.go` has three phases: `phaseStartup`, `phaseDashboard`, and `phaseWatching`. The `phaseWatching` → `phaseDashboard` return path is a no-op stub in `internal/ui/components/watching/watching.go` — when the project file disappears at runtime, the app enters the watching phase and stays there forever, even if the file reappears.

The watcher (`internal/services/watcher/service.go`) delivers `msgs.FileAvailabilityChanged` on every filesystem event. The watching component receives these messages but discards them (stub comment: "the return path is not yet implemented").

This plan implements the return path: the watching component reacts to `FileAvailabilityChanged`, attempts to parse the original file when it reappears, and emits `msgs.ProjectLoaded` on success to trigger the transition back to dashboard.

---

## Decision Log

All decisions were reached through a design interview. Do not re-open them without a specific technical reason.

| # | Decision |
|---|---|
| 1 | The watching component tracks the original project file (`Model.File`), which is already stored. |
| 2 | Reuse `msgs.ProjectLoaded{Project *domain.Project}` for the watching → dashboard transition. `domain.Project.File` carries the file path. No new message needed. |
| 3 | On parse failure (original file reappears but YAML is invalid): show the error inline in the watching view and auto-retry on the next `FileAvailabilityChanged`. No user action required. |
| 4 | On parse success: the component emits a `ProjectLoaded` command synchronously from `Update`. App's existing `case msgs.ProjectLoaded:` handler creates a fresh dashboard and transitions phase. |
| 5 | The component parses the original file only. Alternative compose files appearing while watching are ignored — the user is recovering a known project, not browsing for a new one. |
| 6 | The component's state machine has two states: `stateIdle` (waiting) and `stateParseError` (file present but unparseable). No alternatives list, no cursor, no key bindings. |
| 7 | The component owns its parser dependency, created internally in `New()` — consistent with the startup flow pattern (`startup.New` creates `parser.New(ctx, logger)`). |
| 8 | The component renders its own output (no delegation to `views/watching`). The view package stays for the startup flow's cold-mode watching. |

---

## Implementation Steps

### 1. `internal/ui/components/watching/watching.go` — state type and Model fields

- Add imports: `"context"`, `"log/slog"`, `"slices"`
- Add `state` type and two consts:
  ```go
  type state int
  const (
      stateIdle state = iota
      stateParseError
  )
  ```
- Add fields to `Model`:
  ```go
  ctx     context.Context
  log     *slog.Logger
  st      state
  parseErr error
  ```
- Remove the orphaned `parser parser.Parser` field (it's never wired — `New()` will create the parser).

### 2. `internal/ui/components/watching/watching.go` — New() signature

Change from:
```go
func New(file string, w, h int) Model
```
to:
```go
func New(ctx context.Context, logger *slog.Logger, file string, w, h int) Model
```

Body creates `parser.Service`:
```go
return Model{
    parser: parser.New(ctx, logger),
    File:   file,
    st:     stateIdle,
    w:      w,
    h:      h,
    ctx:    ctx,
    log:    logger,
}
```

### 3. `internal/ui/components/watching/watching.go` — Update: FileAvailabilityChanged handler

Replace the empty `case msgs.FileAvailabilityChanged:` branch:

```go
case msgs.FileAvailabilityChanged:
    if slices.Contains(msg.Files, m.File) {
        p, err := m.parser.Parse(m.File)
        if err != nil {
            m.st = stateParseError
            m.parseErr = err
            return m, nil
        }
        return m, func() tea.Msg {
            return msgs.ProjectLoaded{Project: p}
        }
    }
    m.st = stateIdle
    m.parseErr = nil
    return m, nil
```

### 4. `internal/ui/components/watching/watching.go` — View

Render different output based on state:

```go
func (m Model) View() tea.View {
    var body string
    switch m.st {
    case stateParseError:
        body = fmt.Sprintf(
            "compose file unavailable — waiting...\n\nParse error: %v\nWaiting for file to change...",
            m.parseErr,
        )
    default:
        body = "compose file unavailable — waiting..."
    }
    return tea.NewView(body)
}
```

### 5. `internal/app/app.go` — update New() call

Line 186:
```go
// Before:
m.watching = watching.New(msg.File, m.width, m.height)
// After:
m.watching = watching.New(m.ctx, m.log, msg.File, m.width, m.height)
```

---

## Out of Scope

- Alternative compose file selection during watching (e.g., picking a different file than the original).
- Changes to `views/watching` — the display-only view remains for the startup flow's cold-watching mode.
- Changes to `msgs.go`, `domain/domain.go`, or any other file.
- UI polish beyond the two-state rendering described above.
- `msgs.ProjectLoadFailed` — the watching component handles parse errors inline without emitting messages.

---

## Post-Implementation

When all implementation steps are complete and the build passes:

1. Move this file to `docs/plans/archive/`
