# feat: log streaming

Status: **Ready for implementation.** All design decisions resolved. No open questions.

---

## Context

ogle is a terminal UI for monitoring Docker Compose projects. The Service Inspector
(right pane of the Dashboard) currently shows a placeholder in the log area
("Connecting to Docker…", "Docker unavailable — retrying in Xs…"). Log streaming
is the core observability feature of the product — the Service Inspector is defined
as the place where the Selected Service's live log tail is displayed.

State Polling is also unimplemented, but log streaming can be built independently
by deriving the container name directly from the Compose File.

---

## Decision Log

All decisions were reached through a design interview. Do not re-open them without
a specific technical reason.

| # | Decision |
|---|---|
| 1 | Container identified by `container_name` from Compose File if set; otherwise `<project>-<service>-1` (Compose v2 naming). v1 (`_`) gap documented in a code comment. No dependency on State Polling. |
| 2 | Raw `net/http` over the Unix socket (`/var/run/docker.sock`) — no Docker SDK. Frame demultiplexer written inline (~30 lines). Consistent with the existing `docker` service. |
| 3 | `LogStreamer` service at `internal/services/docker/logs/`: goroutine + output channel + cancel func. Exposes `Start(ctx, containerName)`, `Next() tea.Cmd`, `Close()`. Mirrors the watcher pattern. Null Object (`NullLogStreamer`) follows ADR-0006. |
| 4 | Log Buffer owned by `*Dashboard` (pointer receiver). Cap: 1000 lines. Oldest lines discarded when exceeded. Never stored inside `inspector.Model` (value type — copying a large buffer is expensive). |
| 5 | Stream starts on `DaemonConnected` for the currently selected service. |
| 6 | On `ServiceSelected`: stop old streamer, clear buffer, start new. On `ProjectLoaded` (Live Reload): same — reset to first service in the reloaded Project. |
| 7 | On `LogStreamError` (mid-stream read failure): stop streamer, freeze buffer, show unavailable placeholder. Restart when `DaemonConnected` fires again. |
| 8 | On `LogStreamContainerNotFound` (404 at stream start): show `"No container — service not started"` and retry every 5 seconds. |
| 9 | Lines wrap at pane width. `ansi.StringWidth` (already imported) used for ANSI-safe width measurement. |
| 10 | ANSI escape codes passed through — the terminal renders them natively. |
| 11 | stderr lines rendered in theme error colour. stdout lines unstyled. |
| 12 | Timestamps off. |
| 13 | Initial tail: 1000 lines (`?tail=1000`). |
| 14 | PgUp/PgDn scroll half a pane. Mouse wheel scrolls one line. Arrow keys are not used for log scroll (they drive the service list). Scroll is available in both split and fullscreen modes — no focus switching required. |
| 15 | Scrolling up pauses auto-tail. Reaching the bottom automatically resumes. Paused indicator: `── paused · PgDn to resume ──` pinned to the last row of the log area (dimmed theme colour). |

---

## Target Structure

```
internal/
  msgs/
    msgs.go                  ← add LogLine, LogStreamError, LogStreamContainerNotFound
  services/
    docker/
      service.go             ← Connect cmd (unchanged)
      logs/
        service.go           ← LogStreamer: Start, Next, Close, frame demux, ContainerName
        null.go              ← NullLogStreamer (Null Object, ADR-0006)
  ui/
    theme/
      theme.go               ← add LogStderr lipgloss.Color field
      builtin.go             ← populate LogStderr in Default() and CatppuccinoMocha()
    components/
      inspector/
        inspector.go         ← add SetLogView(); renderLogArea() renders passed lines
    flows/
      dashboard/
        project/
          states/
            dashboard.go     ← logStreamer, logBuffer, scroll offset, all lifecycle
            logbuffer.go     ← unexported logBuffer + logLine types
```

---

## Implementation Steps

Each step must leave the build passing before the next begins.

### Step 1 — Add new message types to `internal/msgs/msgs.go`

```go
// LogLine carries one demultiplexed log frame from the Docker logs API.
type LogLine struct {
    Text     string
    IsStderr bool
}

// LogStreamError is emitted when the LogStreamer goroutine hits a read error.
type LogStreamError struct{ Err error }

// LogStreamContainerNotFound is emitted when the logs endpoint returns 404.
type LogStreamContainerNotFound struct{}
```

No logic changes. Build passes.

---

### Step 2 — Add `LogStderr` colour to theme

In `internal/ui/theme/theme.go`, add `LogStderr lipgloss.Color` to the `Theme` struct.

In `internal/ui/theme/builtin.go`, populate:
- `Default()`: a muted red/orange readable on light terminals.
- `CatppuccinoMocha()`: `#f38ba8` (Catppuccino Mocha red).

Build passes.

---

### Step 3 — Implement `internal/services/docker/logs/`

The transport setup (Unix socket dialer, `http.Transport`, `http.Client`) is identical to
`internal/services/docker/service.go` — copy that pattern verbatim. The logs endpoint is
`http://localhost/containers/{name}/logs?follow=true&stdout=1&stderr=1&tail=1000` issued
over the same socket at `/var/run/docker.sock`.

**`service.go`**:

```go
type LogStreamer struct {
    cancel context.CancelFunc
    ch     chan tea.Msg   // untyped, matching the watcher pattern; buffered cap 64
    wg     sync.WaitGroup
}

func New() *LogStreamer {
    return &LogStreamer{ch: make(chan tea.Msg, 64)}
}
```

- `func ContainerName(project, service, containerNameOverride string) string` — returns
  override if non-empty; otherwise `project + "-" + service + "-1"`. Add a comment:
  `// Compose v1 used underscores (project_service_1); v2 uses dashes. Only v2 is supported.`

- `func (s *LogStreamer) Start(appCtx context.Context, containerName string)` — creates a
  child context: `ctx, cancel := context.WithCancel(appCtx); s.cancel = cancel`. Builds
  the request, sends it. On 404 writes `msgs.LogStreamContainerNotFound{}` to `s.ch` and
  returns (no goroutine). On other non-200 writes `msgs.LogStreamError{Err: ...}` and
  returns. On 200, starts the reader goroutine:

  ```go
  s.wg.Add(1)
  go func() {
      defer s.wg.Done()
      defer resp.Body.Close()
      for {
          var header [8]byte
          if _, err := io.ReadFull(resp.Body, header[:]); err != nil {
              select {
              case <-ctx.Done():
                  // normal shutdown — do not emit an error
              default:
                  s.ch <- msgs.LogStreamError{Err: err}
              }
              return
          }
          size := binary.BigEndian.Uint32(header[4:])
          payload := make([]byte, size)
          if _, err := io.ReadFull(resp.Body, payload); err != nil {
              select {
              case <-ctx.Done():
              default:
                  s.ch <- msgs.LogStreamError{Err: err}
              }
              return
          }
          select {
          case s.ch <- msgs.LogLine{Text: string(payload), IsStderr: header[0] == 2}:
          case <-ctx.Done():
              return
          }
      }
  }()
  ```

- `func (s *LogStreamer) Next() tea.Cmd` — returns a cmd that blocks on `s.ch`:
  ```go
  return func() tea.Msg { return <-s.ch }
  ```
  **The caller must call `Next()` again after each received message** — there is no
  automatic re-subscription (same contract as `watcher.Next()`).

- `func (s *LogStreamer) Close()`:
  ```go
  func (s *LogStreamer) Close() {
      if s.cancel != nil {
          s.cancel()
      }
      s.wg.Wait()
      // Drain any buffered messages so a blocking Next() cmd does not get stuck.
      for {
          select {
          case <-s.ch:
          default:
              return
          }
      }
  }
  ```

**`null.go`**:

```go
type NullLogStreamer struct{}

func (NullLogStreamer) Start(_ context.Context, _ string) {}
func (NullLogStreamer) Next() tea.Cmd {
    // Block forever — no events will arrive.
    return func() tea.Msg { return (<-make(chan tea.Msg)) }
}
func (NullLogStreamer) Close() {}
```

Build passes.

---

### Step 4 — Implement `logBuffer` in `states/logbuffer.go`

```go
type logLine struct {
    text     string
    isStderr bool
}

type logBuffer struct {
    lines []logLine
    cap   int
}

func newLogBuffer(cap int) logBuffer
func (b *logBuffer) Append(line logLine)  // drops oldest when cap exceeded
func (b *logBuffer) Clear()
func (b *logBuffer) Lines() []logLine
```

Build passes.

---

### Step 5 — Wire `*Dashboard`: add fields and handle all log stream messages

Add to `Dashboard` struct:

```go
theme         *theme.Theme              // needed by computeDisplayLines for stderr colour
logStreamer   *logs.LogStreamer
logBuffer     logBuffer
logScrollRows int                       // display rows from bottom; 0 = tailing
logPaused     bool
logState      inspector.LogAreaState
```

`theme` is already available as the `th` parameter in `NewDashboard()` — store it alongside
the other fields. `NewDashboard()` also initialises:
- `logBuffer = newLogBuffer(1000)`
- `logStreamer = logs.New()`
- `logState = inspector.LogAreaConnecting`

**Add `logStreamRetryMsg` to `states/msgs.go`** (alongside `gracePeriodExpiredMsg` and
`retryTickMsg`):

```go
type logStreamRetryMsg struct{}
```

**Add `startLogStream` helper on `*Dashboard`:**

```go
func (d *Dashboard) startLogStream(svc domain.ServiceDef) tea.Cmd {
    name := logs.ContainerName(d.project.Name, svc.Name, svc.ContainerName)
    d.logStreamer.Start(d.ctx, name)
    d.logState = inspector.LogAreaStreaming
    return d.logStreamer.Next()
}
```

Handle in `Update()`:

| Message | Action |
|---|---|
| `msgs.DaemonConnected` | existing handler + `return d, tea.Batch(existingCmds, d.startLogStream(d.selectedService))` |
| `msgs.LogLine` | `d.logBuffer.Append(logLine{text: msg.Text, isStderr: msg.IsStderr})`; if not paused `d.logScrollRows = 0`; re-subscribe `d.logStreamer.Next()` |
| `msgs.LogStreamError` | `d.logStreamer.Close()`; `d.logState = inspector.LogAreaUnavailable` (buffer frozen — do not clear) |
| `msgs.LogStreamContainerNotFound` | `d.logState = inspector.LogAreaNotFound`; return 5s ticker: `tea.Tick(5*time.Second, func(_ time.Time) tea.Msg { return logStreamRetryMsg{} })` |
| `logStreamRetryMsg` | `return d, d.startLogStream(d.selectedService)` |
| `msgs.ServiceSelected` | existing handler + `d.logStreamer.Close(); d.logBuffer.Clear()`; if `d.connectState == inspector.ConnectStateConnected` return `d.startLogStream(msg.Service)` |
| `msgs.ProjectLoaded` | existing handler + `d.logStreamer.Close(); d.logBuffer.Clear()`; if connected return `d.startLogStream(first)` |
| `msgs.DaemonUnavailable` | existing handler + `d.logStreamer.Close()` |

**Important:** `tea.MouseWheelMsg` events must be handled in Dashboard's `Update` switch
**before** the `inspector.Update(msg)` passthrough at the bottom. The inspector's
`adjustMouseY` does not handle `tea.MouseWheelMsg`, and `inspector.Update` ignores wheel
events entirely when `showLabels` is false — they are silently dropped. Add an explicit
`case tea.MouseWheelMsg:` (see Step 7).

After every relevant mutation, push state to the inspector:

```go
d.inspector = d.inspector.SetLogView(d.computeDisplayLines(), d.logPaused, d.logState)
```

`computeDisplayLines()` is a stub returning `nil` at this step. Build passes; inspector
still shows existing placeholders.

---

### Step 6 — Update `inspector.Model` to render log lines

**Export `HeaderLines` and define `LogAreaState` in `inspector` package.**

`headerLines` is currently an unexported constant (`= 2`) in `header.go`. Rename it to
the exported `HeaderLines` so that `computeDisplayLines()` in package `states` can
reference it as `inspector.HeaderLines` without hardcoding the magic number.

Define the log area state type in the `inspector` package (not in `states`) so that both
packages can use it without creating a circular import — `states` imports `inspector`,
never the reverse.

```go
// LogAreaState describes what the log area should render.
type LogAreaState int

const (
    LogAreaConnecting  LogAreaState = iota // daemon ping in-flight or grace period active
    LogAreaStreaming                        // stream attached; lines available
    LogAreaUnavailable                     // daemon unreachable; buffer frozen
    LogAreaNotFound                        // container does not exist yet
)
```

Add to `inspector.Model`:

```go
logLines  []string
logPaused bool
logState  LogAreaState
```

Add `SetLogView(lines []string, paused bool, state LogAreaState) Model`.

Update `renderLogArea()`:

- `LogAreaConnecting` → `"Connecting to Docker…"`
- `LogAreaUnavailable` → existing retry countdown string
- `LogAreaNotFound` → `"No container — service not started"`
- `LogAreaStreaming` → join `m.logLines` with `"\n"`; if `m.logPaused` append the
  styled paused indicator (`"── paused · PgDn to resume ──"`) rendered with a dim
  style, as the final line.

Build passes. Logs render (empty until Step 7 fills `computeDisplayLines`).

---

### Step 7 — Implement `computeDisplayLines()` and scroll

Implement `computeDisplayLines() []string` on `*Dashboard`:

1. Obtain log view bounds: `lb := d.layout.LogViewBounds()`.
2. Iterate `d.logBuffer.Lines()`; for each `logLine`, word-wrap at `lb.w` using
   `ansi.StringWidth` for width measurement. Apply `d.theme.LogStderr` colour (via
   `lipgloss.NewStyle().Foreground(d.theme.LogStderr).Render(row)`) to all display
   rows produced from stderr lines.
3. Compute available rows: when tailing `availRows = lb.h - inspector.HeaderLines`;
   when paused `availRows = lb.h - inspector.HeaderLines - 1` (last row reserved for
   paused indicator). `inspector.HeaderLines` is the exported constant added in Step 6.
4. Clamp `d.logScrollRows` to `[0, max(0, totalRows-availRows)]`.
5. Slice: `displayRows[totalRows-availRows-d.logScrollRows : totalRows-d.logScrollRows]`.
6. Return the slice.

Handle scroll input in `Update()`:

- `tea.KeyPressMsg` PgUp → `d.logScrollRows += halfPane; d.logPaused = true`
- `tea.KeyPressMsg` PgDn → `d.logScrollRows -= halfPane; clamp; auto-resume at 0`
- `tea.MouseWheelMsg` — add an explicit `case tea.MouseWheelMsg:` to the `Update`
  switch **before** the `inspector.Update(msg)` passthrough. Hit-test using
  `d.layout.LogViewBounds()`: only act if `msg.Y >= lb.y && msg.Y < lb.y+lb.h`.
  `tea.MouseButton` values: `tea.MouseWheelUp` scrolls up (pause + increment),
  `tea.MouseWheelDown` scrolls down (decrement + auto-resume at 0). Return early
  after handling so the event is not also forwarded to the inspector.
- New `msgs.LogLine` when not paused → `d.logScrollRows = 0`

Build passes. Feature complete.

---

## Out of Scope

- **State Polling** — `ServiceRuntimeData` remains nil; not part of this plan.
- **Timestamps** — off; addable via Settings overlay in a future plan.
- **Log Filter** — defined in CONTEXT.md; separate feature.
- **Tab focus switching** in split mode — deferred (`dashboard.go:93`); not needed
  since scroll uses PgUp/PgDn and mouse wheel.
- **Per-service buffer persistence** — buffer clears on every service change.
- **Compose v1 container naming** (`project_service_1`) — not handled; gap documented
  in code comment.
- **Service Actions** and **Orphan Toggle** — out of scope.

---

## Post-Implementation

When all implementation steps are complete and the build passes:

1. Move this file to `docs/plans/archive/`
