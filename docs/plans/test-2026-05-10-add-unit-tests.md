# test: add unit tests

Status: **Ready for implementation.** All design decisions resolved. No open questions.

---

## Context

ogle is a pure-Go Bubble Tea TUI for observing Docker Compose projects. The codebase has zero test coverage. The services layer (`parser`, `scanner`, `watcher`) contains pure-ish logic that is immediately testable. The UI layer follows the Elm Architecture — each model exposes `Init/Update/View` — making `Update()` unit-testable by injecting `tea.Msg` values and asserting on returned state and commands.

---

## Decision Log

All decisions were reached through a design interview. Do not re-open them without a specific technical reason.

| # | Decision |
|---|---|
| 1 | Motivation is regression safety on the services layer AND full coverage including UI |
| 2 | UI tests unit-test `Update()` directly — no `teatest` or running program |
| 3 | Each startup state is tested in isolation; `startup.Model` is tested separately for cross-cutting behaviour (e.g. `WatcherError` transitions any state to watching-with-error) |
| 4 | `parser.Service` is tested with `t.TempDir()` and real fixture YAML files — no filesystem abstraction |
| 5 | `scanner.Service` is tested with `t.TempDir()` and real files — no filesystem abstraction |
| 6 | `watcher.Service` is tested via its observable contract only — `Next()` returns a cmd that delivers `msgs.FileAvailabilityChanged` reflecting real filesystem events in a temp dir; goroutine internals are not tested directly |
| 7 | Interface dependencies in UI state tests are satisfied with `testify/mock` mocks |
| 8 | Assertions use `testify/assert` and `testify/require` throughout |
| 9 | Test files are co-located with the package they test, using `package foo_test` (black-box) |
| 10 | Mocks live in a local `mocks/` subdirectory per package (e.g. `internal/services/parser/mocks/mock_parser.go`) |

---

## Implementation Steps

Each step must leave the build passing before the next begins.

1. **Add test dependencies** — `go get github.com/stretchr/testify` (assert, require, mock). Verify `go build ./...` still passes.

2. **Test `scanner.Service`** — create `internal/services/scanner/service_test.go` (`package scanner_test`). Use `t.TempDir()`. Cover: no files present, one file present, multiple files present (assert priority order), unknown filenames ignored.

3. **Test `parser.Service`** — create `internal/services/parser/service_test.go` (`package parser_test`). Use `t.TempDir()` with fixture YAML. Cover: valid file with `name` field, valid file without `name` (falls back to directory name), file does not exist (`ErrReadComposeFile`), invalid YAML (`ErrParseComposeFile`), file with multiple services.

4. **Test `watcher.Service` observable contract** — create `internal/services/watcher/service_test.go` (`package watcher_test`). Use `t.TempDir()`. Cover: file created → `FileAvailabilityChanged` delivered; file deleted → `FileAvailabilityChanged` delivered. Use a timeout context to avoid hanging tests.

5. **Generate mocks for `Parser` and `Scanner` interfaces** — create `internal/services/parser/mocks/mock_parser.go` and `internal/services/scanner/mocks/mock_scanner.go` using `testify/mock`. These are hand-written (no codegen tool required).

6. **Test `startup/states` in isolation** — create `service_test.go` per state file. Cover:
   - `Scanning`: `Init()` returns a `ScanCmd`
   - `Watching`: `FileAvailabilityChanged` with 0 files stays watching; with 1 file transitions to parsing; with 2+ files transitions to selecting
   - `Selecting`: `FileSelected` msg transitions to parsing; `FileAvailabilityChanged` with 0 files transitions to watching
   - `Parsing`: `parseDoneMsg` success emits `ProjectLoaded`; `parseDoneMsg` failure returns to appropriate state

7. **Test `startup.Model` cross-cutting behaviour** — create `internal/ui/flows/startup/startup_test.go`. Cover: `WatcherError` msg transitions any state to watching-with-error regardless of current state.

8. **Run full test suite** — `go test ./...` must pass with no failures. Fix any issues found.

---

## Target Structure

```
internal/
  services/
    parser/
      service.go
      service_test.go
      mocks/
        mock_parser.go
    scanner/
      service.go
      service_test.go
      mocks/
        mock_scanner.go
    watcher/
      service.go
      service_test.go
      null.go
  ui/
    flows/
      startup/
        startup.go
        startup_test.go
        states/
          scanning_test.go
          watching_test.go
          selecting_test.go
          parsing_test.go
```

---

## Out of Scope

- Tests for `cmd/` (Cobra bootstrap, config init)
- Tests for `internal/ui/views/` (fileselect, watching views)
- Tests for `internal/ui/flows/dashboard/` (dashboard, project flows)
- Tests for `internal/tools/docgen/`
- Integration tests using `teatest` or a running Bubble Tea program
- Code generation tooling (e.g. `mockery` CLI) — mocks are hand-written

---

## Post-Implementation

When all implementation steps are complete and the build passes:

1. Move this file to `docs/plans/archive/`
