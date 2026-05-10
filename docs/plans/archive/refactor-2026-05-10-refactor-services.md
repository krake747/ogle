# refactor: refactor-services

Status: **Ready for implementation.** All design decisions resolved. No open questions.

---

## Context

ogle is a terminal UI for observing Docker Compose projects. The startup flow drives
File Discovery — scanning a directory for Compose Files, validating candidates, and
parsing the chosen file into a Project.

Currently `scanner.Service` and `parser.Service` are concrete structs passed directly
into every module in the startup flow (states, dashboard orchestrator, watcher). There
are no `Scanner` or `Parser` interfaces, so the startup state machine cannot be tested
without a real filesystem and real Compose File parsing.

Additionally, `msgs.ProjectLoaded` holds `*parser.Project`, forcing the `msgs` package
to import `parser`. ADR-0001 intends `msgs` to be a dependency-free message bus; this
import violates the intended layering and creates a transitive dependency on the parser
implementation for every `msgs` importer.

This refactor introduces `internal/domain` as the home for shared types (`Project`,
`ServiceDef`) and the `Scanner` / `Parser` interfaces. All call sites switch from
concrete types to interfaces. The concrete `scanner.Service` and `parser.Service` types
continue to be constructed at the entry point (`cmd/root.go`) and satisfy their
respective interfaces implicitly, verified by compile-time assertions.

---

## Decision Log

All decisions were reached through a design interview. Do not re-open them without a specific technical reason.

| # | Decision |
|---|---|
| 1 | `Scanner` and `Parser` interfaces live in `internal/domain`, not in the `scanner` or `parser` packages. Mirrors the pattern already used by `Watcher` (interface in the service package) but avoids the concrete-type import problem by placing shared types and interfaces in a neutral package. |
| 2 | `Project` and `ServiceDef` move from `internal/services/parser` to `internal/domain`. This is the only clean resolution of the `msgs → parser` layering leak. |
| 3 | `internal/msgs` imports `internal/domain` instead of `internal/services/parser`. `ProjectLoaded.Project` becomes `*domain.Project`. |
| 4 | `internal/services/watcher` accepts `domain.Scanner` instead of the concrete `scanner.Service`. Included in this refactor. |
| 5 | Sentinel errors (`ErrReadComposeFile`, `ErrParseComposeFile`) stay in `internal/services/parser` — they are implementation detail, not domain types. Files that need them retain a direct `parser` import only for those errors. |
| 6 | Compile-time interface assertions (`var _ domain.Parser = Service{}`, `var _ domain.Scanner = Service{}`) are added to `parser` and `scanner` packages respectively. |
| 7 | No behaviour changes. Pure structural refactor. |

---

## Implementation Steps

Each step must leave `go build ./...` passing before the next begins.

**Step 1 — Create `internal/domain/domain.go`**

New file. No imports. Contains:
- `Project` struct (`Name string`, `File string`, `Services []ServiceDef`)
- `ServiceDef` struct (`Name string`, `Image string`, `ContainerName string`)
- `Scanner` interface (`KnownFilenames() []string`, `ScanAll(dir string) []string`)
- `Parser` interface (`Validate(path string) error`, `Parse(path string) (*Project, error)`)

**Step 2 — Update `internal/services/parser`**

- Remove `Project` and `ServiceDef` struct definitions.
- Import `internal/domain`.
- `Parse` returns `*domain.Project`.
- Update `readAndUnmarshal` return assembly to use `domain.Project` / `domain.ServiceDef`.
- Add: `var _ domain.Parser = Service{}`

**Step 3 — Update `internal/services/scanner`**

- Import `internal/domain`.
- Add: `var _ domain.Scanner = Service{}`
- No method signature changes needed.

**Step 4 — Update `internal/services/watcher`**

- `New` signature: `func New(dir string, sc domain.Scanner, logger *slog.Logger) (Watcher, error)`
- Remove import of `internal/services/scanner`; add `internal/domain`.
- Update all internal references from `scannerSvc` / `scanner.Service` to `domain.Scanner`.

**Step 5 — Update `internal/msgs`**

- Remove import of `internal/services/parser`.
- Add import of `internal/domain`.
- `ProjectLoaded.Project` field type: `*domain.Project`.

**Step 6 — Update call sites**

Update the following files to use `domain.Scanner` / `domain.Parser` / `*domain.Project`
in place of the concrete types. Construction calls (`scanner.New`, `parser.New`) are
unchanged — they return concrete types that satisfy the interfaces.

| File | Changes |
|---|---|
| `cmd/root.go` | Local vars typed as `domain.Scanner` / `domain.Parser`. `validateProjectFile` accepts `domain.Parser`. Drop `scanner` and `parser` package imports; add `domain`. Keep `scanner.New` / `parser.New` construction calls. |
| `internal/ui/flows/dashboard/dashboard.go` | Fields `scanner` / `parser` typed as `domain.Scanner` / `domain.Parser`. `retryWatcherCmd` accepts `domain.Scanner`. |
| `internal/ui/flows/startup/startup.go` | Fields and all constructor params typed as `domain.Scanner` / `domain.Parser`. |
| `internal/ui/flows/startup/states/handler.go` | `fileHandler` fields typed as `domain.Scanner` / `domain.Parser`. Retain `import "parser"` only for `parser.ErrReadComposeFile` sentinel in `newWatching`. |
| `internal/ui/flows/startup/states/msgs.go` | `ScanCmd` / `ParseCmd` / `validateFiles` params typed as `domain.Scanner` / `domain.Parser`. `parseDoneMsg.project` typed as `*domain.Project`. |
| `internal/ui/flows/startup/states/scanning.go` | Constructor params typed as `domain.Scanner` / `domain.Parser`. |
| `internal/ui/flows/startup/states/watching.go` | Constructor params typed as `domain.Scanner` / `domain.Parser`. |
| `internal/ui/flows/startup/states/parsing.go` | Retain `import "parser"` only for `parser.ErrReadComposeFile` sentinel. No `*parser.Project` references remain after `msgs.go` update. |
| `internal/ui/flows/dashboard/project/project.go` | `New` accepts `*domain.Project`. |
| `internal/ui/flows/dashboard/project/states/idle.go` | Field and constructor param typed as `*domain.Project`. |

**Step 7 — Verify**

```
go build ./...
go test ./...
```

No failures expected. No behaviour changes.

---

## Target Structure

```
internal/
  domain/
    domain.go          ← NEW: Project, ServiceDef, Scanner, Parser
  services/
    parser/
      service.go       ← removes Project/ServiceDef; imports domain; adds assertion
    scanner/
      service.go       ← imports domain; adds assertion
    watcher/
      service.go       ← accepts domain.Scanner instead of scanner.Service
  msgs/
    msgs.go            ← imports domain instead of parser
  ui/
    flows/
      dashboard/
        dashboard.go   ← domain.Scanner / domain.Parser fields
        project/
          project.go   ← *domain.Project
          states/
            idle.go    ← *domain.Project
      startup/
        startup.go     ← domain.Scanner / domain.Parser
        states/
          handler.go   ← domain.Scanner / domain.Parser; parser import for errors only
          msgs.go      ← domain.Scanner / domain.Parser / *domain.Project
          parsing.go   ← parser import for errors only
          scanning.go  ← domain.Scanner / domain.Parser
          watching.go  ← domain.Scanner / domain.Parser
cmd/
  root.go              ← domain.Scanner / domain.Parser locals
```

---

## Import Path Reference

| Old | New |
|---|---|
| `github.com/ma-tf/ogle/internal/services/parser` (for `Project`) | `github.com/ma-tf/ogle/internal/domain` |
| `github.com/ma-tf/ogle/internal/services/parser` (for `ServiceDef`) | `github.com/ma-tf/ogle/internal/domain` |
| `github.com/ma-tf/ogle/internal/services/scanner` (as concrete type) | `github.com/ma-tf/ogle/internal/domain` (as interface) |
| `github.com/ma-tf/ogle/internal/services/parser` (as concrete type) | `github.com/ma-tf/ogle/internal/domain` (as interface) |
| `github.com/ma-tf/ogle/internal/services/parser` (for sentinel errors only) | retained in `handler.go` and `parsing.go` |

---

## Out of Scope

- Candidate 3: `Parsing` state coupling to sibling states via type switch (separate refactor).
- Candidate 4: `cmd/root.go` package-level globals / non-reentrant command (separate refactor).
- Candidate 5: Watcher goroutine start before Bubble Tea runtime (separate refactor).
- Writing tests that exploit the new seams — this plan only creates the seams.
- Any behaviour changes to File Discovery, parsing, or watching logic.

---

## Post-Implementation

When all implementation steps are complete and the build passes:

1. Move this file to `docs/plans/archive/`
