# Refactor: Internal Services Layer + UI Structure Rules

Status: **Ready for implementation.** All design decisions resolved. No open questions.

---

## Context

`ogle` is a Bubble Tea TUI that watches a directory for Docker Compose files, lets the user select one, parses it, and displays a project dashboard. Module: `github.com/ma-tf/ogle`.

Current `internal/` layout:

```
internal/
  compose/
    scanner.go     # KnownFilenames(), ScanAll(dir) — returns []string
    parser.go      # Validate(path), Parse(path) — returns Project; owns Project, Service types and sentinel errors
  watcher/
    watcher.go     # fsnotify wrapper; calls compose.ScanAll + compose.KnownFilenames internally
  msgs/
    msgs.go        # Bubble Tea message types shared across UI and domain
  ui/
    flows/
      dashboard/
        dashboard.go
        project/
          project.go
          states/
            idle.go    # renders inline — correct, stateless projection of Project
            msgs.go
            state.go
      startup/
        startup.go
        states/
          msgs.go      # calls compose.ScanAll and compose.Parse directly in goroutines
          parsing.go   # errors.Is against compose.ErrReadComposeFile
          scanning.go
          selecting.go
          watching.go
    views/
      fileselect/
        fileselect.go
      watching/
        watching.go
    components/
      (empty — .gitkeep only)
```

---

## Decision Log

All decisions were reached through a design interview. Do not re-open them without a specific technical reason.

| # | Decision |
|---|---|
| 1 | `internal/services/` is a **domain services layer** — a semantic architectural boundary between business logic and UI. Not just a namespace. |
| 2 | Domain types (`Project`, `Service` struct, `ErrReadComposeFile`, `ErrParseComposeFile`) stay in `services/parser`. `msgs` and UI import from there. No separate `domain/` package. |
| 3 | Each package exposes a **`Service` struct** as its primary export. `service.go` is the primary file in each package. |
| 4 | `Service` structs are constructor-injected with a **`*slog.Logger`**. No empty structs; logger is the injection point for cross-cutting concerns. |
| 5 | All current exported free functions (`ScanAll`, `KnownFilenames`, `Validate`, `Parse`) are **privatised**. All callers must go through a `Service` instance. |
| 6 | All three services are **constructed at `cmd/root.go`** (the composition root) and injected inward. No service constructs another service itself. |
| 7 | `watcher.Service` **accepts `scanner.Service` as a dependency** — it retains its current behaviour of scanning after detecting a change and emitting `msgs.FileAvailabilityChanged`. |
| 8 | `internal/ui/components/` = reusable sub-models. Extracted when the **same interactive behaviour appears in 2+ views**. Not before. |
| 9 | `internal/ui/views/` = full-screen Bubble Tea models with their **own `Update` loop and local state**, owned by a flow state. |
| 10 | **Inline rendering in a flow state** is correct when the render is a pure stateless projection of data owned by that state. `states.Idle` rendering inline is correct and should not be moved. |

---

## Target Structure

```
internal/
  services/
    scanner/
      service.go   # scanner.Service{logger}; New(logger) Service; private: knownFilenames(), scanAll(dir)
    parser/
      service.go   # parser.Service{logger}; New(logger) Service; private: validate(path), parse(path)
                   # also owns: Project, Service (domain types), ErrReadComposeFile, ErrParseComposeFile
    watcher/
      service.go   # watcher.Service; New(dir, scanner scanner.Service, logger) (*Service, error)
                   # retains: Next() tea.Cmd, Close()
  msgs/
    msgs.go        # UNCHANGED — not a service; stays at internal/msgs/
  ui/              # UNCHANGED structure
    ...
```

---

## Implementation Steps

Work through these in order. Each step should leave the build passing before moving to the next.

### Step 1 — Create `internal/services/scanner/service.go`

- Define `type Service struct { logger *slog.Logger }`
- `func New(logger *slog.Logger) Service`
- Move logic from `compose/scanner.go` into private methods: `(s Service) knownFilenames() []string`, `(s Service) scanAll(dir string) []string`
- Expose public methods: `(s Service) KnownFilenames() []string`, `(s Service) ScanAll(dir string) []string`

Wait — decision 5 says free functions are privatised. Public surface is **only via `Service` methods**. The methods themselves can be exported (callers hold a `Service`); what's privatised is the package-level free functions.

Correct public API:
```go
func New(logger *slog.Logger) Service
func (s Service) ScanAll(dir string) []string
func (s Service) KnownFilenames() []string
```

### Step 2 — Create `internal/services/parser/service.go`

- Define `type Service struct { logger *slog.Logger }`
- `func New(logger *slog.Logger) Service`
- Move types here: `type Project struct`, `type ServiceDef struct` (or whatever the compose service type is named — check existing code)
- Move sentinel errors: `var ErrReadComposeFile`, `var ErrParseComposeFile`
- Expose public methods: `(s Service) Validate(path string) error`, `(s Service) Parse(path string) (Project, error)`
- Internal logic becomes private helpers

### Step 3 — Create `internal/services/watcher/service.go`

- Move `internal/watcher/watcher.go` logic here
- Update `New` signature: `func New(dir string, scanner scanner.Service, logger *slog.Logger) (*Service, error)`
- Replace internal calls to `compose.KnownFilenames()` / `compose.ScanAll()` with `scanner.KnownFilenames()` / `scanner.ScanAll()`
- Update import: `github.com/ma-tf/ogle/internal/services/scanner`
- `Next()` and `Close()` signatures unchanged

### Step 4 — Update `internal/msgs/msgs.go`

- Update import of `compose.Project` → `parser.Project`
- Import path: `github.com/ma-tf/ogle/internal/services/parser`

### Step 5 — Update `cmd/root.go` (composition root)

- Construct all three services here, after logger is initialised:
  ```go
  scannerSvc := scanner.New(logger)
  parserSvc  := parser.New(logger)
  watcherSvc, err := watcher.New(dir, scannerSvc, logger)
  ```
- Pass `parserSvc` and `scannerSvc` into `startup.New(...)` (or `dashboard.New(...)` — follow the existing injection chain)
- Pass `watcherSvc` into `dashboard.New(...)`

### Step 6 — Update `internal/ui/flows/startup/states/msgs.go`

- Replace direct calls to `compose.ScanAll` / `compose.Parse` with calls on the injected `scanner.Service` and `parser.Service`
- Update `ScanCmd` and `ParseCmd` signatures to accept the service instances

### Step 7 — Update `internal/ui/flows/startup/states/parsing.go`

- Replace `errors.Is(err, compose.ErrReadComposeFile)` with `errors.Is(err, parser.ErrReadComposeFile)`
- Update import to `github.com/ma-tf/ogle/internal/services/parser`

### Step 8 — Delete old packages

- Delete `internal/compose/` (both files)
- Delete `internal/watcher/` (watcher.go)
- Confirm no remaining imports of the old paths: `grep -r "internal/compose" .` and `grep -r "internal/watcher" .` should return nothing

### Step 9 — Verify

```
go build ./...
go test ./...
```

Fix any remaining import path errors. The build must pass clean.

---

## Import Path Reference

| Old | New |
|---|---|
| `github.com/ma-tf/ogle/internal/compose` | `github.com/ma-tf/ogle/internal/services/parser` (for types/parse) |
| `github.com/ma-tf/ogle/internal/compose` | `github.com/ma-tf/ogle/internal/services/scanner` (for scan) |
| `github.com/ma-tf/ogle/internal/watcher` | `github.com/ma-tf/ogle/internal/services/watcher` |

---

## What Is Explicitly Out of Scope

- Moving `msgs/` — it is not a service and stays at `internal/msgs/`
- Populating `internal/ui/components/` — no component meets the extraction threshold yet (no behaviour appears in 2+ views)
- Moving `states.Idle` rendering — it is correctly inline; do not extract it to `views/`
- Changing `Project` or `Service` type shapes — structural refactor only
- Writing new tests — out of scope for this refactor
