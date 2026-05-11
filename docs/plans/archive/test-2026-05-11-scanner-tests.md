# test: scanner tests

Status: **Ready for implementation.** All design decisions resolved. No open questions.

---

## Context

`internal/services/scanner` provides File Discovery — the startup process of scanning the working directory for Compose Files. The package has two public methods: `KnownFilenames() []string` and `ScanAll(dir string) []string`. It has zero test coverage. The parser service (`internal/services/parser`) is the reference implementation for test conventions (ADR-0010).

---

## Decision Log

All decisions were reached through a design interview. Do not re-open them without a specific technical reason.

| # | Decision |
|---|---|
| 1 | Use `require.Empty` for the no-results case — nil vs empty slice is an implementation detail, not a behaviour contract |
| 2 | Include a mutation-isolation test case for `KnownFilenames` — mutate the returned slice, assert a second call returns the full list unaffected |
| 3 | Cover all four canonical filenames individually in `TestScanAll`, plus one multi-file case |
| 4 | Include a case where `dir` does not exist — assert empty result, not an error |
| 5 | Assert full absolute paths in `ScanAll` results |
| 6 | The multi-file case asserts results in canonical priority order; `scanAll` already iterates `knownFilenames()` in priority order so no production code change is required |
| 7 | Use a package-local `newTestLogger()` helper: writes to `bytes.Buffer`, strips `slog.TimeKey` via `ReplaceAttr` — copied from meta1v service test pattern |

---

## Implementation Steps

1. Create `internal/services/scanner/service_test.go` with `package scanner_test`.
2. Add `newTestLogger()` helper: `slog.NewTextHandler` writing to `bytes.Buffer`, `ReplaceAttr` stripping `slog.TimeKey`.
3. Implement `TestKnownFilenames` with two cases:
   - returns exactly `["compose.yml", "compose.yaml", "docker-compose.yml", "docker-compose.yaml"]`
   - mutating the returned slice does not affect a subsequent call
4. Implement `TestScanAll` with six cases, each using a `setup func(tc *testCase, dir string)` field and `t.TempDir()`:
   - `compose.yml` present — assert full absolute path
   - `compose.yaml` present — assert full absolute path
   - `docker-compose.yml` present — assert full absolute path
   - `docker-compose.yaml` present — assert full absolute path
   - all four present — assert full ordered slice of absolute paths
   - directory does not exist — assert empty
5. Run `go test ./internal/services/scanner/...` and confirm all cases pass.

---

## Target Structure

```
internal/services/scanner/
├── mocks/
│   └── mock_Scanner.go
├── service.go
└── service_test.go        ← new
```

---

## Out of Scope

- Adding logging calls to `service.go` (the unused `logger` field is flagged as a low-severity issue but is not addressed here)
- Tests for the `Scanner` interface or the mock
- Integration tests involving the watcher or startup flow

---

## Post-Implementation

When all implementation steps are complete and the build passes:

1. Move this file to `docs/plans/archive/`
