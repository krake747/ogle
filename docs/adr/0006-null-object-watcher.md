# ADR-0006: Null Object pattern for Watcher; no nil sentinel

**Status:** Accepted

## Context

`watcher.New()` can fail (e.g. filesystem permissions, missing CWD). The original implementation returned `(*Watcher, error)` and stored `nil` as a meaningful sentinel in `app.model.w`. This forced three nil guards in `app.go` — at `Init`, at every `FileAvailabilityChanged` handler, and at shutdown.

## Decision

A `NullWatcher` is introduced (`internal/watcher/null.go`). It satisfies the `Watcher` interface but never delivers events — its `Next()` blocks until `Close()` is called. When `watcher.New()` fails, `app.go` assigns `watcher.NewNull()` instead of leaving `w` as `nil`. All nil guards are removed.

`watcher.New()` return type changes from `(*Watcher, error)` to `(Watcher, error)`.

## Consequences

- `app.model.w` is always non-nil; call sites need no guards.
- The error from `watcher.New()` is still surfaced to the UI (as a `WatcherError` message to the watching view), but the watcher field itself is always a valid object.
- `NullWatcher` lives in the same package as the interface it satisfies (`internal/watcher/null.go`), following Go convention for Null Object placement.
- Callers that previously handled `nil` watcher as "no watcher" now express that state through the `Watcher` interface itself — a more honest representation.
