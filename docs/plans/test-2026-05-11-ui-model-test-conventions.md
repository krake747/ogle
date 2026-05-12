# test: UI model test conventions

Status: **Ready for implementation.** All design decisions resolved. No open questions.

---

## Context

Service-layer tests follow ADR-0010: pure functions, full struct equality,
`testify/require`, data-driven `testCase` structs, `t.Parallel()` everywhere.
Bubble Tea models — views, startup states, the startup flow, and the dashboard
orchestrator — are state machines (`msg → (model, cmd)`), not pure functions.
They require three distinct assertion shapes:

1. **State transitions** — send a message, assert the returned model's type
   and fields.
2. **View output** — assert rendered strings contain expected substrings.
3. **Command emissions** — call the returned `tea.Cmd`, assert the resulting
   `tea.Msg`.

None of these patterns exist in ADR-0010. A design interview was conducted to
resolve all outstanding decisions. ADR-0011 records the resulting conventions.
This plan implements them across the full UI stack (excluding
`flows/dashboard/project`, which is deferred).

---

## Decision Log

All decisions were reached through a design interview. Do not re-open them
without a specific technical reason.

| # | Decision |
|---|---|
| 1 | Assert all three: state transitions, `View()` substrings, and command emissions. |
| 2 | View assertions: `require.Contains` on rendered strings. No snapshot/golden-file testing. |
| 3 | `wrapLine` exported as `WrapLine`; tested directly as a pure function. Snapshot testing rejected: ADR-0010 prohibits `testdata/`, and themes (queued plan) will add ANSI codes to views. |
| 4 | Internal message types exported for testability: `ScanDoneMsg{Valid []string}`, `ParseDoneMsg{Project *domain.Project, Err error}`, `WatcherReadyMsg{W svcwatcher.Watcher}`. |
| 5 | All four layers tested: `views/watching`, `views/fileselect`, `flows/startup/states`, `flows/startup`, `flows/dashboard`. |
| 6 | `flows/dashboard/project/states/Dashboard` deferred — lipgloss ANSI complicates view assertions and the project flow is under active development. |
| 7 | `dashboard.New()` refactored to accept `w svcwatcher.Watcher, watcherErr error`; watcher construction moves to `cmd/root.go`. Enables `MockWatcher` injection in tests without real filesystem. |
| 8 | `Watcher` interface added to `.mockery.yaml`; `MockWatcher` generated at `internal/services/watcher/mocks/`. |
| 9 | New ADR-0011 for UI model test conventions. ADR-0010 is the baseline; ADR-0011 documents only the deltas. |

---

## Implementation Steps

Each step must leave the build passing before the next begins.

### Step 1 — Write ADR-0011

Create `docs/adr/0011-ui-model-test-conventions.md` documenting the conventions
established by this plan. Content:

- **Status:** Proposed
- **Context:** ADR-0010 covers service-layer tests only. Bubble Tea models are
  state machines requiring different assertion patterns.
- **Inherits from ADR-0010:** `package foo_test`, `testify/require`, full struct
  equality, `t.Parallel()` everywhere, `testCase` struct per function, `setup`
  callback, `t.TempDir()`, mockery-generated mocks.
- **Delta — state machine driving:** call `Init()` or `Update(msg)`, receive
  `(model, cmd)`. To assert on a command, call `cmd()` to obtain the `tea.Msg`
  and assert on it directly. For multi-step flows, feed the message back into
  `Update()` to drive the next transition.
- **Delta — view assertions:** `require.Contains(t, m.View(), "expected text")`.
  No snapshot/golden files. No exact-string equality on full rendered output.
- **Delta — exported-for-testability types:** internal message types and
  unexported functions may be exported when they are the natural assertion point
  for a test. Each such export must be documented with a `// Exported for
  testing.` comment on the type or function.
- **Delta — constructor injection:** when a model constructs infrastructure
  internally (e.g., the watcher), refactor to accept the dependency as a
  parameter so tests can inject mocks.
- **Consequences:** minor API surface increase in UI packages; exported types
  are still internal to the `ui` subtree in practice.

### Step 2 — Export types for testability

**`internal/ui/views/watching/watching.go`**

Rename `wrapLine` → `WrapLine`. Add `// Exported for testing.` comment.
Update the single internal call site (`View()`).

**`internal/ui/flows/startup/states/msgs.go`**

Rename and export fields:

```
scanDoneMsg{valid []string}    →  ScanDoneMsg{Valid []string}
parseDoneMsg{project, err}     →  ParseDoneMsg{Project *domain.Project, Err error}
```

Add `// Exported for testing.` comments on both types.

Update all internal usages:
- `scanning.go`: construction of `scanDoneMsg` → `ScanDoneMsg`, field `valid` → `Valid`
- `msgs.go` (`ScanCmd`, `ParseCmd`): return literal field names
- `parsing.go`: type assertion and field access on `parseDoneMsg` → `ParseDoneMsg`

**`internal/ui/flows/dashboard/dashboard.go`**

Rename and export field:

```
watcherReadyMsg{w svcwatcher.Watcher}  →  WatcherReadyMsg{W svcwatcher.Watcher}
```

Add `// Exported for testing.` comment. Update the type assertion in `Update`
and the construction in `retryWatcherCmd`.

Add a `Watcher() svcwatcher.Watcher` accessor to `Model` for asserting watcher
replacement in tests:

```go
// Watcher returns the active Watcher. Exported for testing.
func (m Model) Watcher() svcwatcher.Watcher { return m.w }
```

Run `go build ./...` to confirm no broken references.

### Step 3 — Add `MockWatcher` to `.mockery.yaml` and generate

Add to `.mockery.yaml` under `packages:`:

```yaml
github.com/ma-tf/ogle/internal/services/watcher:
  interfaces:
    Watcher:
      config:
        dir: "internal/services/watcher/mocks"
        outpkg: mocks
```

Run:

```
mockery
```

Verify `internal/services/watcher/mocks/mock_Watcher.go` is generated. Do not
edit it manually.

### Step 4 — Refactor `dashboard.New()` for watcher injection

**`internal/ui/flows/dashboard/dashboard.go`**

Change the signature:

```go
// Before
func New(cfg config.Config, logger *slog.Logger, sc scanner.Scanner, p parser.Parser) Model

// After
func New(cfg config.Config, logger *slog.Logger, sc scanner.Scanner, p parser.Parser, w svcwatcher.Watcher, watcherErr error) Model
```

Remove the `svcwatcher.New(...)` call from `New()`. Use the injected `w` and
`watcherErr` directly.

**`cmd/root.go`**

Construct the watcher before calling `dashboard.New()`:

```go
w, watcherErr := svcwatcher.New(dir, sc, logger)
m := dashboard.New(cfg, logger, sc, p, w, watcherErr)
```

Run `go build ./...` to confirm the build passes.

### Step 5 — Write `internal/ui/views/watching/watching_test.go`

Package: `package watching_test`

```
TestWrapLine
  └── empty string returns empty
  └── string shorter than width returns unchanged
  └── string exactly at width returns unchanged
  └── breaks at last space before width
  └── breaks mid-word when no space available
  └── multi-byte runes: em dash does not split mid-character
  └── leading spaces trimmed from continuation line

TestWatchingView   (state/view tests on watching.Model)
  └── New: View contains working directory and "ctrl+c quit"
  └── NewDisconnected: View contains target filename and "Disconnected"
  └── SetNotice: View contains "notice:" prefix and message
  └── ClearNotice: View no longer contains notice text
  └── SetError: View contains "Error:" and "r retry"
  └── ClearError: View no longer contains error text
  └── SetParsing true: View contains "Parsing..."
  └── SetParsing false: View does not contain "Parsing..."
  └── Update key "r" in stateIdle: no cmd returned
  └── Update key "r" in stateError: cmd returns msgs.RetryWatcher{}
  └── Update WindowSizeMsg: subsequent View uses new dimensions
```

Use fixed dimensions (e.g. `width: 80, height: 24`) throughout. For view
assertions use `require.Contains`. For cmd assertions call `cmd()` and
type-assert the result.

### Step 6 — Write `internal/ui/views/fileselect/fileselect_test.go`

Package: `package fileselect_test`

```
TestFileselectModel
  └── New: model constructed with provided file paths
  └── SetFiles: replaces file list
  └── SetFiles: clears parse error when errored file is no longer present
  └── SetFiles: preserves parse error when errored file is still present
  └── SetError: View contains error message for the named file
  └── SetParsing true: View contains "Parsing..."
  └── SetParsing false: "Parsing..." no longer present
  └── Update "enter" key: cmd returns msgs.FileSelected for selected item
```

Note: do not test `bubbles/list` navigation internals — only ogle-specific
logic. For the `"enter"` key test, construct a model with one file and send
`tea.KeyPressMsg` with `Key: tea.KeyEnter`.

### Step 7 — Write `internal/ui/flows/startup/states/states_test.go`

Package: `package states_test`

Use `MockScanner` (`internal/services/scanner/mocks/`) and `MockParser`
(`internal/services/parser/mocks/`).

```
TestScanning
  └── Init returns a cmd that produces ScanDoneMsg
  └── ScanDoneMsg with 0 valid files → model is states.Watching
  └── ScanDoneMsg with 1 valid file → model is states.Parsing, cmd fires ParseCmd
  └── ScanDoneMsg with 2+ valid files → model is states.Selecting
  └── unrelated msg dropped: model unchanged, nil cmd

TestWatching
  └── Init returns nil
  └── FileAvailabilityChanged 0 valid → model is states.Watching
  └── FileAvailabilityChanged 1 valid → model is states.Parsing
  └── FileAvailabilityChanged 2+ valid → model is states.Selecting
  └── WindowSizeMsg: subsequent handler uses updated dimensions
  └── unrelated msg delegated to inner watching.Model

TestSelecting
  └── Init returns nil
  └── FileAvailabilityChanged 0 valid → model is states.Watching
  └── FileAvailabilityChanged 1 valid → model is states.Parsing
  └── FileAvailabilityChanged 2+ valid → model is states.Selecting with updated files
  └── FileSelected → model is states.Parsing for the selected path
  └── unrelated msg delegated to fileselect.Model

TestParsing
  └── Init fires parse cmd
  └── ParseDoneMsg success → cmd returns msgs.ProjectLoaded with correct Project
  └── ParseDoneMsg ErrReadComposeFile → returns to display state, nil cmd
  └── ParseDoneMsg other error, Watching display → Watching with notice containing filename
  └── ParseDoneMsg other error, Selecting display → Selecting with error set on view
  └── unrelated msg forwarded to display state
```

For state-type assertions use type switches: `_, ok := next.(states.Watching)`.
For cmd assertions call `cmd()` and assert on the returned `tea.Msg`.
`MockParser.Validate` and `MockParser.Parse` control what `ScanCmd`/`ParseCmd`
produce.

### Step 8 — Write `internal/ui/flows/startup/startup_test.go`

Package: `package startup_test`

```
TestStartupNew
  └── watcherErr non-nil → current state renders watcher error (View contains error text)
  └── cfg.ProjectFile non-empty → Init cmd produces ParseDoneMsg (drives parse)
  └── default → Init cmd produces ScanDoneMsg (drives scan)

TestStartupUpdate
  └── msgs.WatcherError intercepted: next model's View contains the error regardless of current state
  └── tea.WindowSizeMsg propagated: width and height stored, passed to subsequent states
  └── unrelated msg delegated to current state
```

### Step 9 — Write `internal/ui/flows/dashboard/dashboard_test.go`

Package: `package dashboard_test`

Use `MockWatcher` (`internal/services/watcher/mocks/`), `MockScanner`, and
`MockParser`. Construct via `dashboard.New(cfg, logger, mockScanner, mockParser,
mockWatcher, nil)`.

```
TestDashboardUpdate
  └── msgs.ProjectLoaded → m.current is project.Model (assert via View() no longer showing startup content)
  └── msgs.RetryWatcher → non-nil cmd returned
  └── WatcherReadyMsg → m.Watcher() returns the new watcher; cmd returned is non-nil
  └── msgs.FileAvailabilityChanged → MockWatcher.Next() called (re-subscribe); msg forwarded to current
  └── tea.WindowSizeMsg → subsequent View uses new dimensions
  └── tea.KeyPressMsg "ctrl+c" → cmd returns tea.Quit

TestDashboardInit
  └── Init: MockWatcher.Next() called; m.current.Init() result batched
```

For `WatcherReadyMsg`, set up `MockWatcher.Next` and `MockWatcher.Snapshot` to
return safe no-op commands (`func() tea.Msg { return nil }`), send the message,
call `cmd()` to execute the batch, then assert `m.Watcher() == newMockWatcher`.

### Step 10 — Verify

Run:

```
go test ./internal/ui/... -race
```

All tests must pass. Fix any failures before marking the plan complete.

---

## Out of Scope

- `flows/dashboard/project/states/Dashboard` — lipgloss ANSI codes complicate
  view assertions; project flow is under active development (themes plan
  queued). Revisit once the project flow has more states and themes have landed.
- `components/servicelist` — tested indirectly through project flow states;
  direct tests deferred alongside project flow.
- Snapshot/golden-file testing — rejected (see Decision 3).
- White-box tests (`package foo`) — ADR-0010 and ADR-0011 both require black-box
  style only.
- The `fileselect` mouse-release bug (emits `msgs.FileSelected` on any
  `tea.MouseReleaseMsg`) — flagged separately; fix is out of scope for this plan.

---

## Post-Implementation

When all implementation steps are complete and the build passes:

1. Move this file to `docs/plans/archive/`
