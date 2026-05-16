# refactor: log refactor

Status: **Ready for implementation.** All design decisions resolved. No open questions.

---

## Context

Each Docker log line arrives as a `msgs.LogLine` tea.Msg through the Bubble Tea
message loop. When logs stream at 100+/sec, the update loop is saturated calling
`viewport.SetContentLines` (two O(n) scans of all buffered lines per message).
Keyboard/mouse events for scrolling the service list sit queued behind hundreds
of LogLine messages.

Currently: Streamer writes `LogLine` messages to a `chan tea.Msg` (cap 64).
`servicehost` forwards them to `logpane.Update`, which calls `SetContentLines`
on every single line. The pipeline is synchronous — every line is a message
through the Bubble Tea loop.

---

## Decision Log

All decisions were reached through a design interview. Do not re-open them
without a specific technical reason.

| # | Question | Decision |
|---|---|---|
| 1 | Transport mechanism | `chan string` with bounded back-pressure |
| 2 | Channel capacity | 5000 (`lineBufferCap` const, promote later) |
| 3 | Streamer API shape | Split: `Lines() <-chan string` for lines, keep `Next() tea.Cmd` for errors |
| 4 | Flush tick ownership | Per-logpane `tea.Tick(50ms)` — self-contained, no shared msg types |
| 5 | Flush interval | 50ms (20 fps) |
| 6 | `IsStderr` field | Dropped — defined but never consumed anywhere |
| 7 | `msgs.LogLine` | Removed from `internal/msgs/msgs.go` |
| 8 | Error display | Unchanged — `LogStreamError`/`LogStreamContainerNotFound` silently re-subscribe via `Next()` |
| 9 | Channel lifecycle | Streamer closes `lineCh` in `Close()`. Logpane detects non-ok read, sets `m.lineCh = nil`, stops re-scheduling tick. |
| 10 | Trailing `\n` | Streamer trims via `strings.TrimRight` before write |
| 11 | Viewport update | `SetContentLines` once per flush tick (was: once per line) |

---

## Implementation Steps

1. **`internal/services/docker/logs/service.go`** — Add `lineBufferCap = 5000`
   const. Add `lineCh chan string` field. Create in `New()` with
   `make(chan string, lineBufferCap)`. Add `Lines() <-chan string` method.
   In `readFrames`, write trimmed `string(payload)` to `lineCh` instead of
   `msgs.LogLine` to `msgCh`. In `Close()`, close `lineCh` after draining
   error messages.

2. **`internal/services/docker/logs/streamer.go`** — Add `Lines() <-chan string`
   to the `Streamer` interface.

3. **`internal/services/docker/logs/null.go`** — Add `Lines()` returning
   `(<-chan string)(nil)`.

4. **`internal/msgs/msgs.go`** — Remove `LogLine` struct and any dead
   references.

5. **`internal/ui/components/logpane/logpane.go`** — Accept `<-chan string`
   in `New()`. Remove `msgs.LogLine` case from Update. Define private
   `flushTickMsg` type. Add `flushTick()` cmd in `Init()`. Add
   `flushTickMsg` handler: non-blocking drain of `lineCh`, append to
   `m.lines`, cap enforcement, single `SetContentLines`, auto-scroll.
   On closed channel: `m.lineCh = nil`, stop re-scheduling tick.

6. **`internal/ui/components/servicehost/servicehost.go`** — Pass
   `m.streamer.Lines()` to `logpane.New()`. Remove `case msgs.LogLine:`
   handler and re-subscription logic. Clean up imports.

7. **`internal/ui/components/servicehost/servicehost_test.go`** — Remove
   or update tests that send `msgs.LogLine` through servicehost.Update.

8. **Build and verify** — `go build ./...` and `go test ./...` pass.

---

## Out of Scope

- Keyboard scrolling for the log pane (viewport KeyMap is intentionally empty
  — separate focus-management concern).
- Stderr highlighting (`IsStderr`) — add `chan logEntry{Text, IsStderr}` when a
  consumer exists.
- Changes to log cap (1000), flush tick interval config, or any other existing
  configuration.
- `LogStreamError` / `LogStreamContainerNotFound` handling.

---

## Post-Implementation

When all implementation steps are complete and the build passes:

1. Move this file to `docs/plans/archive/`
