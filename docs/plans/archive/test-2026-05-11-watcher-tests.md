# test: watcher tests

Status: **Ready for implementation.** All design decisions resolved. No open questions.

---

## Context

`internal/services/watcher` contains two types: `Service` (real fsnotify-backed
watcher) and `nullWatcher` (Null Object returned when construction fails). The
package has no test coverage. All other service packages (`scanner`, `parser`)
follow an established convention (ADR-0010): black-box `package foo_test`,
`testify/require`, data-driven `testCase` structs, `t.Parallel()` everywhere,
`MockFoo` from `mockery`-generated mocks in a `mocks/` subdirectory.

---

## Decision Log

All decisions were reached through a design interview. Do not re-open them
without a specific technical reason.

| # | Decision |
|---|---|
| 1 | Use `MockScanner` from `internal/services/scanner/mocks/` — keeps watcher tests focused on watcher behaviour. |
| 2 | Use real filesystem events via `t.TempDir()` + real fsnotify — no fake fsnotify, no interface extraction. |
| 3 | Assert async delivery via a goroutine + channel + `select` with a 2s timeout. No `time.Sleep`. |
| 4 | Skip the three unreachable-via-filesystem branches (`fw.Events` closed, `fw.Errors` fired, `fw.Errors` closed). No production refactor to reach them. |
| 5 | Do not refactor `New()`. The constructor creating + starting the watcher in one call is intentional — misuse is impossible by design. |
| 6 | Do not mock `*fsnotify.Watcher`. The only branches it would cover are the ones explicitly skipped in decision 4. |

---

## Implementation Steps

Each step must leave the build passing before the next begins.

1. **Generate `MockScanner`** if not already present under
   `internal/services/watcher/mocks/`. Run:
   `go generate ./internal/services/watcher/...`
   (add a `//go:generate mockery --name=Scanner ...` directive to `service.go`
   if absent — check `internal/services/scanner/mocks/` for the existing
   invocation pattern first).

2. **Create `internal/services/watcher/service_test.go`** with
   `package watcher_test`. Implement the following test functions and subtests
   in order:

   ```
   TestNewNull
     └── Dir returns empty string
     └── Snapshot delivers empty FileAvailabilityChanged
     └── Next blocks until Close then returns nil
     └── Close is idempotent

   TestNew
     └── Dir returns the watched directory
     └── non-existent directory returns error and valid Watcher
     └── Close is idempotent

   TestSnapshot
     └── delivers current files without waiting for a filesystem event

   TestNext
     └── known file created triggers FileAvailabilityChanged
     └── unknown file created does not trigger an event
     └── known file deleted triggers FileAvailabilityChanged
     └── rapid events deliver only the latest snapshot
   ```

3. **Verify** `go test ./internal/services/watcher/...` passes with `-race`.

---

## Out of Scope

- `fw.Events` closed, `fw.Errors` fired, `fw.Errors` channel closed branches
  inside `run()` — unreachable via real filesystem operations; not worth a
  production abstraction.
- `LoggingWatcher` middleware (ADR-0009) — separate concern.
- Any refactor of `New()`, `run()`, or the `Watcher` interface.

---

## Pre-Implementation Findings

Recorded after codebase inspection. Do not re-open without a specific technical reason.

| # | Finding |
|---|---|
| A | **Step 1 is a no-op.** `MockScanner` already exists at `internal/services/scanner/mocks/mock_Scanner.go`. Decision 1 is confirmed: import it directly from `scanner/mocks`. Do not generate a duplicate under `watcher/mocks`. |
| B | **The project has no `//go:generate` directives.** Mock generation is driven entirely by `.mockery.yaml` + `mockery`. The Step 1 instruction to add a `//go:generate` directive to `service.go` does not apply. |

---

## Implementation — Complete Test File

**File to create:** `internal/services/watcher/service_test.go`

Step 1 of the original plan is a no-op — skip it entirely. `MockScanner` already
exists at `internal/services/scanner/mocks/mock_Scanner.go`; import it directly.
The project has no `//go:generate` directives and uses `.mockery.yaml` instead —
do not add one.

### Imports and helper

```go
package watcher_test

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ma-tf/ogle/internal/msgs"
	scannermocks "github.com/ma-tf/ogle/internal/services/scanner/mocks"
	"github.com/ma-tf/ogle/internal/services/watcher"
)

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
```

---

### TestNewNull

```go
func TestNewNull(t *testing.T) {
	t.Parallel()

	t.Run("Dir returns empty string", func(t *testing.T) {
		t.Parallel()
		w := watcher.NewNull()
		defer w.Close()
		require.Empty(t, w.Dir())
	})

	t.Run("Snapshot delivers empty FileAvailabilityChanged", func(t *testing.T) {
		t.Parallel()
		w := watcher.NewNull()
		defer w.Close()
		msg := w.Snapshot()()
		require.Equal(t, msgs.FileAvailabilityChanged{}, msg)
	})

	t.Run("Next blocks until Close then returns nil", func(t *testing.T) {
		t.Parallel()
		w := watcher.NewNull()

		got := make(chan any, 1)
		go func() { got <- w.Next()() }()

		require.NoError(t, w.Close())

		select {
		case msg := <-got:
			require.Nil(t, msg)
		case <-time.After(2 * time.Second):
			t.Fatal("Next did not return after Close")
		}
	})

	t.Run("Close is idempotent", func(t *testing.T) {
		t.Parallel()
		w := watcher.NewNull()
		require.NoError(t, w.Close())
		require.NoError(t, w.Close())
		require.NoError(t, w.Close())
	})
}
```

---

### TestNew

```go
func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("Dir returns the watched directory", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		sc := scannermocks.NewMockScanner(t)

		w, err := watcher.New(dir, sc, newTestLogger())
		require.NoError(t, err)
		defer w.Close()

		require.Equal(t, dir, w.Dir())
	})

	t.Run("non-existent directory returns error and valid Watcher", func(t *testing.T) {
		t.Parallel()
		sc := scannermocks.NewMockScanner(t)

		w, err := watcher.New("/nonexistent/path/that/cannot/exist", sc, newTestLogger())
		require.Error(t, err)
		require.NotNil(t, w)
		require.NoError(t, w.Close())
	})

	t.Run("Close is idempotent", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		sc := scannermocks.NewMockScanner(t)

		w, err := watcher.New(dir, sc, newTestLogger())
		require.NoError(t, err)

		require.NoError(t, w.Close())
		require.NoError(t, w.Close())
		require.NoError(t, w.Close())
	})
}
```

Mock setup notes:
- `Dir returns the watched directory` and `Close is idempotent`: no mock
  expectations needed. No filesystem events are generated, so `KnownFilenames`
  and `ScanAll` are never called. `NewMockScanner(t)` asserts no unexpected
  calls on cleanup — this passes correctly.
- `non-existent directory`: `New()` fails at `fw.Add(dir)` before `run()`
  starts. No scanner methods are called. No mock setup needed.

---

### TestSnapshot

```go
func TestSnapshot(t *testing.T) {
	t.Parallel()

	t.Run("delivers current files without waiting for a filesystem event", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		expected := []string{filepath.Join(dir, "compose.yml")}

		sc := scannermocks.NewMockScanner(t)
		sc.On("ScanAll", dir).Return(expected).Once()

		w, err := watcher.New(dir, sc, newTestLogger())
		require.NoError(t, err)
		defer w.Close()

		msg := w.Snapshot()()

		require.Equal(t, msgs.FileAvailabilityChanged{Files: expected}, msg)
	})
}
```

`Snapshot()` calls `sc.ScanAll(w.dir)` directly — no filesystem event involved.
`.Once()` is satisfied by `AssertExpectations` on cleanup.

---

### TestNext

#### known file created triggers FileAvailabilityChanged

```go
t.Run("known file created triggers FileAvailabilityChanged", func(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	composePath := filepath.Join(dir, "compose.yml")

	sc := scannermocks.NewMockScanner(t)
	sc.On("KnownFilenames").Return([]string{"compose.yml"})
	sc.On("ScanAll", dir).Return([]string{composePath})

	w, err := watcher.New(dir, sc, newTestLogger())
	require.NoError(t, err)
	defer w.Close()

	got := make(chan any, 1)
	go func() { got <- w.Next()() }()

	require.NoError(t, os.WriteFile(composePath, []byte{}, 0o600))

	select {
	case msg := <-got:
		require.Equal(t, msgs.FileAvailabilityChanged{Files: []string{composePath}}, msg)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for FileAvailabilityChanged")
	}
})
```

No `.Once()` on either expectation: a new file write may generate multiple
fsnotify events (CREATE + WRITE on Linux); the mock must match any number of
calls.

#### unknown file created does not trigger an event

```go
t.Run("unknown file created does not trigger an event", func(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	sc := scannermocks.NewMockScanner(t)
	sc.On("KnownFilenames").Return([]string{"compose.yml"})
	// ScanAll is intentionally NOT set up. If called, the mock panics —
	// acting as a hard assertion that unknown files never reach ScanAll.

	w, err := watcher.New(dir, sc, newTestLogger())
	require.NoError(t, err)

	require.NoError(t, os.WriteFile(filepath.Join(dir, "unknown.txt"), []byte{}, 0o600))

	// Close before calling Next so Next returns nil (not a FileAvailabilityChanged).
	require.NoError(t, w.Close())

	got := make(chan any, 1)
	go func() { got <- w.Next()() }()

	select {
	case msg := <-got:
		require.Nil(t, msg)
	case <-time.After(2 * time.Second):
		t.Fatal("Next did not return nil after Close")
	}
})
```

`run()` receives the unknown.txt event, calls `KnownFilenames()`, finds no
match, and `continue`s without touching `events`. `Close()` closes `done`.
`Next()` reads `select { events | done }` — `events` is empty, `done` is
closed → returns `nil`.

#### known file deleted triggers FileAvailabilityChanged

```go
t.Run("known file deleted triggers FileAvailabilityChanged", func(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	composePath := filepath.Join(dir, "compose.yml")

	// Pre-create so the watcher sees a REMOVE event.
	require.NoError(t, os.WriteFile(composePath, []byte{}, 0o600))

	sc := scannermocks.NewMockScanner(t)
	sc.On("KnownFilenames").Return([]string{"compose.yml"})
	sc.On("ScanAll", dir).Return([]string{}) // file is gone; scanner returns empty

	w, err := watcher.New(dir, sc, newTestLogger())
	require.NoError(t, err)
	defer w.Close()

	got := make(chan any, 1)
	go func() { got <- w.Next()() }()

	require.NoError(t, os.Remove(composePath))

	select {
	case msg := <-got:
		fac, ok := msg.(msgs.FileAvailabilityChanged)
		require.True(t, ok)
		require.Empty(t, fac.Files)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for FileAvailabilityChanged")
	}
})
```

`require.Empty` (not `require.Equal`) because `ScanAll` returning `[]string{}`
produces a non-nil empty slice. `require.Empty` accepts both nil and empty,
avoiding a nil-vs-empty mismatch.

#### rapid events deliver only the latest snapshot

This test verifies the drain-before-send logic in `run()`: when events arrive
faster than the consumer calls `Next()`, the buffered `events` channel (capacity
1) always holds only the latest snapshot — stale ones are discarded.

**Technique:** Gate `ScanAll` with a `release` channel. While `run()` is blocked
inside event 1's `ScanAll`, write event 2 to the filesystem. Closing `release`
unblocks event 1, which sends `snapshot_1` to `events`. `run()` then immediately
processes event 2, drains `snapshot_1`, and sends `snapshot_2`. `Next()` returns
`snapshot_2` only.

The file is pre-created before the watcher starts so that subsequent writes
generate only WRITE events (not CREATE+WRITE), giving exactly one fsnotify event
per `os.WriteFile` call. If extra events do occur (OS-dependent), all calls with
`callNum >= 2` return `"file_v2"`, so the assertion holds regardless.

```go
t.Run("rapid events deliver only the latest snapshot", func(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	composePath := filepath.Join(dir, "compose.yml")

	// Pre-create the file before the watcher starts so subsequent writes
	// generate only WRITE events (not CREATE+WRITE).
	require.NoError(t, os.WriteFile(composePath, []byte("initial"), 0o600))

	sc := scannermocks.NewMockScanner(t)
	sc.On("KnownFilenames").Return([]string{"compose.yml"})

	// release: closed by test to unblock all pending ScanAll calls.
	// ready:   buffered; ScanAll signals before blocking so the test knows
	//          run() has reached ScanAll for that event.
	release := make(chan struct{})
	ready := make(chan struct{}, 10)

	var n atomic.Int32
	sc.On("ScanAll", dir).Return(func(string) []string {
		callNum := int(n.Add(1))
		ready <- struct{}{} // signal: ScanAll called for this event
		<-release           // block until test closes release
		if callNum == 1 {
			return []string{"file_v1"}
		}
		return []string{"file_v2"}
	})

	w, err := watcher.New(dir, sc, newTestLogger())
	require.NoError(t, err)
	defer w.Close()

	// Event 1: run() calls ScanAll and blocks.
	require.NoError(t, os.WriteFile(composePath, []byte("v1"), 0o600))
	select {
	case <-ready:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for first ScanAll call")
	}

	// Event 2: written while run() is blocked. fsnotify queues this in
	// fw.Events; run() will pick it up after event 1 completes.
	require.NoError(t, os.WriteFile(composePath, []byte("v2"), 0o600))

	// Unblock: event 1 ScanAll returns "file_v1" → run() sends snapshot_1
	// to events → loops → processes event 2 → ScanAll returns "file_v2"
	// → drains snapshot_1 → sends snapshot_2.
	close(release)

	// Wait for event 2's ScanAll to start (confirms drain has occurred or
	// will occur imminently). Next() blocks until events has a value, so
	// it safely handles the case where snapshot_2 hasn't landed yet.
	select {
	case <-ready:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for second ScanAll call")
	}

	got := make(chan any, 1)
	go func() { got <- w.Next()() }()

	select {
	case msg := <-got:
		fac, ok := msg.(msgs.FileAvailabilityChanged)
		require.True(t, ok)
		require.Equal(t, []string{"file_v2"}, fac.Files)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for message from Next()")
	}
})
```

---

### Verification command

```
go test ./internal/services/watcher/... -race -count=1
```

---

## Post-Implementation

When all implementation steps are complete and the build passes:

1. Move this file to `docs/plans/archive/`
