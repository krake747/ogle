# refactor: Log Streamer Injection

Status: **Ready for implementation.** All design decisions resolved. No open questions.

---

## Context

`LogPane` in `internal/ui/flows/dashboard/project/states/logpane.go` holds a
`streamer *logs.LogStreamer` field — a concrete pointer type. `LogPane` is
embedded in `Dashboard` as its `logView LogPane` field. This makes the log
streaming path untestable in isolation: any test that constructs a
`Dashboard` must either wire a real Docker-connected streamer or accept that
log-streaming behaviour cannot be exercised at all.

Introducing a `Streamer` interface at the `logs` package level, and injecting
it through `NewDashboard` → `LogPane`, removes the concrete dependency. A
`FakeStreamer` (buffered channel, `Send` method) enables unit tests to drive
log output without Docker.

---

## Decision Log

All decisions were reached through a design interview. Do not re-open them without a specific technical reason.

| # | Decision |
|---|---|
| 1 | Interface is named `Streamer` (not `LogStreamer`) and declared in `internal/services/docker/logs/streamer.go`. |
| 2 | Compile-time interface assertions added for both `*LogStreamer` and `NullLogStreamer` in `streamer.go`. |
| 3 | `FakeStreamer` lives in `internal/services/docker/logs/fake.go`; exported for testing; buffered channel; exposes `Send(msg tea.Msg)` plus the full `Streamer` method set. |
| 4 | `NullLogStreamer` (already in `logs/null.go`) is left as-is — it already satisfies the method set. |
| 5 | `LogPane.streamer` field type changes from `*logs.LogStreamer` to `logs.Streamer` in `logpane.go`. |
| 6 | `NewLogPane(bufCap int)` gains a `streamer logs.Streamer` parameter; callers pass `logs.New()` in production and `logs.NewFakeStreamer()` in tests. |
| 7 | `NewDashboard` gains a `streamer logs.Streamer` parameter; it is forwarded into the `LogPane` struct literal as `streamer: streamer`. |
| 8 | All three callers pass `logs.New()` explicitly: `project.New` (project.go:34), `Settings.Confirm` (settings.go:167), `Settings.Cancel` (settings.go:172). |
| 9 | `Settings` gains an import of the `logs` package as a result of passing `logs.New()`. This is accepted. |
| 10 | The Settings message-drop bug (candidate 4) is deferred and out of scope for this plan. |

---

## Implementation Steps

Each step must leave the build passing before the next begins.

### Step 1 — Declare the `Streamer` interface

Create `internal/services/docker/logs/streamer.go`:

```go
package logs

import (
	"context"

	tea "charm.land/bubbletea/v2"
)

// Streamer is the interface satisfied by any log source. It mirrors the method
// set of *LogStreamer and NullLogStreamer exactly.
type Streamer interface {
	Start(ctx context.Context, containerName string)
	Next() tea.Cmd
	Close()
}

// Compile-time assertions.
var _ Streamer = (*LogStreamer)(nil)
var _ Streamer = NullLogStreamer{}
```

Run `go build ./internal/services/docker/logs/...`. The assertions confirm the
existing types already satisfy the interface.

### Step 2 — Create `FakeStreamer`

Create `internal/services/docker/logs/fake.go`:

```go
package logs

import (
	"context"

	tea "charm.land/bubbletea/v2"
)

// FakeStreamer is a test double for Streamer. Exported for testing.
type FakeStreamer struct {
	ch chan tea.Msg
}

// NewFakeStreamer returns a FakeStreamer with a buffered channel.
func NewFakeStreamer() *FakeStreamer {
	return &FakeStreamer{ch: make(chan tea.Msg, 64)}
}

// Send enqueues a message to be delivered on the next Next() call.
func (f *FakeStreamer) Send(msg tea.Msg) {
	f.ch <- msg
}

// Start is a no-op — FakeStreamer does not connect to Docker.
func (f *FakeStreamer) Start(_ context.Context, _ string) {}

// Next returns a cmd that blocks until Send delivers a message.
func (f *FakeStreamer) Next() tea.Cmd {
	return func() tea.Msg {
		return <-f.ch
	}
}

// Close is a no-op.
func (f *FakeStreamer) Close() {}
```

Run `go build ./internal/services/docker/logs/...`.

### Step 3 — Update `LogPane` to use the interface

In `internal/ui/flows/dashboard/project/states/logpane.go`:

1. Change the `streamer` field from `*logs.LogStreamer` to `logs.Streamer`.
2. Add `streamer logs.Streamer` as the first parameter to `NewLogPane`.
3. Update the `NewLogPane` body to assign `streamer: streamer` (removing the
   internal `logs.New()` call).

```go
type LogPane struct {
	streamer   logs.Streamer
	// ... remaining fields unchanged
}

func NewLogPane(streamer logs.Streamer, bufCap int) *LogPane {
	return &LogPane{
		streamer:   streamer,
		buffer:     newLogBuffer(bufCap),
		scrollRows: 0,
		paused:     false,
		state:      inspector.LogAreaStreaming,
	}
}
```

Run `go build ./...` — this will fail at the `NewLogPane` call site and the
`LogPane` struct literal in `NewDashboard`, which are fixed in the next step.

### Step 4 — Update `NewDashboard` and its callers

**`internal/ui/flows/dashboard/project/states/dashboard.go`:**

1. Add `streamer logs.Streamer` as the last parameter to `NewDashboard`.
2. Update the `LogPane` struct literal to use the injected streamer:

```go
logView: LogPane{
    streamer:   streamer,
    buffer:     newLogBuffer(logBufCap),
    scrollRows: 0,
    paused:     false,
    state:      inspector.LogAreaConnecting,
},
```

**`internal/ui/flows/dashboard/project/project.go` (line ~34):**
```go
states.NewDashboard(ctx, project, th, themeName, poll, logBufCap, logs.New())
```
Add the `logs` import.

**`internal/ui/flows/dashboard/project/states/settings.go` (lines ~167 and ~172):**
```go
NewDashboard(s.ctx, s.project, s.liveTheme, s.themeNames[s.themeIdx], s.pollInterval, s.logBufferCap, logs.New())
// and
NewDashboard(s.ctx, s.project, s.origTheme, s.origThemeName, s.origPoll, s.origCap, logs.New())
```
Add the `logs` import to `settings.go`.

Run `go build ./...` — build must pass cleanly.

### Step 5 — Verify with tests

Run `go test ./...`. Fix any failures. Confirm that a test constructing
`NewDashboard` can pass `logs.NewFakeStreamer()` without compile errors.

---

## Out of Scope

- Settings message-drop bug (candidate 4) — deferred.
- Any changes to `NullLogStreamer` implementation.
- Adding new unit tests that exercise `FakeStreamer` — that is follow-on work.

---

## Post-Implementation

When all implementation steps are complete and the build passes:

1. Move this file to `docs/plans/archive/`
