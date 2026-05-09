# ADR-0004: app/app.go is the root Bubble Tea model

**Status:** Superseded

> **Superseded by implementation reality.** The `internal/app/` package was never created. The root Bubble Tea model ended up in `internal/ui/flows/dashboard` (`dashboard.Model`), which owns the watcher lifecycle, drives the startup → project transition, and is instantiated directly by `cmd/root.go` via `dashboard.New(cfg, logger)`. The intent documented here — a thin `cmd/root.go`, a testable root model, and clear ownership of top-level state — is preserved; only the package path changed.

## Context

Early iterations had the Bubble Tea entry point in `internal/tui.go` as a file-level function. As the application grew, the root model needed to manage sub-models (startup flow and dashboard), dispatch messages between them, and own the watcher lifecycle.

## Decision

The root Bubble Tea model lives in `internal/app/app.go` as a proper package. `cmd/root.go` calls `app.Start()` and remains thin — it validates flags and delegates immediately.

## Consequences

- The `app` package has a clear, testable interface (`app.Start()`).
- `cmd/root.go` contains no Bubble Tea or TUI logic, only CLI flag parsing and pre-TUI validation (Explicit File mode hard exits happen here).
- The root model owns the two top-level states (`appStartup`, `appDashboard`) and dispatches `FileAvailabilityChanged` to the correct active sub-model.
- `internal/tui.go` is removed.
